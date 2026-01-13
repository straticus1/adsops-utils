# hostctl Feature Overview

## Core Features

### 1. Complete CRUD Operations

#### Create (Add)
- Add new hosts with comprehensive metadata
- Support for multiple host types (server, container, vm, k8s-node, load-balancer, database)
- Multiple cloud providers (OCI, GCP, on-premises, other)
- Environment classification (production, staging, development)
- Cost tracking (daily and monthly)
- Custom metadata as JSON
- Owner and mail group management

#### Read (List, Show, Search)
- **List**: View all hosts with filtering options
- **Show**: Detailed view of a single host
- **Search**: Full-text search across hostname, IP, tags, and external IDs

#### Update
- Selective field updates
- Metadata merging (doesn't overwrite entire metadata)
- Dynamic query building (only updates specified fields)
- Validation of all field values

#### Delete (Remove)
- Safe deletion with existence check
- Returns clear error if host not found
- Permanent removal from database

### 2. Status Management

#### Status Workflow
```
build → active → maintenance → active
         ↓           ↓
     inactive   blackout
         ↓           ↓
  decommissioned ←────┘
```

#### Status Change Tracking
- Automatic logging to `status_changes` table
- Captures: hostname, old status, new status, timestamp, user
- Creates audit trail for compliance
- User identified from $USER environment variable

### 3. Advanced Filtering

#### Filter by:
- **Status**: active, inactive, build, blackout, maintenance, decommissioned
- **Environment**: production, staging, development
- **Type**: server, container, vm, k8s-node, load-balancer, database
- **Provider**: oci, gcp, onprem, other
- **Region**: Any region string
- **Limit**: Control result count

#### Multiple Filters
Combine filters for precise queries:
```bash
hostctl list --env production --type server --provider oci --status active
```

### 4. Flexible Output Formats

#### Table Format (Default)
- Colorized output for easy reading
- Status colors: Green (active), Yellow (inactive), Cyan (build), Purple (blackout), Blue (maintenance), Red (decommissioned)
- Environment colors: Red (production), Yellow (staging), Green (development)
- Columns: Hostname, Type, Provider, Region, Status, Environment, IP

#### JSON Format
- Machine-readable output
- Perfect for scripting and automation
- Works with all commands
- Compatible with jq and other JSON tools

#### Detailed Format
- Show command provides comprehensive view
- Organized sections: Basic Info, Contacts, Metadata, Cost, External Refs, Timestamps
- Human-readable formatting

### 5. Cost Tracking

- Daily cost tracking
- Monthly cost tracking
- Decimal precision (2 places)
- Optional fields (can be NULL)
- Easy aggregation for reporting

### 6. Metadata Management

#### JSON-Based Metadata
- Flexible key-value storage
- No schema restrictions
- Automatic merging on updates
- Full-text searchable

#### Built-in Fields
- `ip`: IP address
- `size`: Instance size
- `shape`: Instance shape
- Any custom fields via --tags

#### Example Metadata
```json
{
  "ip": "10.0.1.100",
  "size": "large",
  "shape": "VM.Standard2.4",
  "role": "webserver",
  "app": "nginx",
  "version": "1.24.0",
  "backup_enabled": true,
  "monitoring": true
}
```

### 7. Owner Management

- Multiple owners per host (array)
- Multiple mail groups per host (array)
- Comma-separated input
- Easy to query and report

### 8. External Reference Tracking

- **External ID**: Link to cloud provider resource ID
- **External URL**: Direct link to management console
- Useful for integration with cloud APIs
- Searchable fields

### 9. Database Features

#### Connection Management
- Environment-based configuration
- Secure SSL/TLS connection (sslmode=require)
- Connection pooling
- Automatic reconnection
- Configurable timeout

#### Performance
- Indexed columns for fast queries
- JSONB for efficient metadata storage and querying
- Prepared statements
- Batch operations support

### 10. Error Handling

#### Comprehensive Validation
- Field value validation
- Type checking
- Existence verification
- Clear error messages
- Exit codes (0 = success, 1 = error)

#### Error Messages
- Database connection errors
- Invalid field values
- Host not found
- Duplicate hosts
- Missing required fields

### 11. Production-Ready Features

#### Security
- SSL/TLS required for database connections
- No credentials in code
- Environment variable configuration
- No SQL injection (prepared statements)

#### Reliability
- Connection retry logic
- Transaction support
- Proper error handling
- Graceful degradation

#### Maintainability
- Clean code structure
- Modular design
- Comprehensive comments
- Type safety (Go)

#### Observability
- Verbose mode for debugging
- Clear success/error messages
- Structured logging capability
- Audit trail for status changes

## Use Cases

### 1. Infrastructure Management
- Track all servers across multiple clouds
- Monitor host status and environment
- Manage host lifecycle (build → active → decommissioned)

### 2. Cost Management
- Track infrastructure costs per host
- Generate cost reports by environment/provider
- Identify expensive resources

### 3. Compliance and Auditing
- Track host owners and contacts
- Maintain status change history
- Link to external compliance tools

### 4. Automation
- JSON output for scripting
- Batch operations support
- Integration with CI/CD pipelines
- Webhook notifications (via external tools)

### 5. Reporting
- Generate inventory reports
- Export data for analytics
- Monitor infrastructure growth
- Track resource utilization

## Integration Examples

### With jq
```bash
# Filter active production servers
hostctl list --json | jq '.[] | select(.status == "active" and .environment == "production")'

# Calculate total monthly cost
hostctl list --json | jq '[.[].average_monthly_cost // 0] | add'

# Group by provider
hostctl list --json | jq 'group_by(.provider) | map({provider: .[0].provider, count: length})'
```

### With Shell Scripts
```bash
# Bulk status update
for host in $(hostctl list --env staging --json | jq -r '.[].hostname'); do
  hostctl status $host maintenance
done

# Export to CSV
hostctl list --json | jq -r '.[] | [.hostname, .type, .status, .environment] | @csv'
```

### With Ansible
```yaml
- name: Get active production hosts
  shell: hostctl list --env production --status active --json
  register: prod_hosts

- name: Process each host
  debug:
    msg: "{{ item.hostname }}"
  loop: "{{ prod_hosts.stdout | from_json }}"
```

### With Terraform
```bash
# Pre-deployment check
terraform plan
if [ $? -eq 0 ]; then
  terraform apply
  # Register new resources
  hostctl add "$HOSTNAME" --ip "$IP" --type server --env production
fi
```

## Command Reference

| Command | Purpose | Example |
|---------|---------|---------|
| `add` | Add new host | `hostctl add web-01 --ip 10.0.1.1 --type server` |
| `remove` | Delete host | `hostctl remove web-01` |
| `update` | Update host fields | `hostctl update web-01 --status active` |
| `status` | Change host status | `hostctl status web-01 maintenance` |
| `list` | List hosts | `hostctl list --env production` |
| `show` | Show host details | `hostctl show web-01` |
| `search` | Search hosts | `hostctl search nginx` |

## Global Flags

| Flag | Purpose | Example |
|------|---------|---------|
| `--json` | JSON output | `hostctl list --json` |
| `--verbose` | Verbose output | `hostctl add web-01 --verbose` |
| `--help` | Show help | `hostctl --help` |

## Field Reference

### Required Fields
- `hostname` - Unique identifier
- `ip` - IP address (for add command)

### Optional Fields
- `type` - Host type (default: server)
- `provider` - Cloud provider (default: other)
- `region` - Region/datacenter
- `environment` - Environment (default: development)
- `status` - Host status (default: build)
- `owners` - Owner emails
- `mailgroups` - Mail groups
- `cost-daily` - Daily cost
- `cost-monthly` - Monthly cost
- `size` - Instance size
- `shape` - Instance shape
- `tags` - Custom JSON metadata
- `external-id` - External resource ID
- `external-url` - External resource URL

## Performance Considerations

### Optimizations
- Database indexes on frequently queried fields
- JSONB for efficient JSON storage and querying
- Connection pooling
- Prepared statements
- Result limiting (default: 100)

### Scalability
- Handles thousands of hosts
- Efficient search via PostgreSQL full-text search
- Pagination support via LIMIT
- Minimal memory footprint

## Future Enhancements

Potential future features:
- Export/import functionality
- Bulk update operations
- Host groups/clusters
- Relationship tracking (dependencies)
- Integration with monitoring tools
- API server mode
- Web UI
- Backup/restore operations
- Historical cost tracking
- Resource allocation tracking
- Custom report generation
