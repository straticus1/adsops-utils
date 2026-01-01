#!/usr/bin/env python3
"""
VMware VM to Oracle Cloud Infrastructure (OCI) Migration Tool

Converts VMware VMs (VMDK, OVA, OVF) to OCI custom images.
Supports ESXi, vSphere, Workstation, and Fusion.

Requirements:
    pip install oci pyyaml paramiko

Usage:
    ./vmware2oci.py list                            # List VMs on ESXi
    ./vmware2oci.py export --vm myvm                # Export from ESXi
    ./vmware2oci.py convert disk.vmdk               # Convert to OCI format
    ./vmware2oci.py upload disk.qcow2               # Upload to OCI
    ./vmware2oci.py migrate --vm myvm --name "VM"   # Full migration
    ./vmware2oci.py migrate --ova file.ova          # Migrate from OVA
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
import tarfile
import tempfile
import time
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Optional

try:
    import oci
    import yaml
except ImportError:
    print("Missing dependencies. Install with:")
    print("  pip install oci pyyaml")
    sys.exit(1)

try:
    import paramiko
    HAS_PARAMIKO = True
except ImportError:
    HAS_PARAMIKO = False

# ============================================
# CONFIGURATION
# ============================================

DEFAULT_CONFIG = {
    "vmware": {
        "type": "esxi",  # esxi, vcenter, fusion, workstation
        "host": "",
        "port": 22,
        "username": "root",
        "password": "",
        "key_file": "~/.ssh/id_rsa",
        "datacenter": "",  # For vCenter
        "datastore": "",   # Default datastore
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
        "work_dir": "/tmp/vmware2oci",
        "stream_vmdk": True,  # Stream conversion without full download
    },
    "image": {
        "launch_mode": "PARAVIRTUALIZED",
        "operating_system": "Custom",
    },
}

CONFIG_PATHS = [
    Path("./vmware2oci.yaml"),
    Path("~/.config/vmware2oci/config.yaml").expanduser(),
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
    """Run command"""
    log.debug(f"Running: {' '.join(cmd)}")
    result = subprocess.run(cmd, capture_output=True, text=True, check=False)
    if check and result.returncode != 0:
        log.error(f"Command failed: {result.stderr}")
        raise RuntimeError(f"Command failed: {' '.join(cmd)}")
    return result


def check_dependencies():
    """Check required tools"""
    required = ["qemu-img"]
    optional = ["ovftool"]
    missing = []

    for tool in required:
        if not shutil.which(tool):
            missing.append(tool)

    if missing:
        log.error(f"Missing required tools: {', '.join(missing)}")
        log.error("Install with: brew install qemu")
        sys.exit(1)

    for tool in optional:
        if not shutil.which(tool):
            log.warning(f"Optional tool not found: {tool} (OVA export may not work)")


def format_size(size: int) -> str:
    """Format bytes"""
    for unit in ["B", "KB", "MB", "GB", "TB"]:
        if size < 1024:
            return f"{size:.1f} {unit}"
        size /= 1024
    return f"{size:.1f} PB"


# ============================================
# VMWARE DATA STRUCTURES
# ============================================


@dataclass
class VMDisk:
    path: str
    size_gb: float
    thin: bool = True
    controller: str = "scsi"


@dataclass
class VMInfo:
    vmid: int
    name: str
    vmx_path: str
    power_state: str
    datastore: str = ""
    guest_os: str = ""
    disks: list = field(default_factory=list)


# ============================================
# ESXI CONNECTION
# ============================================


class ESXiHost:
    """Connect to ESXi via SSH"""

    def __init__(self, config: dict):
        self.config = config.get("vmware", {})
        self.client: Optional[paramiko.SSHClient] = None
        self.sftp = None

    def connect(self):
        """Connect to ESXi"""
        if not HAS_PARAMIKO:
            raise ImportError("paramiko not installed: pip install paramiko")

        self.client = paramiko.SSHClient()
        self.client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

        kwargs = {
            "hostname": self.config["host"],
            "port": self.config.get("port", 22),
            "username": self.config["username"],
            "timeout": 30,
        }

        if self.config.get("password"):
            kwargs["password"] = self.config["password"]
        elif self.config.get("key_file"):
            key_path = Path(self.config["key_file"]).expanduser()
            if key_path.exists():
                kwargs["key_filename"] = str(key_path)

        log.info(f"Connecting to ESXi: {self.config['host']}")
        self.client.connect(**kwargs)
        self.sftp = self.client.open_sftp()
        log.info("Connected")

    def disconnect(self):
        """Disconnect"""
        if self.sftp:
            self.sftp.close()
        if self.client:
            self.client.close()

    def run(self, cmd: str, check: bool = True) -> tuple[int, str, str]:
        """Run command on ESXi"""
        if not self.client:
            raise RuntimeError("Not connected")

        stdin, stdout, stderr = self.client.exec_command(cmd, timeout=300)
        exit_code = stdout.channel.recv_exit_status()
        out = stdout.read().decode("utf-8", errors="replace")
        err = stderr.read().decode("utf-8", errors="replace")

        if check and exit_code != 0:
            raise RuntimeError(f"Command failed: {err}")

        return exit_code, out, err

    def download(self, remote: str, local: str, callback=None):
        """Download file"""
        self.sftp.get(remote, local, callback=callback)

    def file_size(self, path: str) -> int:
        """Get remote file size"""
        return self.sftp.stat(path).st_size

    def list_vms(self) -> list[VMInfo]:
        """List VMs on ESXi"""
        _, out, _ = self.run("vim-cmd vmsvc/getallvms")
        vms = []

        for line in out.strip().split("\n")[1:]:
            if not line.strip():
                continue

            match = re.match(r"(\d+)\s+(\S+)\s+\[([^\]]+)\]\s+(\S+)", line)
            if match:
                vmid, name, datastore, vmx_rel = match.groups()
                vmx_path = f"/vmfs/volumes/{datastore}/{vmx_rel}"

                _, state_out, _ = self.run(f"vim-cmd vmsvc/power.getstate {vmid}", check=False)
                power_state = "unknown"
                if "Powered on" in state_out:
                    power_state = "on"
                elif "Powered off" in state_out:
                    power_state = "off"
                elif "Suspended" in state_out:
                    power_state = "suspended"

                vms.append(VMInfo(
                    vmid=int(vmid),
                    name=name,
                    vmx_path=vmx_path,
                    power_state=power_state,
                    datastore=datastore,
                ))

        return vms

    def get_vm(self, name: str) -> Optional[VMInfo]:
        """Get VM by name"""
        for vm in self.list_vms():
            if vm.name == name:
                return vm
        return None

    def get_vm_disks(self, vm: VMInfo) -> list[VMDisk]:
        """Get disks for a VM"""
        vmx_dir = str(Path(vm.vmx_path).parent)
        _, out, _ = self.run(f"ls -la {vmx_dir}/*.vmdk 2>/dev/null", check=False)

        disks = []
        for line in out.strip().split("\n"):
            if not line or "-flat.vmdk" in line or "-delta.vmdk" in line:
                continue

            match = re.search(r"(\S+\.vmdk)$", line)
            if match:
                vmdk_name = match.group(1)
                vmdk_path = f"{vmx_dir}/{vmdk_name}"

                # Get size
                size_gb = 0.0
                _, size_out, _ = self.run(f"du -sh {vmdk_path} 2>/dev/null", check=False)
                if size_out:
                    size_match = re.match(r"([\d.]+)([GMK])", size_out)
                    if size_match:
                        val, unit = size_match.groups()
                        size_gb = float(val)
                        if unit == "M":
                            size_gb /= 1024
                        elif unit == "K":
                            size_gb /= 1024 * 1024

                disks.append(VMDisk(path=vmdk_path, size_gb=size_gb))

        return disks

    def create_snapshot(self, vm: VMInfo, name: str) -> bool:
        """Create snapshot"""
        log.info(f"Creating snapshot: {name}")
        _, out, _ = self.run(
            f'vim-cmd vmsvc/snapshot.create {vm.vmid} "{name}" "Migration snapshot" 0 0',
            check=False
        )
        return "Create Snapshot" in out or out.strip() == ""

    def remove_snapshot(self, vm: VMInfo):
        """Remove all snapshots"""
        self.run(f"vim-cmd vmsvc/snapshot.removeall {vm.vmid}", check=False)

    def export_disk(self, disk: VMDisk, output_dir: Path) -> Path:
        """Export disk to local path"""
        vmdk_name = Path(disk.path).name
        local_path = output_dir / vmdk_name

        log.info(f"Downloading: {vmdk_name}")

        try:
            size = self.file_size(disk.path)
        except Exception:
            size = 0

        downloaded = [0]

        def progress(transferred, total):
            downloaded[0] = transferred
            if total > 0:
                pct = int((transferred / total) * 100)
                if pct % 10 == 0:
                    print(f"\r  Progress: {pct}%", end="", flush=True)

        self.download(disk.path, str(local_path), callback=progress if size > 0 else None)
        print()

        # Also download flat file
        flat_path = disk.path.replace(".vmdk", "-flat.vmdk")
        try:
            flat_local = str(local_path).replace(".vmdk", "-flat.vmdk")
            self.download(flat_path, flat_local, callback=progress)
            print()
        except Exception:
            pass  # Not all VMDKs have flat files

        return local_path

    def __enter__(self):
        self.connect()
        return self

    def __exit__(self, *args):
        self.disconnect()


# ============================================
# LOCAL VMWARE (Fusion/Workstation)
# ============================================


class LocalVMware:
    """Handle local VMware (Fusion/Workstation)"""

    def __init__(self, config: dict):
        self.config = config.get("vmware", {})
        vm_type = self.config.get("type", "fusion")

        if vm_type == "fusion":
            self.vm_dir = Path.home() / "Virtual Machines.localized"
        else:
            self.vm_dir = Path.home() / "Documents" / "Virtual Machines"

    def list_vms(self) -> list[VMInfo]:
        """List local VMs"""
        vms = []

        for vm_bundle in self.vm_dir.glob("*.vmwarevm"):
            vmx_files = list(vm_bundle.glob("*.vmx"))
            if vmx_files:
                vmx = vmx_files[0]
                name = vm_bundle.stem

                # Check power state via vmrun
                state = "unknown"
                result = subprocess.run(
                    ["vmrun", "list"],
                    capture_output=True,
                    text=True,
                    check=False
                )
                if str(vmx) in result.stdout:
                    state = "on"
                else:
                    state = "off"

                vms.append(VMInfo(
                    vmid=0,
                    name=name,
                    vmx_path=str(vmx),
                    power_state=state,
                ))

        return vms

    def get_vm(self, name: str) -> Optional[VMInfo]:
        """Get VM by name"""
        for vm in self.list_vms():
            if vm.name == name:
                return vm
        return None

    def get_vm_disks(self, vm: VMInfo) -> list[VMDisk]:
        """Get disks for VM"""
        vm_dir = Path(vm.vmx_path).parent
        disks = []

        for vmdk in vm_dir.glob("*.vmdk"):
            if "-flat" in vmdk.name or "-delta" in vmdk.name:
                continue
            size = vmdk.stat().st_size / (1024**3)
            disks.append(VMDisk(path=str(vmdk), size_gb=size))

        return disks

    def export_disk(self, disk: VMDisk, output_dir: Path) -> Path:
        """Copy disk to output"""
        src = Path(disk.path)
        dst = output_dir / src.name

        log.info(f"Copying: {src.name}")
        shutil.copy2(src, dst)

        # Also copy flat file
        flat_src = Path(str(src).replace(".vmdk", "-flat.vmdk"))
        if flat_src.exists():
            flat_dst = output_dir / flat_src.name
            shutil.copy2(flat_src, flat_dst)

        return dst


# ============================================
# OVA/OVF HANDLING
# ============================================


class OVAHandler:
    """Handle OVA/OVF files"""

    def __init__(self, work_dir: Path):
        self.work_dir = work_dir

    def extract_ova(self, ova_path: Path) -> list[Path]:
        """Extract VMDK from OVA"""
        log.info(f"Extracting OVA: {ova_path.name}")

        extract_dir = self.work_dir / ova_path.stem
        extract_dir.mkdir(parents=True, exist_ok=True)

        with tarfile.open(ova_path, "r") as tar:
            tar.extractall(extract_dir)

        vmdks = list(extract_dir.glob("*.vmdk"))
        log.info(f"Found {len(vmdks)} disk(s)")
        return vmdks

    def export_ova(self, vmx_path: Path, output_path: Path) -> Path:
        """Export VM to OVA using ovftool"""
        if not shutil.which("ovftool"):
            raise RuntimeError("ovftool not found. Download from VMware.")

        log.info(f"Exporting to OVA: {output_path.name}")
        run_cmd(["ovftool", str(vmx_path), str(output_path)])
        return output_path


# ============================================
# DISK CONVERSION
# ============================================


class DiskConverter:
    """Convert VMDK to OCI-compatible format"""

    def __init__(self, config: dict):
        self.config = config.get("conversion", {})
        self.work_dir = Path(self.config.get("work_dir", "/tmp/vmware2oci"))
        self.work_dir.mkdir(parents=True, exist_ok=True)

    def convert(self, input_path: Path, output_format: str = None) -> Path:
        """Convert disk"""
        if output_format is None:
            output_format = self.config.get("output_format", "qcow2")

        log.info(f"Converting {input_path.name} to {output_format}")

        output_path = self.work_dir / f"{input_path.stem}.{output_format}"

        cmd = ["qemu-img", "convert", "-f", "vmdk"]

        if self.config.get("compress") and output_format == "qcow2":
            cmd.append("-c")

        cmd.extend(["-O", output_format, str(input_path), str(output_path)])

        run_cmd(cmd)
        log.info(f"Converted: {format_size(output_path.stat().st_size)}")
        return output_path

    def get_info(self, path: Path) -> dict:
        """Get disk info"""
        result = run_cmd(["qemu-img", "info", "--output=json", str(path)])
        return json.loads(result.stdout)


# ============================================
# OCI OPERATIONS
# ============================================


class OCIClient:
    """OCI operations"""

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
        """Upload to Object Storage"""
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
        """Multipart upload"""
        namespace = self.config["namespace"]
        bucket = self.config["bucket_name"]

        create_response = self.object_storage.create_multipart_upload(
            namespace, bucket,
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

                    log.info(f"  Part {part_num}...")
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
        """Import as OCI image"""
        compartment_id = self.config["compartment_id"]
        namespace = self.config["namespace"]
        bucket = self.config["bucket_name"]

        if not compartment_id:
            raise ValueError("compartment_id not configured")

        log.info(f"Importing: {image_name}")

        source_type = "QCOW2"
        if object_name.endswith(".vmdk"):
            source_type = "VMDK"

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
        log.info(f"Import started: {image_id}")

        log.info("Waiting for completion...")
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
        self.converter = DiskConverter(config)
        self.oci = OCIClient(config)
        self.work_dir = Path(config.get("conversion", {}).get("work_dir", "/tmp/vmware2oci"))
        self.work_dir.mkdir(parents=True, exist_ok=True)

    def _get_vmware_host(self):
        """Get appropriate VMware handler"""
        vm_type = self.config.get("vmware", {}).get("type", "esxi")
        if vm_type in ("fusion", "workstation"):
            return LocalVMware(self.config)
        return ESXiHost(self.config)

    def migrate(
        self,
        vm_name: str = None,
        disk_path: Path = None,
        ova_path: Path = None,
        image_name: str = None,
        cleanup: bool = True,
    ) -> str:
        """Run migration"""
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")

        # Source: OVA file
        if ova_path:
            log.info("Step 1/4: Extracting OVA")
            ova_handler = OVAHandler(self.work_dir)
            vmdks = ova_handler.extract_ova(ova_path)
            if not vmdks:
                raise RuntimeError("No VMDK found in OVA")
            disk_path = vmdks[0]
            if image_name is None:
                image_name = ova_path.stem

        # Source: VM on VMware host
        elif vm_name:
            log.info("Step 1/4: Exporting VM")
            vmware = self._get_vmware_host()

            if isinstance(vmware, ESXiHost):
                with vmware:
                    vm = vmware.get_vm(vm_name)
                    if not vm:
                        raise ValueError(f"VM not found: {vm_name}")

                    # Create snapshot for consistency
                    vmware.create_snapshot(vm, f"migration_{timestamp}")

                    try:
                        disks = vmware.get_vm_disks(vm)
                        if not disks:
                            raise RuntimeError("No disks found")
                        disk_path = vmware.export_disk(disks[0], self.work_dir)
                    finally:
                        vmware.remove_snapshot(vm)
            else:
                vm = vmware.get_vm(vm_name)
                if not vm:
                    raise ValueError(f"VM not found: {vm_name}")
                disks = vmware.get_vm_disks(vm)
                if not disks:
                    raise RuntimeError("No disks found")
                disk_path = vmware.export_disk(disks[0], self.work_dir)

            if image_name is None:
                image_name = vm_name

        # Source: Disk file
        elif disk_path:
            log.info("Step 1/4: Using provided disk")
            if image_name is None:
                image_name = disk_path.stem
        else:
            raise ValueError("Provide --vm, --disk, or --ova")

        # Convert
        log.info("Step 2/4: Converting disk")
        converted = self.converter.convert(disk_path)

        # Upload
        log.info("Step 3/4: Uploading to OCI")
        object_name = f"vmware2oci/{timestamp}/{converted.name}"
        self.oci.upload_disk(converted, object_name)

        # Import
        log.info("Step 4/4: Importing as OCI image")
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
    """Load config"""
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
    sample = """# VMware to OCI Migration Configuration

vmware:
  type: esxi  # esxi, vcenter, fusion, workstation
  host: esxi.local
  port: 22
  username: root
  password: ""
  key_file: ~/.ssh/id_rsa

oci:
  config_file: ~/.oci/config
  profile: DEFAULT
  compartment_id: ocid1.compartment.oc1..xxxxx
  bucket_name: vm-migrations

conversion:
  output_format: qcow2
  compress: true
  work_dir: /tmp/vmware2oci

image:
  launch_mode: PARAVIRTUALIZED
  operating_system: Custom
"""
    config_path = CONFIG_PATHS[0]
    config_path.write_text(sample)
    print(f"Created: {config_path}")


def main():
    parser = argparse.ArgumentParser(description="VMware to OCI Migration Tool")
    parser.add_argument("-v", "--verbose", action="store_true")
    subparsers = parser.add_subparsers(dest="command")

    # List
    subparsers.add_parser("list", help="List VMs")

    # Export
    export_p = subparsers.add_parser("export", help="Export VM")
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
    migrate_p.add_argument("--ova", help="OVA file path")
    migrate_p.add_argument("--name", help="OCI image name")
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
        vm_type = config.get("vmware", {}).get("type", "esxi")
        if vm_type in ("fusion", "workstation"):
            vmware = LocalVMware(config)
            vms = vmware.list_vms()
        else:
            with ESXiHost(config) as vmware:
                vms = vmware.list_vms()

        print(f"\n{'Name':<30} {'State':<12} {'Datastore':<20}")
        print("-" * 65)
        for vm in vms:
            print(f"{vm.name:<30} {vm.power_state:<12} {vm.datastore:<20}")

    elif args.command == "export":
        output = Path(args.output)
        output.mkdir(parents=True, exist_ok=True)

        vm_type = config.get("vmware", {}).get("type", "esxi")
        if vm_type in ("fusion", "workstation"):
            vmware = LocalVMware(config)
            vm = vmware.get_vm(args.vm)
            if vm:
                for disk in vmware.get_vm_disks(vm):
                    vmware.export_disk(disk, output)
        else:
            with ESXiHost(config) as vmware:
                vm = vmware.get_vm(args.vm)
                if vm:
                    for disk in vmware.get_vm_disks(vm):
                        vmware.export_disk(disk, output)

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
        if not args.vm and not args.disk and not args.ova:
            print("Error: --vm, --disk, or --ova required")
            sys.exit(1)
        workflow = MigrationWorkflow(config)
        workflow.migrate(
            vm_name=args.vm,
            disk_path=Path(args.disk) if args.disk else None,
            ova_path=Path(args.ova) if args.ova else None,
            image_name=args.name,
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
