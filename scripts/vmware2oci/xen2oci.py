#!/usr/bin/env python3
"""
Xen VM to Oracle Cloud Infrastructure (OCI) Migration Tool

Converts Xen VMs (XVA, VHD, RAW) to OCI custom images.

Requirements:
    pip install oci pyyaml

Usage:
    ./xen2oci.py export --vm myvm               # Export from Xen host
    ./xen2oci.py convert disk.vhd               # Convert to OCI format
    ./xen2oci.py upload disk.qcow2              # Upload to OCI
    ./xen2oci.py import --name "My Image"       # Import as OCI image
    ./xen2oci.py migrate --vm myvm --name "VM"  # Full migration
"""

import argparse
import hashlib
import json
import logging
import os
import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
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

# ============================================
# CONFIGURATION
# ============================================

DEFAULT_CONFIG = {
    "xen": {
        "host": "",  # XenServer/XCP-ng host
        "username": "root",
        "password": "",
    },
    "oci": {
        "config_file": "~/.oci/config",
        "profile": "DEFAULT",
        "compartment_id": "",
        "bucket_name": "vm-migrations",
        "namespace": "",  # Object storage namespace
    },
    "conversion": {
        "output_format": "qcow2",  # qcow2, vmdk, vhd, raw
        "compress": True,
        "work_dir": "/tmp/xen2oci",
    },
    "image": {
        "launch_mode": "PARAVIRTUALIZED",  # PARAVIRTUALIZED, EMULATED, NATIVE
        "operating_system": "Custom",
    },
}

CONFIG_PATHS = [
    Path("./xen2oci.yaml"),
    Path("~/.config/xen2oci/config.yaml").expanduser(),
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


def run_cmd(cmd: list, check: bool = True, capture: bool = True) -> subprocess.CompletedProcess:
    """Run a command and return result"""
    log.debug(f"Running: {' '.join(cmd)}")
    result = subprocess.run(
        cmd,
        capture_output=capture,
        text=True,
        check=False,
    )
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
        log.error("Install with: brew install qemu (macOS) or apt install qemu-utils (Linux)")
        sys.exit(1)


def format_size(size: int) -> str:
    """Format bytes to human readable"""
    for unit in ["B", "KB", "MB", "GB", "TB"]:
        if size < 1024:
            return f"{size:.1f} {unit}"
        size /= 1024
    return f"{size:.1f} PB"


def get_file_hash(path: Path, algorithm: str = "md5") -> str:
    """Calculate file hash"""
    h = hashlib.new(algorithm)
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(8192), b""):
            h.update(chunk)
    return h.hexdigest()


# ============================================
# XEN OPERATIONS
# ============================================


class XenHost:
    """Interact with Xen/XCP-ng host"""

    def __init__(self, config: dict):
        self.config = config.get("xen", {})
        self.host = self.config.get("host")

    def export_vm(self, vm_name: str, output_path: Path) -> Path:
        """Export VM from Xen host as XVA"""
        if not self.host:
            raise ValueError("Xen host not configured")

        log.info(f"Exporting VM '{vm_name}' from {self.host}")

        # Use xe command via SSH
        xva_path = output_path / f"{vm_name}.xva"
        cmd = [
            "ssh", f"{self.config['username']}@{self.host}",
            f"xe vm-export vm={vm_name} filename=-"
        ]

        log.info(f"Saving to: {xva_path}")
        with open(xva_path, "wb") as f:
            proc = subprocess.Popen(cmd, stdout=f, stderr=subprocess.PIPE)
            _, stderr = proc.communicate()
            if proc.returncode != 0:
                raise RuntimeError(f"Export failed: {stderr.decode()}")

        log.info(f"Exported: {format_size(xva_path.stat().st_size)}")
        return xva_path

    def list_vms(self) -> list[dict]:
        """List VMs on Xen host"""
        if not self.host:
            raise ValueError("Xen host not configured")

        cmd = [
            "ssh", f"{self.config['username']}@{self.host}",
            "xe vm-list params=name-label,power-state,memory-static-max"
        ]
        result = run_cmd(cmd)

        vms = []
        current = {}
        for line in result.stdout.split("\n"):
            if line.startswith("name-label"):
                if current:
                    vms.append(current)
                current = {"name": line.split(":")[-1].strip()}
            elif line.startswith("power-state"):
                current["state"] = line.split(":")[-1].strip()
            elif line.startswith("memory-static-max"):
                mem = int(line.split(":")[-1].strip())
                current["memory_gb"] = mem / (1024**3)

        if current:
            vms.append(current)

        return vms


# ============================================
# DISK CONVERSION
# ============================================


class DiskConverter:
    """Convert disk images between formats"""

    SUPPORTED_INPUT = [".xva", ".vhd", ".vhdx", ".raw", ".img", ".qcow2", ".vmdk"]
    SUPPORTED_OUTPUT = ["qcow2", "vmdk", "vhd", "raw"]

    def __init__(self, config: dict):
        self.config = config.get("conversion", {})
        self.work_dir = Path(self.config.get("work_dir", "/tmp/xen2oci"))
        self.work_dir.mkdir(parents=True, exist_ok=True)

    def convert(self, input_path: Path, output_format: str = None) -> Path:
        """Convert disk to specified format"""
        if output_format is None:
            output_format = self.config.get("output_format", "qcow2")

        suffix = input_path.suffix.lower()
        log.info(f"Converting {input_path.name} to {output_format}")

        # XVA needs special handling - extract VHD first
        if suffix == ".xva":
            input_path = self._extract_xva(input_path)
            suffix = input_path.suffix.lower()

        # Convert using qemu-img
        output_path = self.work_dir / f"{input_path.stem}.{output_format}"

        cmd = ["qemu-img", "convert", "-f", self._detect_format(suffix)]

        if self.config.get("compress") and output_format == "qcow2":
            cmd.extend(["-c"])

        cmd.extend(["-O", output_format, str(input_path), str(output_path)])

        log.info("Running qemu-img conversion...")
        run_cmd(cmd)

        log.info(f"Converted: {format_size(output_path.stat().st_size)}")
        return output_path

    def _extract_xva(self, xva_path: Path) -> Path:
        """Extract VHD from XVA archive"""
        log.info("Extracting VHD from XVA...")
        extract_dir = self.work_dir / xva_path.stem

        # XVA is a tar archive
        run_cmd(["tar", "-xf", str(xva_path), "-C", str(self.work_dir)])

        # Find the disk image
        for vhd in extract_dir.rglob("*.vhd"):
            return vhd
        for raw in extract_dir.rglob("*[0-9]"):
            # Raw chunks - need to concatenate
            return self._concat_raw_chunks(extract_dir)

        raise RuntimeError("No disk image found in XVA")

    def _concat_raw_chunks(self, extract_dir: Path) -> Path:
        """Concatenate raw disk chunks from XVA"""
        output = self.work_dir / f"{extract_dir.name}.raw"
        log.info("Concatenating raw disk chunks...")

        with open(output, "wb") as outf:
            # Find numbered chunk files
            chunks = sorted(extract_dir.rglob("[0-9]*"))
            for chunk in chunks:
                if chunk.is_file():
                    with open(chunk, "rb") as inf:
                        shutil.copyfileobj(inf, outf)

        return output

    def _detect_format(self, suffix: str) -> str:
        """Detect qemu-img format from suffix"""
        format_map = {
            ".vhd": "vpc",
            ".vhdx": "vhdx",
            ".vmdk": "vmdk",
            ".qcow2": "qcow2",
            ".raw": "raw",
            ".img": "raw",
        }
        return format_map.get(suffix, "raw")


# ============================================
# OCI OPERATIONS
# ============================================


class OCIClient:
    """Oracle Cloud Infrastructure client"""

    def __init__(self, config: dict):
        self.config = config.get("oci", {})
        oci_config_path = Path(self.config.get("config_file", "~/.oci/config")).expanduser()

        if not oci_config_path.exists():
            raise FileNotFoundError(
                f"OCI config not found: {oci_config_path}\n"
                "Run: oci setup config"
            )

        self.oci_config = oci.config.from_file(
            str(oci_config_path),
            self.config.get("profile", "DEFAULT")
        )

        self.object_storage = oci.object_storage.ObjectStorageClient(self.oci_config)
        self.compute = oci.core.ComputeClient(self.oci_config)

        # Get namespace if not configured
        if not self.config.get("namespace"):
            self.config["namespace"] = self.object_storage.get_namespace().data

    def upload_disk(self, disk_path: Path, object_name: str = None) -> str:
        """Upload disk image to OCI Object Storage"""
        bucket = self.config["bucket_name"]
        namespace = self.config["namespace"]

        if object_name is None:
            object_name = disk_path.name

        file_size = disk_path.stat().st_size
        log.info(f"Uploading {object_name} ({format_size(file_size)}) to bucket '{bucket}'")

        # Use multipart upload for large files
        if file_size > 100 * 1024 * 1024:  # > 100MB
            return self._multipart_upload(disk_path, object_name)
        else:
            with open(disk_path, "rb") as f:
                self.object_storage.put_object(
                    namespace,
                    bucket,
                    object_name,
                    f,
                )
            log.info("Upload complete")
            return f"https://objectstorage.{self.oci_config['region']}.oraclecloud.com/n/{namespace}/b/{bucket}/o/{object_name}"

    def _multipart_upload(self, disk_path: Path, object_name: str) -> str:
        """Multipart upload for large files"""
        namespace = self.config["namespace"]
        bucket = self.config["bucket_name"]

        # Create multipart upload
        create_response = self.object_storage.create_multipart_upload(
            namespace,
            bucket,
            oci.object_storage.models.CreateMultipartUploadDetails(object=object_name),
        )
        upload_id = create_response.data.upload_id

        try:
            parts = []
            part_size = 50 * 1024 * 1024  # 50MB parts
            part_num = 1

            with open(disk_path, "rb") as f:
                while True:
                    data = f.read(part_size)
                    if not data:
                        break

                    log.info(f"  Uploading part {part_num}...")
                    response = self.object_storage.upload_part(
                        namespace,
                        bucket,
                        object_name,
                        upload_id,
                        part_num,
                        data,
                    )
                    parts.append(
                        oci.object_storage.models.CommitMultipartUploadPartDetails(
                            part_num=part_num,
                            etag=response.headers["etag"],
                        )
                    )
                    part_num += 1

            # Commit upload
            self.object_storage.commit_multipart_upload(
                namespace,
                bucket,
                object_name,
                upload_id,
                oci.object_storage.models.CommitMultipartUploadDetails(
                    parts_to_commit=parts
                ),
            )
            log.info("Upload complete")

        except Exception as e:
            # Abort on failure
            self.object_storage.abort_multipart_upload(
                namespace, bucket, object_name, upload_id
            )
            raise

        return f"https://objectstorage.{self.oci_config['region']}.oraclecloud.com/n/{namespace}/b/{bucket}/o/{object_name}"

    def import_image(
        self,
        object_name: str,
        image_name: str,
        launch_mode: str = "PARAVIRTUALIZED",
        operating_system: str = "Custom",
    ) -> str:
        """Import disk as OCI custom image"""
        compartment_id = self.config["compartment_id"]
        namespace = self.config["namespace"]
        bucket = self.config["bucket_name"]

        if not compartment_id:
            raise ValueError("compartment_id not configured")

        log.info(f"Importing image: {image_name}")

        # Determine source type from filename
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
                    operating_system=operating_system,
                    operating_system_version="Custom",
                ),
            )
        )

        image_id = response.data.id
        log.info(f"Image import started: {image_id}")

        # Wait for import to complete
        log.info("Waiting for import to complete...")
        waiter = oci.wait_until(
            self.compute,
            self.compute.get_image(image_id),
            "lifecycle_state",
            "AVAILABLE",
            max_wait_seconds=3600,
        )

        log.info(f"Image ready: {waiter.data.display_name}")
        return image_id

    def list_images(self, compartment_id: str = None) -> list:
        """List custom images in compartment"""
        if compartment_id is None:
            compartment_id = self.config["compartment_id"]

        response = self.compute.list_images(
            compartment_id,
            lifecycle_state="AVAILABLE",
        )
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
        self.xen = XenHost(config)
        self.converter = DiskConverter(config)
        self.oci = OCIClient(config)
        self.work_dir = Path(config.get("conversion", {}).get("work_dir", "/tmp/xen2oci"))
        self.work_dir.mkdir(parents=True, exist_ok=True)

    def migrate(
        self,
        vm_name: str = None,
        disk_path: Path = None,
        image_name: str = None,
        cleanup: bool = True,
    ) -> str:
        """Run full migration workflow"""
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")

        if disk_path is None and vm_name:
            # Export from Xen
            log.info("Step 1/4: Exporting VM from Xen")
            disk_path = self.xen.export_vm(vm_name, self.work_dir)
        elif disk_path:
            log.info("Step 1/4: Using provided disk image")
        else:
            raise ValueError("Either vm_name or disk_path required")

        # Convert to OCI format
        log.info("Step 2/4: Converting disk format")
        converted = self.converter.convert(disk_path)

        # Upload to OCI
        log.info("Step 3/4: Uploading to OCI Object Storage")
        object_name = f"xen2oci/{timestamp}/{converted.name}"
        self.oci.upload_disk(converted, object_name)

        # Import as image
        log.info("Step 4/4: Importing as OCI image")
        if image_name is None:
            image_name = vm_name or disk_path.stem
        image_name = f"{image_name}-{timestamp}"

        image_id = self.oci.import_image(
            object_name,
            image_name,
            launch_mode=self.config.get("image", {}).get("launch_mode", "PARAVIRTUALIZED"),
        )

        # Cleanup
        if cleanup:
            log.info("Cleaning up temporary files")
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
    """Create sample configuration"""
    sample = """# Xen to OCI Migration Configuration

xen:
  host: xenserver.local
  username: root
  password: ""

oci:
  config_file: ~/.oci/config
  profile: DEFAULT
  compartment_id: ocid1.compartment.oc1..xxxxx
  bucket_name: vm-migrations
  # namespace: auto-detected

conversion:
  output_format: qcow2
  compress: true
  work_dir: /tmp/xen2oci

image:
  launch_mode: PARAVIRTUALIZED  # PARAVIRTUALIZED, EMULATED, NATIVE
  operating_system: Custom
"""
    config_path = CONFIG_PATHS[0]
    config_path.write_text(sample)
    print(f"Created: {config_path}")


def main():
    parser = argparse.ArgumentParser(description="Xen to OCI Migration Tool")
    parser.add_argument("-v", "--verbose", action="store_true")
    subparsers = parser.add_subparsers(dest="command")

    # List VMs
    subparsers.add_parser("list", help="List VMs on Xen host")

    # Export
    export_p = subparsers.add_parser("export", help="Export VM from Xen")
    export_p.add_argument("--vm", required=True, help="VM name")
    export_p.add_argument("-o", "--output", help="Output directory")

    # Convert
    convert_p = subparsers.add_parser("convert", help="Convert disk format")
    convert_p.add_argument("disk", help="Disk image path")
    convert_p.add_argument("-f", "--format", default="qcow2", help="Output format")

    # Upload
    upload_p = subparsers.add_parser("upload", help="Upload to OCI")
    upload_p.add_argument("disk", help="Disk image path")
    upload_p.add_argument("-n", "--name", help="Object name")

    # Import
    import_p = subparsers.add_parser("import", help="Import as OCI image")
    import_p.add_argument("--object", required=True, help="Object name in bucket")
    import_p.add_argument("--name", required=True, help="Image name")

    # Migrate (full workflow)
    migrate_p = subparsers.add_parser("migrate", help="Full migration workflow")
    migrate_p.add_argument("--vm", help="VM name on Xen host")
    migrate_p.add_argument("--disk", help="Local disk path (alternative to --vm)")
    migrate_p.add_argument("--name", help="OCI image name")
    migrate_p.add_argument("--no-cleanup", action="store_true")

    # Images
    subparsers.add_parser("images", help="List OCI custom images")

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
        xen = XenHost(config)
        vms = xen.list_vms()
        print(f"\n{'Name':<30} {'State':<12} {'Memory':<10}")
        print("-" * 55)
        for vm in vms:
            print(f"{vm['name']:<30} {vm.get('state', 'unknown'):<12} {vm.get('memory_gb', 0):.1f} GB")

    elif args.command == "export":
        xen = XenHost(config)
        output = Path(args.output) if args.output else Path(".")
        xen.export_vm(args.vm, output)

    elif args.command == "convert":
        converter = DiskConverter(config)
        converter.convert(Path(args.disk), args.format)

    elif args.command == "upload":
        oci_client = OCIClient(config)
        oci_client.upload_disk(Path(args.disk), args.name)

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
