# Installation Guide - Blackout Tool

Complete installation guide for the blackout maintenance mode management tool.

## Prerequisites

### System Requirements

- **Go**: Version 1.21 or later
- **PostgreSQL**: Version 12 or later
- **Operating System**: Linux (tested on Ubuntu 20.04+, RHEL 8+, OCI Linux)
- **Permissions**: Root/sudo access for installation

### Database Requirements

The tool requires a PostgreSQL database with:
- Database name: `apiproxy` (or custom via `DB_NAME`)
- User with CREATE TABLE and INSERT/UPDATE/DELETE permissions
- Network connectivity from the host where the tool runs

## Quick Install

```bash
# 1. Clone or navigate to the tool directory
cd /Users/ryan/development/adsops-utils/tools/blackout

# 2. Build and install in one command
make install

# 3. Test installation
blackout version

# 4. Run help
blackout help
```

## Manual Installation

### Step 1: Build the Binary

```bash
cd /Users/ryan/development/adsops-utils/tools/blackout

# Download dependencies
go mod download

# Build with optimizations
go build -ldflags="-s -w" -o blackout

# Verify the binary
./blackout version
```

### Step 2: Install Binary

```bash
# Copy to system path
sudo cp blackout /usr/local/bin/blackout

# Set permissions
sudo chmod 755 /usr/local/bin/blackout

# Verify installation
which blackout
blackout version
```

### Step 3: Database Setup

The tool automatically creates tables on first run, but you can pre-create them:

```bash
# Option 1: Let the tool auto-create (recommended)
blackout help  # Will create tables on first database connection

# Option 2: Manual schema creation
psql -h localhost -U apiproxy -d apiproxy -f schema.sql
```

### Step 4: Configure Database Connection

Create environment file:

```bash
sudo mkdir -p /etc/blackout
sudo cat > /etc/blackout/config.env <<EOF
DB_HOST=localhost
DB_PORT=5432
DB_NAME=apiproxy
DB_USER=apiproxy
DB_PASSWORD=your_secure_password_here
EOF

sudo chmod 600 /etc/blackout/config.env
```

### Step 5: Create JSON Export Directory

```bash
sudo mkdir -p /var/lib/adsops
sudo chmod 755 /var/lib/adsops
```

### Step 6: Set Up Auto-Cleanup (Optional but Recommended)

#### Option A: Systemd Timer (Recommended)

```bash
# Copy service and timer files
sudo cp blackout-cleanup.service /etc/systemd/system/
sudo cp blackout-cleanup.timer /etc/systemd/system/

# Create override directory for credentials
sudo mkdir -p /etc/systemd/system/blackout-cleanup.service.d

# Add database credentials
sudo cat > /etc/systemd/system/blackout-cleanup.service.d/override.conf <<EOF
[Service]
Environment="DB_PASSWORD=your_secure_password_here"
EOF

sudo chmod 600 /etc/systemd/system/blackout-cleanup.service.d/override.conf

# Reload systemd
sudo systemctl daemon-reload

# Enable and start timer
sudo systemctl enable blackout-cleanup.timer
sudo systemctl start blackout-cleanup.timer

# Check status
sudo systemctl status blackout-cleanup.timer
```

#### Option B: Cron Job

```bash
# Add to root crontab
sudo crontab -e

# Add this line (runs every 5 minutes)
*/5 * * * * DB_PASSWORD=your_password /usr/local/bin/blackout cleanup >> /var/log/blackout-cleanup.log 2>&1
```

## Post-Installation Testing

### Test 1: Database Connection

```bash
blackout list
# Should show: "No blackouts found" (if empty)
# or list existing blackouts
```

### Test 2: Create a Test Blackout

```bash
# Start a 5-minute test blackout
blackout CHG-TEST-001 test-host 5m "Installation test"

# Verify it was created
blackout show test-host

# List active blackouts
blackout list --active

# End the test blackout
blackout end test-host
```

### Test 3: Verify JSON Export

```bash
# Export blackouts
blackout export

# Check the file was created
ls -la /var/lib/adsops/active-blackouts.json
cat /var/lib/adsops/active-blackouts.json
```

## Configuration Options

### Environment Variables

The tool uses these environment variables (with defaults):

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `apiproxy` | Database name |
| `DB_USER` | `apiproxy` | Database user |
| `DB_PASSWORD` | `apiproxy_secure_2026` | Database password |

### System-Wide Configuration

Create `/etc/profile.d/blackout.sh`:

```bash
sudo cat > /etc/profile.d/blackout.sh <<'EOF'
# Blackout tool configuration
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=apiproxy
export DB_USER=apiproxy
# DB_PASSWORD should be set per-user or in secure location
EOF

sudo chmod 644 /etc/profile.d/blackout.sh
```

### Per-User Configuration

Add to `~/.bashrc` or `~/.zshrc`:

```bash
# Blackout tool
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=apiproxy
export DB_USER=apiproxy
export DB_PASSWORD='your_password_here'
```

## Integration with Monitoring Systems

### OCI Observability Integration

1. Install the Python monitoring integration:

```bash
sudo cp examples/monitoring-integration.py /usr/local/bin/check-blackout
sudo chmod 755 /usr/local/bin/check-blackout
```

2. Modify your monitoring checks to check blackout status:

```python
#!/usr/bin/env python3
import sys
from monitoring_integration import should_check_host

hostname = sys.argv[1]

if not should_check_host(hostname):
    print(f"Host {hostname} in blackout - skipping checks")
    sys.exit(0)

# Proceed with normal monitoring checks...
```

### Grafana/Prometheus Integration

Query active blackouts via SQL:

```sql
SELECT
    hostname,
    ticket_number,
    EXTRACT(EPOCH FROM (end_time - NOW())) / 60 AS minutes_remaining
FROM inventory_blackouts
WHERE status = 'active' AND end_time > NOW();
```

## Upgrading

### From Source

```bash
cd /Users/ryan/development/adsops-utils/tools/blackout

# Pull latest changes
git pull origin main

# Rebuild and reinstall
make install

# Verify new version
blackout version
```

### Database Schema Updates

The tool automatically creates missing tables and indices. To manually update:

```bash
psql -h localhost -U apiproxy -d apiproxy -f schema.sql
```

## Uninstallation

### Remove Binary

```bash
sudo rm /usr/local/bin/blackout
```

### Remove Systemd Services

```bash
sudo systemctl stop blackout-cleanup.timer
sudo systemctl disable blackout-cleanup.timer
sudo rm /etc/systemd/system/blackout-cleanup.service
sudo rm /etc/systemd/system/blackout-cleanup.timer
sudo rm -rf /etc/systemd/system/blackout-cleanup.service.d
sudo systemctl daemon-reload
```

### Remove Configuration

```bash
sudo rm -rf /etc/blackout
sudo rm /etc/profile.d/blackout.sh
```

### Remove Data (Optional - Will Delete All Blackout History!)

```bash
# Remove JSON export
sudo rm -rf /var/lib/adsops/active-blackouts.json

# Drop database tables (WARNING: Deletes all blackout history!)
psql -h localhost -U apiproxy -d apiproxy <<EOF
DROP TABLE IF EXISTS inventory_blackouts CASCADE;
DROP TABLE IF EXISTS inventory_resources CASCADE;
DROP VIEW IF EXISTS active_blackouts;
DROP VIEW IF EXISTS blackout_summary;
DROP FUNCTION IF EXISTS auto_expire_blackouts();
DROP FUNCTION IF EXISTS get_active_blackout(VARCHAR);
EOF
```

## Troubleshooting

### Database Connection Failed

```bash
# Test PostgreSQL connectivity
psql -h localhost -U apiproxy -d apiproxy -c "SELECT NOW();"

# Check if database exists
psql -h localhost -U postgres -c "\l" | grep apiproxy

# Check if user exists
psql -h localhost -U postgres -c "\du" | grep apiproxy
```

### Permission Denied on JSON Export

```bash
# Create directory with proper permissions
sudo mkdir -p /var/lib/adsops
sudo chmod 755 /var/lib/adsops

# Or change ownership to your user
sudo chown $(whoami):$(whoami) /var/lib/adsops
```

### Binary Not Found After Install

```bash
# Check PATH
echo $PATH

# Verify binary location
ls -la /usr/local/bin/blackout

# Add to PATH if needed
export PATH="/usr/local/bin:$PATH"
```

### Cleanup Not Running

```bash
# Check systemd timer status
sudo systemctl status blackout-cleanup.timer

# Check recent logs
sudo journalctl -u blackout-cleanup.service -n 50

# Manually test cleanup
blackout cleanup
```

## Security Considerations

1. **Protect Database Credentials**: Store `DB_PASSWORD` in secure locations
2. **File Permissions**: Set restrictive permissions on config files (600)
3. **Audit Trail**: All blackouts are logged with username and timestamp
4. **Network Security**: Use PostgreSQL SSL if connecting over network
5. **Access Control**: Limit who can run the blackout tool (sudo if needed)

## Support

For issues or questions:
- GitHub Issues: https://github.com/afterdarksystems/adsops-utils/issues
- Internal Slack: #infrastructure
- Email: ops@afterdarksys.com

## Additional Resources

- [README.md](README.md) - Usage guide and examples
- [schema.sql](schema.sql) - Database schema
- [examples/](examples/) - Integration examples
