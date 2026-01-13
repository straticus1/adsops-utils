# hostctl - Host Management CLI

A comprehensive CLI tool for managing hosts in the inventory database with support for CRUD operations, status management, and advanced querying.

## Features

- **Complete CRUD Operations**: Add, update, remove, and query hosts
- **Status Management**: Update and track host status changes with automatic logging
- **Advanced Filtering**: Search and filter by status, environment, type, provider, region
- **Colorized Output**: Easy-to-read table format with color-coded status and environment
- **JSON Support**: All commands support `--json` flag for programmatic use
- **Cost Tracking**: Track daily and monthly costs per host
- **Metadata Support**: Store custom metadata as JSON
- **Owner Management**: Track owners and mail groups for each host

## Installation

```bash
cd /Users/ryan/development/adsops-utils/tools/hostctl
go mod download
go build -o hostctl
sudo mv hostctl /usr/local/bin/
```

Or install directly:

```bash
go install github.com/afterdarksys/adsops-utils/tools/hostctl@latest
```

## Configuration

Set the following environment variables to connect to the inventory database:

```bash
export INVENTORY_DB_HOST="afterdarksys.com"
export INVENTORY_DB_PORT="5432"
export INVENTORY_DB_NAME="inventory"
export INVENTORY_DB_USER="your_username"
export INVENTORY_DB_PASSWORD="your_password"
```

You can add these to your `~/.bashrc`, `~/.zshrc`, or `~/.profile`.

## Usage

### Add a new host

```bash
hostctl add web-server-01 \
  --ip 10.0.1.100 \
  --type server \
  --env production \
  --provider oci \
  --region us-phoenix-1 \
  --status active \
  --owners admin@example.com,ops@example.com \
  --cost-daily 12.50 \
  --cost-monthly 375.00 \
  --tags '{"role":"webserver","app":"nginx"}'
```

### Remove a host

```bash
hostctl remove web-server-01
```

### Update a host

```bash
hostctl update web-server-01 \
  --status maintenance \
  --cost-monthly 400.00
```

### Update host status

```bash
hostctl status web-server-01 active
```

This command also logs the status change with timestamp to the `status_changes` table.

### List hosts

```bash
# List all hosts
hostctl list

# Filter by status
hostctl list --status active

# Filter by environment
hostctl list --env production

# Filter by type and provider
hostctl list --type server --provider oci

# Limit results
hostctl list --limit 50

# JSON output
hostctl list --json
```

### Show host details

```bash
hostctl show web-server-01

# JSON output
hostctl show web-server-01 --json
```

### Search hosts

```bash
# Search by hostname, IP, tags, or external ID
hostctl search web-server
hostctl search 10.0.1
hostctl search nginx
```

## Host Types

- `server` - Physical or virtual server
- `container` - Docker container or similar
- `vm` - Virtual machine
- `k8s-node` - Kubernetes node
- `load-balancer` - Load balancer
- `database` - Database server

## Providers

- `oci` - Oracle Cloud Infrastructure
- `gcp` - Google Cloud Platform
- `onprem` - On-premises
- `other` - Other provider

## Environments

- `production` - Production environment
- `staging` - Staging environment
- `development` - Development environment

## Host Status

- `active` - Host is active and running
- `inactive` - Host is inactive
- `build` - Host is being built/provisioned
- `blackout` - Host is in blackout period (no changes)
- `maintenance` - Host is under maintenance
- `decommissioned` - Host has been decommissioned

## Examples

### Add a Kubernetes node

```bash
hostctl add k8s-worker-03 \
  --ip 10.0.2.15 \
  --type k8s-node \
  --env production \
  --provider gcp \
  --region us-west1 \
  --status active \
  --shape n1-standard-4 \
  --owners platform@example.com \
  --cost-daily 8.50
```

### Add a database server

```bash
hostctl add postgres-prod-01 \
  --ip 10.0.3.50 \
  --type database \
  --env production \
  --provider oci \
  --region us-ashburn-1 \
  --status active \
  --size VM.Standard2.4 \
  --owners dba@example.com,ops@example.com \
  --cost-monthly 450.00 \
  --tags '{"db":"postgresql","version":"15.2"}'
```

### List all production servers

```bash
hostctl list --env production --type server
```

### Put a host into maintenance mode

```bash
hostctl status web-server-01 maintenance
```

### Update host metadata

```bash
hostctl update web-server-01 \
  --tags '{"role":"webserver","app":"nginx","version":"1.24.0"}'
```

### Search for all web servers

```bash
hostctl search webserver
```

## Output Formats

### Table Format (default)

Colorized table with columns: Hostname, Type, Provider, Region, Status, Environment, IP Address

Status colors:
- Green: active
- Yellow: inactive
- Cyan: build
- Purple: blackout
- Blue: maintenance
- Red: decommissioned

Environment colors:
- Red: production
- Yellow: staging
- Green: development

### JSON Format

Use `--json` flag for machine-readable JSON output suitable for scripting and automation.

```bash
hostctl list --json | jq '.[] | select(.status == "active")'
```

## Status Change Tracking

The `hostctl status` command automatically logs all status changes to the `status_changes` table with:
- Hostname
- Old status
- New status
- Timestamp
- User who made the change (from $USER environment variable)

This provides an audit trail of all status changes.

## Error Handling

The tool provides clear error messages for:
- Invalid field values
- Database connection errors
- Host not found
- Duplicate hosts
- Missing required fields

Exit codes:
- 0: Success
- 1: Error

## Development

### Project Structure

```
hostctl/
├── main.go         # CLI setup and command definitions
├── types.go        # Data structures and types
├── database.go     # Database operations
├── commands.go     # Command implementations
├── output.go       # Output formatting and colorization
├── go.mod          # Go module definition
└── README.md       # This file
```

### Building

```bash
go build -o hostctl
```

### Testing

```bash
# Test database connection
hostctl list --limit 1

# Verbose output
hostctl list --verbose
```

## License

Copyright (c) 2024 AfterDark Systems. All rights reserved.
