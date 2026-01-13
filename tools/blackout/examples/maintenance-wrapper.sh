#!/bin/bash
#
# Maintenance Wrapper Script
#
# This script wraps maintenance operations with blackout start/end commands.
# It ensures that even if the maintenance script fails, the blackout is ended.
#
# Usage:
#   ./maintenance-wrapper.sh CHG-2024-001 2:30 "Database migration" ./migrate.sh
#

set -euo pipefail

# Configuration
BLACKOUT_TOOL="${BLACKOUT_TOOL:-/usr/local/bin/blackout}"
HOSTNAME="${HOSTNAME:-$(hostname)}"
LOG_DIR="${LOG_DIR:-/var/log/maintenance}"
LOG_FILE="${LOG_DIR}/maintenance-$(date +%Y%m%d-%H%M%S).log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log() {
    echo -e "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" | tee -a "$LOG_FILE" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*" | tee -a "$LOG_FILE"
}

cleanup() {
    local exit_code=$?

    log "Cleanup: Ending blackout for ${HOSTNAME}"

    if [[ -n "${BLACKOUT_STARTED:-}" ]]; then
        if "${BLACKOUT_TOOL}" end "${HOSTNAME}" >> "$LOG_FILE" 2>&1; then
            success "Blackout ended successfully"
        else
            error "Failed to end blackout (manual intervention required)"
        fi
    fi

    if [[ $exit_code -eq 0 ]]; then
        success "Maintenance completed successfully"
    else
        error "Maintenance failed with exit code: $exit_code"
    fi

    exit $exit_code
}

# Main script
main() {
    # Check arguments
    if [[ $# -lt 4 ]]; then
        echo "Usage: $0 <ticket> <duration> <reason> <command> [args...]"
        echo ""
        echo "Examples:"
        echo "  $0 CHG-2024-001 2:30 'Database migration' ./migrate.sh"
        echo "  $0 CHG-2024-002 1:00 'Security patching' apt-get upgrade -y"
        echo ""
        exit 1
    fi

    local ticket="$1"
    local duration="$2"
    local reason="$3"
    shift 3
    local command=("$@")

    # Create log directory
    mkdir -p "$LOG_DIR"

    log "=========================================="
    log "Maintenance Wrapper"
    log "=========================================="
    log "Ticket:   ${ticket}"
    log "Hostname: ${HOSTNAME}"
    log "Duration: ${duration}"
    log "Reason:   ${reason}"
    log "Command:  ${command[*]}"
    log "=========================================="

    # Verify blackout tool exists
    if [[ ! -x "${BLACKOUT_TOOL}" ]]; then
        error "Blackout tool not found or not executable: ${BLACKOUT_TOOL}"
        exit 1
    fi

    # Set up cleanup trap
    trap cleanup EXIT INT TERM

    # Start blackout
    log "Starting blackout..."
    if "${BLACKOUT_TOOL}" start "${ticket}" "${HOSTNAME}" "${duration}" "${reason}" >> "$LOG_FILE" 2>&1; then
        success "Blackout started"
        BLACKOUT_STARTED=1
    else
        error "Failed to start blackout"
        exit 1
    fi

    # Wait a moment for blackout to propagate
    sleep 2

    # Verify blackout is active
    log "Verifying blackout status..."
    if "${BLACKOUT_TOOL}" show "${HOSTNAME}" >> "$LOG_FILE" 2>&1; then
        success "Blackout verified"
    else
        warning "Could not verify blackout status"
    fi

    # Execute maintenance command
    log "Starting maintenance operations..."
    log "Executing: ${command[*]}"

    local start_time
    start_time=$(date +%s)

    if "${command[@]}" >> "$LOG_FILE" 2>&1; then
        local end_time
        end_time=$(date +%s)
        local elapsed=$((end_time - start_time))

        success "Maintenance command completed successfully"
        log "Elapsed time: ${elapsed} seconds"
    else
        local exit_code=$?
        error "Maintenance command failed with exit code: ${exit_code}"

        # Optionally extend blackout for troubleshooting
        warning "Consider extending blackout with: ${BLACKOUT_TOOL} extend ${HOSTNAME} 1:00"

        exit $exit_code
    fi

    # Cleanup trap will handle ending the blackout
}

# Run main function
main "$@"
