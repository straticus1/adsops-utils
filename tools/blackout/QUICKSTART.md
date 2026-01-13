# Blackout Tool - Quick Start Guide

Get started with the blackout tool in under 5 minutes.

## Installation

```bash
cd /Users/ryan/development/adsops-utils/tools/blackout
make install
```

## Basic Usage

### 1. Start a Blackout

Put a host into maintenance mode:

```bash
blackout CHG-2024-001 api-server-1 2:30 "Database migration"
```

This creates a 2.5 hour blackout starting now.

### 2. Check Status

See details about a host's blackout:

```bash
blackout show api-server-1
```

### 3. List Active Blackouts

View all currently active maintenance windows:

```bash
blackout list --active
```

### 4. End Early

If maintenance finishes before scheduled:

```bash
blackout end api-server-1
```

## Common Scenarios

### Security Patching

```bash
# Start 1-hour blackout
blackout CHG-2024-100 web-server-1 1:00 "Monthly security patching"

# Run your patches
sudo apt-get update && sudo apt-get upgrade -y

# End when done
blackout end web-server-1
```

### Database Migration

```bash
# Start 4-hour blackout
blackout CHG-2024-101 db-primary-1 4:00 "PostgreSQL upgrade"

# Need more time?
blackout extend db-primary-1 1:00

# Done
blackout end db-primary-1
```

### Emergency Maintenance

```bash
# Quick 30-minute blackout
blackout INC-2024-500 app-server-3 30m "Emergency: Memory leak fix"

# Fix the issue...

blackout end app-server-3
```

## Duration Formats

- `2:30` = 2 hours 30 minutes
- `1:00` = 1 hour
- `90m` = 90 minutes
- `4` = 4 hours

## What Happens During Blackout?

1. Host status updated to "blackout" in inventory database
2. Blackout record created with ticket, reason, timestamps
3. JSON file exported to `/var/lib/adsops/active-blackouts.json`
4. Monitoring systems read this file and suppress alerts
5. At end time (or manual end), host restored to "active"

## Configuration

Set database connection via environment variables:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=apiproxy
export DB_USER=apiproxy
export DB_PASSWORD=your_password
```

Or use defaults (localhost, port 5432, database "apiproxy").

## Monitoring Integration

Your monitoring system should check for blackouts before alerting:

```python
import json

def is_blackout(hostname):
    with open('/var/lib/adsops/active-blackouts.json') as f:
        blackouts = json.load(f)
    return any(b['hostname'] == hostname for b in blackouts)

# In your check:
if is_blackout('api-server-1'):
    print("Host in maintenance, skipping check")
    exit(0)
```

## Help

```bash
blackout help        # Show full help
blackout version     # Show version
blackout list        # List all blackouts
```

## Next Steps

- Read [README.md](README.md) for detailed documentation
- See [INSTALL.md](INSTALL.md) for installation options
- Check [examples/](examples/) for integration examples
- Set up auto-cleanup with systemd or cron (see INSTALL.md)

## Troubleshooting

**Can't connect to database:**
```bash
# Test connection
psql -h localhost -U apiproxy -d apiproxy -c "SELECT NOW();"
```

**Permission denied on JSON export:**
```bash
sudo mkdir -p /var/lib/adsops
sudo chmod 755 /var/lib/adsops
```

**Host already in blackout:**
```bash
# Check current status
blackout show hostname

# End existing blackout
blackout end hostname
```

## Support

- Slack: #infrastructure
- Email: ops@afterdarksys.com
- GitHub: https://github.com/afterdarksystems/adsops-utils
