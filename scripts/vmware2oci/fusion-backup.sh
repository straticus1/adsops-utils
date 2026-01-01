#!/bin/bash
#
# VMware Fusion VM Backup Script
# Uses restic for block-level deduplication
#

set -euo pipefail

# ============================================
# CONFIGURATION - Edit these values
# ============================================

# Where your VMs live (Fusion default)
VM_DIR="$HOME/Virtual Machines.localized"

# Backup destination (local path, or restic remote like s3:bucket/path)
BACKUP_REPO="/Volumes/Backup/vm-backups"

# Password file for restic (create this file with your password)
PASSWORD_FILE="$HOME/.config/restic/fusion-password"

# VMs to backup (leave empty to backup all .vmwarevm bundles)
# Example: VMS=("Windows10.vmwarevm" "Windows11.vmwarevm")
VMS=()

# Suspend VMs before backup? (true/false)
SUSPEND_VMS=true

# How many backups to keep
KEEP_DAILY=7
KEEP_WEEKLY=4
KEEP_MONTHLY=3

# ============================================
# END CONFIGURATION
# ============================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check dependencies
check_deps() {
    if ! command -v restic &> /dev/null; then
        log_error "restic not found. Install with: brew install restic"
        exit 1
    fi

    if ! command -v vmrun &> /dev/null; then
        log_warn "vmrun not found. VM suspend/resume won't work."
        log_warn "Install VMware Fusion CLI: sudo ln -s '/Applications/VMware Fusion.app/Contents/Library/vmrun' /usr/local/bin/vmrun"
        SUSPEND_VMS=false
    fi
}

# Initialize restic repo if needed
init_repo() {
    if [ ! -d "$BACKUP_REPO" ] || [ ! -f "$BACKUP_REPO/config" ]; then
        log_info "Initializing restic repository at $BACKUP_REPO"

        if [ ! -f "$PASSWORD_FILE" ]; then
            mkdir -p "$(dirname "$PASSWORD_FILE")"
            log_warn "Creating password file. Please edit: $PASSWORD_FILE"
            echo "changeme-$(date +%s)" > "$PASSWORD_FILE"
            chmod 600 "$PASSWORD_FILE"
        fi

        restic -r "$BACKUP_REPO" -p "$PASSWORD_FILE" init
    fi
}

# Get list of running VMs
get_running_vms() {
    if command -v vmrun &> /dev/null; then
        vmrun list 2>/dev/null | tail -n +2 || true
    fi
}

# Suspend a VM
suspend_vm() {
    local vmx="$1"
    if [ -f "$vmx" ]; then
        log_info "Suspending: $(basename "$(dirname "$vmx")")"
        vmrun suspend "$vmx" 2>/dev/null || true
        sleep 2
    fi
}

# Resume a VM
resume_vm() {
    local vmx="$1"
    if [ -f "$vmx" ]; then
        log_info "Resuming: $(basename "$(dirname "$vmx")")"
        vmrun start "$vmx" nogui 2>/dev/null || true
    fi
}

# Find .vmx file in a .vmwarevm bundle
find_vmx() {
    local vm_bundle="$1"
    find "$vm_bundle" -maxdepth 1 -name "*.vmx" -type f 2>/dev/null | head -1
}

# Main backup function
backup_vms() {
    local suspended_vms=()
    local running_before
    running_before=$(get_running_vms)

    # Discover VMs to backup
    local vms_to_backup=()
    if [ ${#VMS[@]} -eq 0 ]; then
        while IFS= read -r -d '' vm; do
            vms_to_backup+=("$vm")
        done < <(find "$VM_DIR" -maxdepth 1 -name "*.vmwarevm" -type d -print0 2>/dev/null)
    else
        for vm in "${VMS[@]}"; do
            vms_to_backup+=("$VM_DIR/$vm")
        done
    fi

    if [ ${#vms_to_backup[@]} -eq 0 ]; then
        log_error "No VMs found in $VM_DIR"
        exit 1
    fi

    log_info "Found ${#vms_to_backup[@]} VM(s) to backup"

    # Suspend running VMs if configured
    if [ "$SUSPEND_VMS" = true ]; then
        for vm_path in "${vms_to_backup[@]}"; do
            vmx=$(find_vmx "$vm_path")
            if [ -n "$vmx" ] && echo "$running_before" | grep -q "$vmx"; then
                suspend_vm "$vmx"
                suspended_vms+=("$vmx")
            fi
        done

        # Wait for suspend to complete
        if [ ${#suspended_vms[@]} -gt 0 ]; then
            log_info "Waiting for VMs to fully suspend..."
            sleep 5
        fi
    fi

    # Run backup
    log_info "Starting restic backup..."
    local backup_paths=()
    for vm_path in "${vms_to_backup[@]}"; do
        backup_paths+=("$vm_path")
    done

    local start_time=$(date +%s)

    restic -r "$BACKUP_REPO" -p "$PASSWORD_FILE" backup \
        --verbose \
        --tag "fusion-vms" \
        --exclude="*.log" \
        --exclude="*.lck" \
        --exclude="vmware*.log.*" \
        "${backup_paths[@]}"

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log_info "Backup completed in ${duration}s"

    # Resume suspended VMs
    for vmx in "${suspended_vms[@]}"; do
        resume_vm "$vmx"
    done

    # Prune old backups
    log_info "Pruning old backups..."
    restic -r "$BACKUP_REPO" -p "$PASSWORD_FILE" forget \
        --keep-daily "$KEEP_DAILY" \
        --keep-weekly "$KEEP_WEEKLY" \
        --keep-monthly "$KEEP_MONTHLY" \
        --prune
}

# Show backup stats
show_stats() {
    log_info "Repository statistics:"
    restic -r "$BACKUP_REPO" -p "$PASSWORD_FILE" stats
    echo ""
    log_info "Available snapshots:"
    restic -r "$BACKUP_REPO" -p "$PASSWORD_FILE" snapshots
}

# List available snapshots
list_snapshots() {
    restic -r "$BACKUP_REPO" -p "$PASSWORD_FILE" snapshots --tag "fusion-vms"
}

# Restore a VM from backup
restore_vm() {
    local snapshot="${1:-latest}"
    local target="${2:-$VM_DIR/restored}"

    log_info "Restoring snapshot $snapshot to $target"
    mkdir -p "$target"
    restic -r "$BACKUP_REPO" -p "$PASSWORD_FILE" restore "$snapshot" --target "$target"
    log_info "Restore complete. VMs available at: $target"
}

# Usage
usage() {
    cat << EOF
VMware Fusion Backup Script

Usage: $(basename "$0") <command>

Commands:
    backup      Run incremental backup of all configured VMs
    stats       Show repository statistics and snapshots
    list        List available backup snapshots
    restore     Restore from backup (restore [snapshot] [target])
    init        Initialize the backup repository
    help        Show this help message

Examples:
    $(basename "$0") backup              # Run backup
    $(basename "$0") restore latest      # Restore latest to default location
    $(basename "$0") restore abc123 /tmp # Restore specific snapshot

Configuration:
    Edit the variables at the top of this script to configure:
    - VM_DIR: Location of your VMs
    - BACKUP_REPO: Where to store backups
    - VMS: Specific VMs to backup (empty = all)
    - SUSPEND_VMS: Whether to suspend VMs during backup

First run:
    1. Install restic: brew install restic
    2. Edit configuration at top of script
    3. Run: $(basename "$0") init
    4. Run: $(basename "$0") backup
EOF
}

# Main
main() {
    check_deps

    case "${1:-help}" in
        backup)
            init_repo
            backup_vms
            ;;
        stats)
            show_stats
            ;;
        list)
            list_snapshots
            ;;
        restore)
            restore_vm "${2:-latest}" "${3:-}"
            ;;
        init)
            init_repo
            log_info "Repository initialized at $BACKUP_REPO"
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            log_error "Unknown command: $1"
            usage
            exit 1
            ;;
    esac
}

main "$@"
