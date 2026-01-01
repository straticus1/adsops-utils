#!/usr/bin/env python3
"""
KVM/QEMU VM to Oracle Cloud Infrastructure (OCI) Migration Tool

Converts KVM VMs (QCOW2, RAW, VMDK) to OCI custom images.

Requirements:
    pip install oci pyyaml libvirt-python

Usage:
    ./kvm2oci.py list                           # List VMs on KVM host
    ./kvm2oci.py export --vm myvm               # Export VM disk
    ./kvm2oci.py convert disk.raw               # Convert to OCI format
    ./kvm2oci.py upload disk.qcow2              # Upload to OCI
    ./kvm2oci.py migrate --vm myvm --name "VM"  # Full migration
"""

import argparse
import hashlib
import json
import logging
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Optional
import xml.etree.ElementTree as ET

try:
    import oci
    import yaml
except ImportError:
    print("Missing dependencies. Install with:")
    print("  pip install oci pyyaml")
    sys.exit(1)

try:
    import libvirt
    HAS_LIBVIRT = True
except ImportError:
    HAS_LIBVIRT = False

# ============================================
# CONFIGURATION
# ============================================

DEFAULT_CONFIG = {
    "kvm": {
        "uri": "qemu:///system",  # Or qemu+ssh://user@host/system
        "remote_host": "",  # For remote connections
    },
    "oci": {
        "config_file": "~/.oci/config",
        "profile": "DEFAULT",
        "compartment_id": "",
        "bucket_name": "vm-migrations",
        "namespace": "",
    },
    "conversion": {
        "output_format": "qcow2",
        "compress": True,
        "work_dir": "/tmp/kvm2oci",
    },
    "image": {
        "launch_mode": "PARAVIRTUALIZED",
        "operating_system": "Custom",
    },
}

CONFIG_PATHS = [
    Path("./kvm2oci.yaml"),
    Path("~/.config/kvm2oci/config.yaml").expanduser(),
]

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger(__name__)


# ============================================
# UTILITIES
# ============================================


def run_cmd(cmd: list, check: bool = True) -> subprocess.CompletedProcess:
    """Run a command and return result"""
    log.debug(f"Running: {' '.join(cmd)}")
    result = subprocess.run(cmd, capture_output=True, text=True, check=False)
    if check and result.returncode != 0:
        log.error(f"Command failed: {result.stderr}")
        raise RuntimeError(f"Command failed: {' '.join(cmd)}")
    return result


def check_dependencies():
    """Check for required tools"""
    required = ["qemu-img"]
    missing = []
    for tool in required:
        if not shutil.which(tool):
            missing.append(tool)
    if missing:
        log.error(f"Missing required tools: {', '.join(missing)}")
        sys.exit(1)


def format_size(size: int) -> str:
    """Format bytes to human readable"""
    for unit in ["B", "KB", "MB", "GB", "TB"]:
        if size < 1024:
            return f"{size:.1f} {unit}"
        size /= 1024
    return f"{size:.1f} PB"


# ============================================
# KVM OPERATIONS
# ============================================


@dataclass
class VMDisk:
    path: str
    format: str
    device: str
    size_bytes: int = 0


@dataclass
class VMInfo:
    name: str
    uuid: str
    state: str
    memory_mb: int
    vcpus: int
    disks: list


class KVMHost:
    """Interact with KVM/libvirt host"""

    def __init__(self, config: dict):
        self.config = config.get("kvm", {})
        self.uri = self.config.get("uri", "qemu:///system")
        self.conn = None

    def connect(self):
        """Connect to libvirt"""
        if not HAS_LIBVIRT:
            raise ImportError("libvirt-python not installed. Install with: pip install libvirt-python")

        log.info(f"Connecting to: {self.uri}")
        self.conn = libvirt.open(self.uri)
        if self.conn is None:
            raise RuntimeError(f"Failed to connect to {self.uri}")
        log.info("Connected to libvirt")

    def disconnect(self):
        """Close libvirt connection"""
        if self.conn:
            self.conn.close()

    def list_vms(self) -> list[VMInfo]:
        """List all VMs"""
        if not self.conn:
            self.connect()

        vms = []
        for domain in self.conn.listAllDomains():
            state, _ = domain.state()
            state_names = {
                libvirt.VIR_DOMAIN_RUNNING: "running",
                libvirt.VIR_DOMAIN_PAUSED: "paused",
                libvirt.VIR_DOMAIN_SHUTDOWN: "shutdown",
                libvirt.VIR_DOMAIN_SHUTOFF: "off",
                libvirt.VIR_DOMAIN_CRASHED: "crashed",
            }

            # Parse XML for disk info
            xml = domain.XMLDesc()
            disks = self._parse_disks(xml)

            vms.append(VMInfo(
                name=domain.name(),
                uuid=domain.UUIDString(),
                state=state_names.get(state, "unknown"),
                memory_mb=domain.maxMemory() // 1024,
                vcpus=domain.maxVcpus(),
                disks=disks,
            ))

        return vms

    def _parse_disks(self, xml: str) -> list[VMDisk]:
        """Parse disk info from domain XML"""
        disks = []
        root = ET.fromstring(xml)

        for disk in root.findall(".//disk"):
            if disk.get("device") != "disk":
                continue

            source = disk.find("source")
            driver = disk.find("driver")
            target = disk.find("target")

            if source is not None:
                path = source.get("file") or source.get("dev") or ""
                fmt = driver.get("type", "raw") if driver is not None else "raw"
                dev = target.get("dev", "") if target is not None else ""

                # Get size
                size = 0
                if path and os.path.exists(path):
                    size = os.path.getsize(path)

                disks.append(VMDisk(path=path, format=fmt, device=dev, size_bytes=size))

        return disks

    def get_vm(self, name: str) -> Optional[VMInfo]:
        """Get VM by name"""
        for vm in self.list_vms():
            if vm.name == name:
                return vm
        return None

    def shutdown_vm(self, name: str, timeout: int = 120) -> bool:
        """Gracefully shutdown VM"""
        if not self.conn:
            self.connect()

        try:
            domain = self.conn.lookupByName(name)
            if domain.isActive():
                log.info(f"Shutting down {name}...")
                domain.shutdown()

                # Wait for shutdown
                for _ in range(timeout):
                    if not domain.isActive():
                        log.info("VM stopped")
                        return True
                    time.sleep(1)

                log.warning("Shutdown timeout, forcing off")
                domain.destroy()
        except libvirt.libvirtError as e:
            log.error(f"Shutdown failed: {e}")
            return False

        return True

    def export_disk(self, vm_name: str, output_dir: Path) -> list[Path]:
        """Export VM disks"""
        vm = self.get_vm(vm_name)
        if not vm:
            raise ValueError(f"VM not found: {vm_name}")

        exported = []
        for disk in vm.disks:
            if not disk.path:
                continue

            src = Path(disk.path)
            if not src.exists():
                log.warning(f"Disk not found: {disk.path}")
                continue

            dst = output_dir / src.name
            log.info(f"Copying {src.name} ({format_size(disk.size_bytes)})")

            # Use qemu-img convert for efficiency
            run_cmd([
                "qemu-img", "convert",
                "-f", disk.format,
                "-O", disk.format,
                str(src), str(dst)
            ])

            exported.append(dst)

        return exported


class KVMHostCLI:
    """KVM operations via CLI (when libvirt not available)"""

    def __init__(self, config: dict):
        self.config = config.get("kvm", {})
        self.remote = self.config.get("remote_host", "")

    def _run(self, cmd: list) -> subprocess.CompletedProcess:
        """Run command locally or remotely"""
        if self.remote:
            cmd = ["ssh", self.remote] + cmd
        return run_cmd(cmd)

    def list_vms(self) -> list[VMInfo]:
        """List VMs using virsh"""
        result = self._run(["virsh", "list", "--all"])
        vms = []

        for line in result.stdout.strip().split("\n")[2:]:
            if not line.strip():
                continue
            parts = line.split()
            if len(parts) >= 2:
                name = parts[1]
                state = " ".join(parts[2:]) if len(parts) > 2 else "unknown"

                # Get disk info
                disk_result = self._run(["virsh", "domblklist", name])
                disks = []
                for dline in disk_result.stdout.split("\n")[2:]:
                    if dline.strip():
                        dparts = dline.split()
                        if len(dparts) >= 2 and dparts[1] != "-":
                            disks.append(VMDisk(
                                path=dparts[1],
                                format="qcow2",
                                device=dparts[0],
                            ))

                vms.append(VMInfo(
                    name=name,
                    uuid="",
                    state=state,
                    memory_mb=0,
                    vcpus=0,
                    disks=disks,
                ))

        return vms

    def get_vm(self, name: str) -> Optional[VMInfo]:
        """Get VM by name"""
        for vm in self.list_vms():
            if vm.name == name:
                return vm
        return None

    def export_disk(self, vm_name: str, output_dir: Path) -> list[Path]:
        """Export VM disks"""
        vm = self.get_vm(vm_name)
        if not vm:
            raise ValueError(f"VM not found: {vm_name}")

        exported = []
        for disk in vm.disks:
            if not disk.path:
                continue

            src_path = disk.path
            dst = output_dir / Path(src_path).name

            if self.remote:
                log.info(f"Copying {Path(src_path).name} from remote host")
                run_cmd(["scp", f"{self.remote}:{src_path}", str(dst)])
            else:
                log.info(f"Copying {Path(src_path).name}")
                shutil.copy2(src_path, dst)

            exported.append(dst)

        return exported


def get_kvm_host(config: dict):
    """Get appropriate KVM host handler"""
    if HAS_LIBVIRT and not config.get("kvm", {}).get("remote_host"):
        return KVMHost(config)
    return KVMHostCLI(config)


# ============================================
# DISK CONVERSION
# ============================================


class DiskConverter:
    """Convert disk images between formats"""

    def __init__(self, config: dict):
        self.config = config.get("conversion", {})
        self.work_dir = Path(self.config.get("work_dir", "/tmp/kvm2oci"))
        self.work_dir.mkdir(parents=True, exist_ok=True)

    def convert(self, input_path: Path, output_format: str = None) -> Path:
        """Convert disk to specified format"""
        if output_format is None:
            output_format = self.config.get("output_format", "qcow2")

        log.info(f"Converting {input_path.name} to {output_format}")

        # Detect input format
        result = run_cmd(["qemu-img", "info", "--output=json", str(input_path)])
        info = json.loads(result.stdout)
        input_format = info.get("format", "raw")

        output_path = self.work_dir / f"{input_path.stem}.{output_format}"

        cmd = ["qemu-img", "convert", "-f", input_format]

        if self.config.get("compress") and output_format == "qcow2":
            cmd.append("-c")

        cmd.extend(["-O", output_format, str(input_path), str(output_path)])

        log.info("Running conversion...")
        run_cmd(cmd)

        log.info(f"Converted: {format_size(output_path.stat().st_size)}")
        return output_path

    def get_disk_info(self, path: Path) -> dict:
        """Get disk image info"""
        result = run_cmd(["qemu-img", "info", "--output=json", str(path)])
        return json.loads(result.stdout)


# ============================================
# OCI OPERATIONS
# ============================================


class OCIClient:
    """Oracle Cloud Infrastructure client"""

    def __init__(self, config: dict):
        self.config = config.get("oci", {})
        oci_config_path = Path(self.config.get("config_file", "~/.oci/config")).expanduser()

        if not oci_config_path.exists():
            raise FileNotFoundError(f"OCI config not found: {oci_config_path}")

        self.oci_config = oci.config.from_file(
            str(oci_config_path),
            self.config.get("profile", "DEFAULT")
        )

        self.object_storage = oci.object_storage.ObjectStorageClient(self.oci_config)
        self.compute = oci.core.ComputeClient(self.oci_config)

        if not self.config.get("namespace"):
            self.config["namespace"] = self.object_storage.get_namespace().data

    def upload_disk(self, disk_path: Path, object_name: str = None) -> str:
        """Upload disk to OCI Object Storage"""
        bucket = self.config["bucket_name"]
        namespace = self.config["namespace"]

        if object_name is None:
            object_name = disk_path.name

        file_size = disk_path.stat().st_size
        log.info(f"Uploading {object_name} ({format_size(file_size)})")

        if file_size > 100 * 1024 * 1024:
            return self._multipart_upload(disk_path, object_name)

        with open(disk_path, "rb") as f:
            self.object_storage.put_object(namespace, bucket, object_name, f)

        log.info("Upload complete")
        return object_name

    def _multipart_upload(self, disk_path: Path, object_name: str) -> str:
        """Multipart upload for large files"""
        namespace = self.config["namespace"]
        bucket = self.config["bucket_name"]

        create_response = self.object_storage.create_multipart_upload(
            namespace,
            bucket,
            oci.object_storage.models.CreateMultipartUploadDetails(object=object_name),
        )
        upload_id = create_response.data.upload_id

        try:
            parts = []
            part_size = 50 * 1024 * 1024
            part_num = 1

            with open(disk_path, "rb") as f:
                while True:
                    data = f.read(part_size)
                    if not data:
                        break

                    log.info(f"  Uploading part {part_num}...")
                    response = self.object_storage.upload_part(
                        namespace, bucket, object_name, upload_id, part_num, data
                    )
                    parts.append(
                        oci.object_storage.models.CommitMultipartUploadPartDetails(
                            part_num=part_num,
                            etag=response.headers["etag"],
                        )
                    )
                    part_num += 1

            self.object_storage.commit_multipart_upload(
                namespace, bucket, object_name, upload_id,
                oci.object_storage.models.CommitMultipartUploadDetails(parts_to_commit=parts)
            )
            log.info("Upload complete")

        except Exception:
            self.object_storage.abort_multipart_upload(namespace, bucket, object_name, upload_id)
            raise

        return object_name

    def import_image(
        self,
        object_name: str,
        image_name: str,
        launch_mode: str = "PARAVIRTUALIZED",
    ) -> str:
        """Import disk as OCI custom image"""
        compartment_id = self.config["compartment_id"]
        namespace = self.config["namespace"]
        bucket = self.config["bucket_name"]

        if not compartment_id:
            raise ValueError("compartment_id not configured")

        log.info(f"Importing image: {image_name}")

        source_type = "QCOW2"
        if object_name.endswith(".vmdk"):
            source_type = "VMDK"
        elif object_name.endswith(".vhd"):
            source_type = "VHD"

        response = self.compute.create_image(
            oci.core.models.CreateImageDetails(
                compartment_id=compartment_id,
                display_name=image_name,
                launch_mode=launch_mode,
                image_source_details=oci.core.models.ImageSourceViaObjectStorageTupleDetails(
                    source_type="objectStorageTuple",
                    namespace_name=namespace,
                    bucket_name=bucket,
                    object_name=object_name,
                    source_image_type=source_type,
                    operating_system="Custom",
                    operating_system_version="Custom",
                ),
            )
        )

        image_id = response.data.id
        log.info(f"Image import started: {image_id}")

        log.info("Waiting for import to complete...")
        oci.wait_until(
            self.compute,
            self.compute.get_image(image_id),
            "lifecycle_state",
            "AVAILABLE",
            max_wait_seconds=3600,
        )

        log.info("Image ready")
        return image_id

    def list_images(self) -> list:
        """List custom images"""
        compartment_id = self.config["compartment_id"]
        response = self.compute.list_images(compartment_id, lifecycle_state="AVAILABLE")
        return [
            {"id": img.id, "name": img.display_name, "created": img.time_created}
            for img in response.data
            if img.operating_system == "Custom"
        ]


# ============================================
# MIGRATION WORKFLOW
# ============================================


class MigrationWorkflow:
    """Full migration workflow"""

    def __init__(self, config: dict):
        self.config = config
        self.kvm = get_kvm_host(config)
        self.converter = DiskConverter(config)
        self.oci = OCIClient(config)
        self.work_dir = Path(config.get("conversion", {}).get("work_dir", "/tmp/kvm2oci"))
        self.work_dir.mkdir(parents=True, exist_ok=True)

    def migrate(
        self,
        vm_name: str = None,
        disk_path: Path = None,
        image_name: str = None,
        shutdown: bool = True,
        cleanup: bool = True,
    ) -> str:
        """Run full migration"""
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")

        if disk_path is None and vm_name:
            log.info("Step 1/4: Exporting VM disk")

            if shutdown and hasattr(self.kvm, 'shutdown_vm'):
                vm = self.kvm.get_vm(vm_name)
                if vm and vm.state == "running":
                    self.kvm.shutdown_vm(vm_name)

            disks = self.kvm.export_disk(vm_name, self.work_dir)
            if not disks:
                raise RuntimeError("No disks exported")
            disk_path = disks[0]  # Primary disk
        elif disk_path:
            log.info("Step 1/4: Using provided disk")
        else:
            raise ValueError("Either vm_name or disk_path required")

        log.info("Step 2/4: Converting disk format")
        converted = self.converter.convert(disk_path)

        log.info("Step 3/4: Uploading to OCI")
        object_name = f"kvm2oci/{timestamp}/{converted.name}"
        self.oci.upload_disk(converted, object_name)

        log.info("Step 4/4: Importing as OCI image")
        if image_name is None:
            image_name = vm_name or disk_path.stem
        image_name = f"{image_name}-{timestamp}"

        image_id = self.oci.import_image(
            object_name,
            image_name,
            launch_mode=self.config.get("image", {}).get("launch_mode", "PARAVIRTUALIZED"),
        )

        if cleanup:
            log.info("Cleaning up")
            shutil.rmtree(self.work_dir, ignore_errors=True)

        log.info("=" * 50)
        log.info("MIGRATION COMPLETE")
        log.info(f"Image ID: {image_id}")
        log.info(f"Image Name: {image_name}")
        log.info("=" * 50)

        return image_id


# ============================================
# CLI
# ============================================


def load_config() -> dict:
    """Load configuration"""
    for path in CONFIG_PATHS:
        if path.exists():
            log.info(f"Loading config: {path}")
            with open(path) as f:
                user_config = yaml.safe_load(f) or {}
            config = DEFAULT_CONFIG.copy()
            for key, value in user_config.items():
                if isinstance(value, dict) and key in config:
                    config[key].update(value)
                else:
                    config[key] = value
            return config
    return DEFAULT_CONFIG


def create_sample_config():
    """Create sample config"""
    sample = """# KVM to OCI Migration Configuration

kvm:
  uri: qemu:///system  # Or qemu+ssh://user@host/system
  remote_host: ""      # For remote hosts without libvirt

oci:
  config_file: ~/.oci/config
  profile: DEFAULT
  compartment_id: ocid1.compartment.oc1..xxxxx
  bucket_name: vm-migrations

conversion:
  output_format: qcow2
  compress: true
  work_dir: /tmp/kvm2oci

image:
  launch_mode: PARAVIRTUALIZED
  operating_system: Custom
"""
    config_path = CONFIG_PATHS[0]
    config_path.write_text(sample)
    print(f"Created: {config_path}")


def main():
    parser = argparse.ArgumentParser(description="KVM to OCI Migration Tool")
    parser.add_argument("-v", "--verbose", action="store_true")
    subparsers = parser.add_subparsers(dest="command")

    # List
    subparsers.add_parser("list", help="List VMs")

    # Export
    export_p = subparsers.add_parser("export", help="Export VM disk")
    export_p.add_argument("--vm", required=True)
    export_p.add_argument("-o", "--output", default=".")

    # Convert
    convert_p = subparsers.add_parser("convert", help="Convert disk")
    convert_p.add_argument("disk")
    convert_p.add_argument("-f", "--format", default="qcow2")

    # Upload
    upload_p = subparsers.add_parser("upload", help="Upload to OCI")
    upload_p.add_argument("disk")

    # Import
    import_p = subparsers.add_parser("import", help="Import as OCI image")
    import_p.add_argument("--object", required=True)
    import_p.add_argument("--name", required=True)

    # Migrate
    migrate_p = subparsers.add_parser("migrate", help="Full migration")
    migrate_p.add_argument("--vm", help="VM name")
    migrate_p.add_argument("--disk", help="Disk path")
    migrate_p.add_argument("--name", help="OCI image name")
    migrate_p.add_argument("--no-shutdown", action="store_true")
    migrate_p.add_argument("--no-cleanup", action="store_true")

    # Images
    subparsers.add_parser("images", help="List OCI images")

    # Init
    subparsers.add_parser("init", help="Create sample config")

    args = parser.parse_args()

    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    if args.command == "init":
        create_sample_config()
        return

    if not args.command:
        parser.print_help()
        return

    check_dependencies()
    config = load_config()

    if args.command == "list":
        kvm = get_kvm_host(config)
        if hasattr(kvm, 'connect'):
            kvm.connect()
        vms = kvm.list_vms()
        print(f"\n{'Name':<25} {'State':<12} {'Disks':<5}")
        print("-" * 45)
        for vm in vms:
            print(f"{vm.name:<25} {vm.state:<12} {len(vm.disks):<5}")

    elif args.command == "export":
        kvm = get_kvm_host(config)
        if hasattr(kvm, 'connect'):
            kvm.connect()
        output = Path(args.output)
        output.mkdir(parents=True, exist_ok=True)
        kvm.export_disk(args.vm, output)

    elif args.command == "convert":
        converter = DiskConverter(config)
        converter.convert(Path(args.disk), args.format)

    elif args.command == "upload":
        oci_client = OCIClient(config)
        oci_client.upload_disk(Path(args.disk))

    elif args.command == "import":
        oci_client = OCIClient(config)
        oci_client.import_image(args.object, args.name)

    elif args.command == "migrate":
        if not args.vm and not args.disk:
            print("Error: --vm or --disk required")
            sys.exit(1)
        workflow = MigrationWorkflow(config)
        workflow.migrate(
            vm_name=args.vm,
            disk_path=Path(args.disk) if args.disk else None,
            image_name=args.name,
            shutdown=not args.no_shutdown,
            cleanup=not args.no_cleanup,
        )

    elif args.command == "images":
        oci_client = OCIClient(config)
        images = oci_client.list_images()
        print(f"\n{'Name':<40} {'Created':<25}")
        print("-" * 65)
        for img in images:
            print(f"{img['name']:<40} {str(img['created']):<25}")


if __name__ == "__main__":
    main()
