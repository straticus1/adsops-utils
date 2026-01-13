#!/bin/bash
# AdsOps Utils - Host Management Tools Deployment Script
# Deploys SSH key tracker, hostctl, blackout tool, and oci-observability integration

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_PREFIX="${INSTALL_PREFIX:-/usr/local}"
DATA_DIR="${DATA_DIR:-/var/lib/adsops}"
CONFIG_DIR="${CONFIG_DIR:-/etc/adsops}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Check if running as root
if [ "$EUID" -ne 0 ] && [ -z "$SUDO_USER" ]; then
    echo -e "${YELLOW}Warning: Not running as root. Some operations may require sudo.${NC}"
fi

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           AdsOps Utils - Host Management Deployment           ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Create directories
echo -e "${BLUE}[1/7] Creating directories...${NC}"
mkdir -p "$DATA_DIR" "$CONFIG_DIR" "$INSTALL_PREFIX/bin" "$INSTALL_PREFIX/bin/adsops"
chmod 755 "$DATA_DIR" "$CONFIG_DIR"
echo -e "${GREEN}✓ Directories created${NC}"
echo ""

# Install SSH Key Tracker
echo -e "${BLUE}[2/7] Installing SSH Key Tracker...${NC}"
if [ -f "$SCRIPT_DIR/tools/ssh-key-tracker.sh" ]; then
    cp "$SCRIPT_DIR/tools/ssh-key-tracker.sh" "$INSTALL_PREFIX/bin/ssh-key-tracker"
    chmod +x "$INSTALL_PREFIX/bin/ssh-key-tracker"
    echo -e "${GREEN}✓ SSH Key Tracker installed to $INSTALL_PREFIX/bin/ssh-key-tracker${NC}"
else
    echo -e "${RED}✗ SSH Key Tracker not found at $SCRIPT_DIR/tools/ssh-key-tracker.sh${NC}"
fi
echo ""

# Build and install hostctl
echo -e "${BLUE}[3/7] Building and installing hostctl...${NC}"
if [ -d "$SCRIPT_DIR/tools/hostctl" ]; then
    cd "$SCRIPT_DIR/tools/hostctl"
    
    # Check for Go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}✗ Go is not installed. Please install Go 1.21+ and try again.${NC}"
        exit 1
    fi
    
    # Build
    echo "   Building hostctl binary..."
    go build -o hostctl .
    
    # Install
    cp hostctl "$INSTALL_PREFIX/bin/hostctl"
    chmod +x "$INSTALL_PREFIX/bin/hostctl"
    echo -e "${GREEN}✓ hostctl installed to $INSTALL_PREFIX/bin/hostctl${NC}"
else
    echo -e "${RED}✗ hostctl source not found at $SCRIPT_DIR/tools/hostctl${NC}"
fi
echo ""

# Build and install blackout
echo -e "${BLUE}[4/7] Building and installing blackout...${NC}"
if [ -d "$SCRIPT_DIR/tools/blackout" ]; then
    cd "$SCRIPT_DIR/tools/blackout"
    
    # Build
    echo "   Building blackout binary..."
    go build -o blackout .
    
    # Install
    cp blackout "$INSTALL_PREFIX/bin/blackout"
    chmod +x "$INSTALL_PREFIX/bin/blackout"
    echo -e "${GREEN}✓ blackout installed to $INSTALL_PREFIX/bin/blackout${NC}"
else
    echo -e "${RED}✗ blackout source not found at $SCRIPT_DIR/tools/blackout${NC}"
fi
echo ""

# Initialize inventory database
echo -e "${BLUE}[5/7] Initializing inventory database...${NC}"
if command -v hostctl &> /dev/null; then
    # Check if database is accessible
    if hostctl list &> /dev/null; then
        echo -e "${GREEN}✓ Inventory database accessible${NC}"
    else
        echo -e "${YELLOW}⚠ Database not accessible. Please configure database connection.${NC}"
        echo -e "${YELLOW}  Set environment variables:${NC}"
        echo -e "${YELLOW}    POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB${NC}"
        echo -e "${YELLOW}    POSTGRES_USER, POSTGRES_PASSWORD${NC}"
    fi
else
    echo -e "${YELLOW}⚠ hostctl not available to test database${NC}"
fi
echo ""

# Install systemd services (if systemd is available)
echo -e "${BLUE}[6/7] Installing systemd services...${NC}"
if command -v systemctl &> /dev/null; then
    # Install blackout cleanup timer
    cat > /etc/systemd/system/blackout-cleanup.service <<'EOF'
[Unit]
Description=Blackout Cleanup Service
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/blackout cleanup
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    cat > /etc/systemd/system/blackout-cleanup.timer <<'EOF'
[Unit]
Description=Blackout Cleanup Timer (runs every 5 minutes)

[Timer]
OnBootSec=5min
OnUnitActiveSec=5min
Unit=blackout-cleanup.service

[Install]
WantedBy=timers.target
EOF

    systemctl daemon-reload
    systemctl enable blackout-cleanup.timer
    systemctl start blackout-cleanup.timer
    
    echo -e "${GREEN}✓ Blackout cleanup timer installed and started${NC}"
else
    echo -e "${YELLOW}⚠ systemd not available, skipping service installation${NC}"
fi
echo ""

# Summary
echo -e "${BLUE}[7/7] Deployment Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${GREEN}Installed Components:${NC}"
echo ""
echo -e "  📦 ${BLUE}ssh-key-tracker${NC} → $INSTALL_PREFIX/bin/ssh-key-tracker"
echo -e "     Track SSH keys and host mappings"
echo -e "     Usage: ssh-key-tracker add <host> <ip> <user> <keyfile>"
echo ""
echo -e "  📦 ${BLUE}hostctl${NC} → $INSTALL_PREFIX/bin/hostctl"
echo -e "     Manage host inventory"
echo -e "     Usage: hostctl add <hostname> --ip <ip> --type <type> --env <env>"
echo ""
echo -e "  📦 ${BLUE}blackout${NC} → $INSTALL_PREFIX/bin/blackout"
echo -e "     Manage maintenance blackouts"
echo -e "     Usage: blackout start <ticket> <hostname> <duration> \"reason\""
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo ""
echo -e "  1. Configure database connection (if not already done):"
echo -e "     ${BLUE}export POSTGRES_HOST=localhost${NC}"
echo -e "     ${BLUE}export POSTGRES_PORT=5432${NC}"
echo -e "     ${BLUE}export POSTGRES_DB=inventory${NC}"
echo -e "     ${BLUE}export POSTGRES_USER=your_user${NC}"
echo -e "     ${BLUE}export POSTGRES_PASSWORD=your_password${NC}"
echo ""
echo -e "  2. Add production hosts to inventory:"
echo -e "     ${BLUE}hostctl add darkapi-api-1 --ip 132.145.179.230 --type application_server --env production${NC}"
echo ""
echo -e "  3. Track SSH keys:"
echo -e "     ${BLUE}ssh-key-tracker add darkapi-api-1 132.145.179.230 opc ~/.ssh/darkapi_key \"API Proxy\"${NC}"
echo ""
echo -e "  4. For monitoring integration, deploy oci-observability role:"
echo -e "     ${BLUE}cd ../oci-observability/ansible${NC}"
echo -e "     ${BLUE}ansible-playbook -i inventory playbook.yml${NC}"
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Deployment complete!${NC}"
echo ""
