# Blackout - Maintenance Mode Management Tool

A robust command-line tool for managing maintenance windows and blackout periods for infrastructure resources. Integrates with monitoring systems to automatically suppress alerts during planned maintenance.

## Features

- **Easy-to-use CLI** - Simple commands for managing blackouts
- **UTC timezone support** - All times stored and displayed in UTC
- **Auto-expiration** - Automatically expires blackouts when the end time is reached
- **Audit trail** - Tracks ticket numbers, reasons, and who initiated each blackout
- **Monitoring integration** - Exports active blackouts to JSON for alert suppression
- **Host status management** - Updates inventory database with current status
- **Extension support** - Extend blackouts without ending and restarting
- **Database-backed** - PostgreSQL storage with proper indexing

## Installation

### Prerequisites

- Go 1.21 or later
- PostgreSQL database
- Access to `/var/lib/adsops/` directory (for JSON export)

### Build

```bash
cd /Users/ryan/development/adsops-utils/tools/blackout
go build -o blackout
sudo mv blackout /usr/local/bin/
```

### Quick Install

```bash
cd /Users/ryan/development/adsops-utils/tools/blackout
go build && sudo cp blackout /usr/local/bin/blackout
```

## Configuration

The tool connects to PostgreSQL using environment variables or defaults:

```bash
export DB_HOST=localhost          # default: localhost
export DB_PORT=5432               # default: 5432
export DB_NAME=apiproxy           # default: apiproxy
export DB_USER=apiproxy           # default: apiproxy
export DB_PASSWORD=your_password  # default: apiproxy_secure_2026
```

## Usage

### Start a Blackout

Start a maintenance window with a ticket number, hostname, duration, and reason:

```bash
# Shorthand format
blackout CHG-2024-001 api-server-1 2:30 "Database migration"

# Explicit format
blackout start CHG-2024-001 api-server-1 2:30 "Database migration"

# 1 hour blackout
blackout CHG-2024-002 web-server-3 1:00 "Security patching"

# 90 minute blackout
blackout CHG-2024-003 cache-server-2 90m "Redis upgrade"
```

**Duration formats:**
- `H:MM` - Hours and minutes (e.g., `2:30` = 2 hours 30 minutes)
- `H` - Hours only (e.g., `2` = 2 hours)
- `MMm` - Minutes only (e.g., `90m` = 90 minutes)

### End a Blackout Early

If maintenance completes before the scheduled end time:

```bash
blackout end api-server-1
```

This will:
- Mark the blackout as completed
- Record the actual end time
- Restore the host status to "active"
- Update the monitoring export file

### List Blackouts

```bash
# List all blackouts
blackout list

# List only active blackouts
blackout list --active

# List blackouts for a specific host
blackout list --hostname api-server-1

# Combine filters
blackout list --active --hostname api-server-1
```

### Show Blackout Details

Get detailed information about the current/most recent blackout for a host:

```bash
blackout show api-server-1
```

This displays:
- Blackout ID and ticket number
- Start/end times
- Duration (scheduled and actual)
- Time remaining or overrun
- Reason and creator
- Status

### Extend a Blackout

If maintenance is taking longer than expected:

```bash
# Extend by 30 minutes
blackout extend api-server-1 0:30

# Extend by 1 hour
blackout extend api-server-1 1:00

# Extend by 45 minutes
blackout extend api-server-1 45m
```

### Manual Export

Trigger a manual export of active blackouts to JSON:

```bash
blackout export
```

This creates/updates `/var/lib/adsops/active-blackouts.json` with current active blackouts.

### Cleanup

Expire old blackouts and restore host status:

```bash
blackout cleanup
```

This:
- Marks expired blackouts as "expired"
- Restores host status to "active" for hosts with no active blackouts
- Updates the monitoring export file

### Version

```bash
blackout version
# or
blackout --version
```

## Database Schema

The tool automatically creates the following tables if they don't exist:

### inventory_resources

Tracks host inventory and status:

```sql
CREATE TABLE inventory_resources (
    id SERIAL PRIMARY KEY,
    hostname VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    resource_type VARCHAR(100),
    environment VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

**Status values:**
- `active` - Normal operation, monitoring enabled
- `blackout` - In maintenance, alerts suppressed
- `inactive` - Decommissioned or offline

### inventory_blackouts

Tracks all blackout/maintenance windows:

```sql
CREATE TABLE inventory_blackouts (
    id SERIAL PRIMARY KEY,
    ticket_number VARCHAR(50) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    actual_end_time TIMESTAMP,
    reason TEXT,
    created_by VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Status values:**
- `active` - Currently in blackout
- `completed` - Ended manually before scheduled time
- `expired` - Ended automatically at scheduled time

## Monitoring Integration

### JSON Export Format

Active blackouts are automatically exported to `/var/lib/adsops/active-blackouts.json`:

```json
[
  {
    "hostname": "api-server-1",
    "ticket": "CHG-2024-001",
    "end_time": "2024-01-13T18:30:00Z",
    "reason": "Database migration",
    "remaining_time": "1h 25m"
  },
  {
    "hostname": "web-server-3",
    "ticket": "CHG-2024-002",
    "end_time": "2024-01-13T17:00:00Z",
    "reason": "Security patching",
    "remaining_time": "45m"
  }
]
```

### Integration with oci-observability

The monitoring system should:

1. Read `/var/lib/adsops/active-blackouts.json` before each check
2. Skip checks for hosts listed in the file
3. Suppress alerts for blackout hosts
4. Optionally display blackout status in dashboards

Example Python integration:

```python
import json
from pathlib import Path

def load_active_blackouts():
    """Load active blackouts from JSON export."""
    blackout_file = Path("/var/lib/adsops/active-blackouts.json")
    if not blackout_file.exists():
        return []

    try:
        return json.loads(blackout_file.read_text())
    except Exception as e:
        print(f"Warning: Failed to load blackouts: {e}")
        return []

def is_host_in_blackout(hostname):
    """Check if a host is currently in blackout."""
    blackouts = load_active_blackouts()
    return any(b["hostname"] == hostname for b in blackouts)

# In monitoring check:
if is_host_in_blackout(hostname):
    print(f"Skipping {hostname} - in maintenance blackout")
    return  # Don't check, don't alert
```

## Automation and Scheduling

### Cron Job for Auto-Cleanup

Add to crontab to automatically expire old blackouts:

```bash
# Run cleanup every 5 minutes
*/5 * * * * /usr/local/bin/blackout cleanup >> /var/log/blackout-cleanup.log 2>&1
```

### Pre/Post Maintenance Scripts

Wrap maintenance operations with blackout commands:

```bash
#!/bin/bash
# maintenance-wrapper.sh

TICKET="CHG-2024-001"
HOSTNAME=$(hostname)
DURATION="2:00"
REASON="Automated security patching"

# Start blackout
blackout start "$TICKET" "$HOSTNAME" "$DURATION" "$REASON"

# Run maintenance
echo "Running maintenance operations..."
# ... your maintenance commands here ...

# End blackout
blackout end "$HOSTNAME"
```

### Ansible Integration

```yaml
- name: Start maintenance blackout
  command: blackout start {{ ticket }} {{ inventory_hostname }} {{ duration }} "{{ reason }}"
  delegate_to: localhost

- name: Perform maintenance
  # ... maintenance tasks ...

- name: End maintenance blackout
  command: blackout end {{ inventory_hostname }}
  delegate_to: localhost
```

## Best Practices

1. **Always use ticket numbers** - Required for audit compliance (SOX, HIPAA, etc.)
2. **Be descriptive with reasons** - Helps with troubleshooting and audit reviews
3. **End blackouts early when done** - Don't leave hosts in blackout unnecessarily
4. **Use appropriate durations** - Overestimate slightly but don't be excessive
5. **Monitor for overruns** - Check `blackout list --active` regularly
6. **Set up auto-cleanup** - Use cron to expire old blackouts automatically
7. **Document in change tickets** - Link to the actual change ticket in your system

## Troubleshooting

### Database Connection Issues

```bash
# Test database connectivity
psql -h localhost -U apiproxy -d apiproxy -c "SELECT NOW();"

# Check environment variables
echo $DB_HOST $DB_PORT $DB_NAME $DB_USER
```

### Permission Issues with JSON Export

```bash
# Create directory and set permissions
sudo mkdir -p /var/lib/adsops
sudo chmod 755 /var/lib/adsops

# Or change export path (requires code modification)
```

### Schema Not Created

The tool automatically creates tables on first run, but you can manually create them:

```bash
psql -h localhost -U apiproxy -d apiproxy -f schema.sql
```

### Host Already in Blackout

If you get an error that a host is already in blackout:

```bash
# Check current status
blackout show hostname

# Either end the current blackout
blackout end hostname

# Or extend it
blackout extend hostname 1:00
```

## Examples

### Database Migration

```bash
# 4-hour blackout for database migration
blackout CHG-2024-100 db-primary-1 4:00 "PostgreSQL 14 to 15 upgrade"
blackout CHG-2024-100 db-replica-1 4:00 "PostgreSQL 14 to 15 upgrade"

# Perform migration...

# End early if done
blackout end db-primary-1
blackout end db-replica-1
```

### Security Patching

```bash
# Batch patch all web servers (1 hour each)
for host in web-server-{1..5}; do
    blackout CHG-2024-101 $host 1:00 "Monthly security patching"
    # patch $host
    blackout end $host
done
```

### Emergency Maintenance

```bash
# Unexpected issue requiring immediate maintenance
blackout INC-2024-500 api-gateway-1 0:30 "Emergency: Fix memory leak"

# Fix issue...

blackout end api-gateway-1
```

### Scheduled Maintenance with Extension

```bash
# Start with 2-hour estimate
blackout CHG-2024-102 cache-cluster-1 2:00 "Redis cluster rebalancing"

# After 1.5 hours, need more time
blackout extend cache-cluster-1 1:00

# Finally done
blackout end cache-cluster-1
```

## Security Considerations

- Database credentials stored in environment variables (use secrets management)
- Audit trail maintained with username and timestamp
- No deletion of historical blackout records (compliance)
- Read-only access to blackout data for monitoring systems

## Compliance

This tool supports compliance requirements for:

- **SOX (Sarbanes-Oxley)** - Audit trail with ticket numbers and reasons
- **HIPAA** - Change management documentation
- **GDPR** - System maintenance tracking
- **GLBA** - Compliance with maintenance procedures
- **PCI DSS** - Change control requirements

All blackouts are logged with:
- Change ticket number
- Timestamp (UTC)
- User who initiated
- Reason for maintenance
- Actual vs. scheduled duration

## License

Copyright (c) 2024 After Dark Systems
Internal tool - All rights reserved

## Support

For issues or questions:
- Internal: #infrastructure Slack channel
- Email: ops@afterdarksys.com
