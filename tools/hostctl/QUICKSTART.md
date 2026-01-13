# hostctl Quick Start Guide

## Installation (30 seconds)

```bash
cd /Users/ryan/development/adsops-utils/tools/hostctl
make deps && make install
```

## Configuration (30 seconds)

```bash
# Add to ~/.bashrc or ~/.zshrc
export INVENTORY_DB_HOST="afterdarksys.com"
export INVENTORY_DB_PORT="5432"
export INVENTORY_DB_NAME="inventory"
export INVENTORY_DB_USER="your_username"
export INVENTORY_DB_PASSWORD="your_password"

source ~/.bashrc  # or ~/.zshrc
```

## Most Common Commands

### Add a Host
```bash
hostctl add myserver \
  --ip 10.0.1.100 \
  --type server \
  --env production \
  --status active
```

### List Hosts
```bash
hostctl list                        # All hosts
hostctl list --env production       # Production only
hostctl list --status active        # Active only
hostctl list --json                 # JSON output
```

### Show Host Details
```bash
hostctl show myserver
```

### Update Host
```bash
hostctl update myserver --status maintenance
hostctl update myserver --cost-monthly 500.00
```

### Change Status
```bash
hostctl status myserver active
hostctl status myserver maintenance
```

### Search Hosts
```bash
hostctl search web              # Search by name
hostctl search 10.0.1          # Search by IP
hostctl search nginx           # Search in metadata
```

### Remove Host
```bash
hostctl remove myserver
```

## Common Workflows

### Add Production Server
```bash
hostctl add web-prod-01 \
  --ip 10.0.1.10 \
  --type server \
  --env production \
  --provider oci \
  --region us-phoenix-1 \
  --status active \
  --owners ops@example.com \
  --cost-monthly 450.00 \
  --tags '{"role":"webserver","app":"nginx"}'
```

### Maintenance Window
```bash
# Before maintenance
hostctl status web-prod-01 maintenance

# After maintenance
hostctl status web-prod-01 active
```

### Find Expensive Hosts
```bash
hostctl list --json | jq '.[] | select(.average_monthly_cost > 500)'
```

### Get Total Cost
```bash
hostctl list --json | jq '[.[].average_monthly_cost // 0] | add'
```

### Export to CSV
```bash
hostctl list --json | jq -r '.[] | [.hostname, .status, .environment, .average_monthly_cost] | @csv' > hosts.csv
```

## Field Reference

### Host Types
`server`, `container`, `vm`, `k8s-node`, `load-balancer`, `database`

### Providers
`oci`, `gcp`, `onprem`, `other`

### Environments
`production`, `staging`, `development`

### Status Values
`active`, `inactive`, `build`, `blackout`, `maintenance`, `decommissioned`

## Tips

1. **Always use --json for scripting**: More reliable parsing
2. **Combine filters**: `--env production --status active --type server`
3. **Use search for quick lookups**: Faster than remembering exact hostname
4. **Track costs**: Use `--cost-monthly` for budget tracking
5. **Tag everything**: Makes searching and categorizing easier

## Help

```bash
hostctl --help              # General help
hostctl add --help          # Command-specific help
hostctl list --help         # List options
```

## Troubleshooting

### Can't connect to database
```bash
# Check environment variables
env | grep INVENTORY_DB

# Test connection
hostctl list --limit 1
```

### Host not found
```bash
# List all hosts
hostctl list

# Search for it
hostctl search partial-name
```

### Invalid field value
```bash
# Check valid values
hostctl add --help
```

## Next Steps

- Read [README.md](README.md) for comprehensive documentation
- Check [FEATURES.md](FEATURES.md) for all capabilities
- Run [examples.sh](examples.sh) to see more examples
- Read [INSTALL.md](INSTALL.md) for advanced installation options
