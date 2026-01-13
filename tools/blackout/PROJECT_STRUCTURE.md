# Blackout Tool - Project Structure

Complete overview of the blackout tool project structure.

```
blackout/
├── main.go                          # Main Go application
├── go.mod                           # Go module definition
├── go.sum                           # Go dependencies checksums
├── Makefile                         # Build automation
├── .gitignore                       # Git ignore patterns
│
├── README.md                        # Main documentation
├── QUICKSTART.md                    # Quick start guide
├── INSTALL.md                       # Installation guide
├── CHANGELOG.md                     # Version history
├── PROJECT_STRUCTURE.md             # This file
│
├── schema.sql                       # Database schema
├── test.sh                          # Integration test suite
│
├── blackout-cleanup.service         # Systemd service unit
├── blackout-cleanup.timer           # Systemd timer unit
│
└── examples/                        # Integration examples
    ├── monitoring-integration.py    # Python monitoring integration
    ├── maintenance-wrapper.sh       # Bash maintenance wrapper
    └── ansible-playbook.yml         # Ansible automation example
```

## File Descriptions

### Core Application

- **main.go** (1,125 lines)
  - Complete Go application implementing all blackout commands
  - Database connection and schema management
  - Command parsing and execution
  - JSON export for monitoring integration
  - Helper functions for duration parsing, formatting

### Build System

- **go.mod**
  - Go module definition
  - Dependency: github.com/lib/pq (PostgreSQL driver)

- **go.sum**
  - Checksums for Go dependencies
  - Ensures reproducible builds

- **Makefile**
  - Build targets: build, install, clean, test, deps
  - Easy installation with `make install`
  - Help system with `make help`

### Documentation

- **README.md** (580 lines)
  - Complete usage documentation
  - All commands with examples
  - Database schema details
  - Monitoring integration guide
  - Best practices and troubleshooting
  - Compliance information

- **QUICKSTART.md** (155 lines)
  - Fast onboarding for new users
  - Common scenarios with copy-paste examples
  - Essential commands only
  - Minimal setup instructions

- **INSTALL.md** (530 lines)
  - Detailed installation steps
  - Prerequisites and requirements
  - Database setup procedures
  - Systemd/cron configuration
  - Post-installation testing
  - Troubleshooting guide
  - Security considerations
  - Upgrade and uninstall procedures

- **CHANGELOG.md** (100 lines)
  - Version history
  - Release notes
  - Planned features
  - Migration guides

- **PROJECT_STRUCTURE.md** (This file)
  - Project organization
  - File descriptions
  - Architecture overview

### Database

- **schema.sql** (220 lines)
  - Table definitions (inventory_resources, inventory_blackouts)
  - Indices for performance
  - Views for common queries
  - Helper functions
  - Sample data (commented)
  - Comprehensive comments

### Testing

- **test.sh** (320 lines)
  - Integration test suite
  - Tests all major commands
  - Database connectivity tests
  - JSON export validation
  - Cleanup verification
  - Color-coded output

### Systemd Integration

- **blackout-cleanup.service**
  - Systemd service unit
  - Runs cleanup command
  - Environment variable support
  - Logging configuration

- **blackout-cleanup.timer**
  - Systemd timer unit
  - Runs every 5 minutes
  - Auto-starts on boot
  - Triggers cleanup service

### Examples

- **examples/monitoring-integration.py** (200 lines)
  - Python integration example
  - Reads JSON export file
  - Provides helper functions
  - Command-line interface
  - Error handling

- **examples/maintenance-wrapper.sh** (200 lines)
  - Bash wrapper for maintenance operations
  - Automatic blackout start/end
  - Error handling and cleanup
  - Logging support
  - Trap handlers

- **examples/ansible-playbook.yml** (150 lines)
  - Ansible automation example
  - Serial execution (one host at a time)
  - Blackout integration
  - Error recovery
  - Post-maintenance reporting

## Architecture

### Database Schema

```
inventory_resources          inventory_blackouts
├── id (SERIAL)              ├── id (SERIAL)
├── hostname (VARCHAR)       ├── ticket_number (VARCHAR)
├── status (VARCHAR)         ├── hostname (VARCHAR)
├── resource_type            ├── start_time (TIMESTAMP)
├── environment              ├── end_time (TIMESTAMP)
├── created_at               ├── actual_end_time (TIMESTAMP)
└── updated_at               ├── reason (TEXT)
                             ├── created_by (VARCHAR)
                             ├── status (VARCHAR)
                             └── created_at (TIMESTAMP)
```

### Data Flow

```
User Command
    ↓
Command Parser
    ↓
Database Operations
    ├── Update inventory_resources.status
    ├── Insert/Update inventory_blackouts
    └── Query active blackouts
    ↓
JSON Export
    ├── Query active blackouts
    ├── Format as JSON
    └── Write to /var/lib/adsops/active-blackouts.json
    ↓
Monitoring System
    ├── Read JSON file
    ├── Check hostname
    └── Suppress alerts if in blackout
```

### Command Flow

#### Start Blackout
1. Parse duration string → time.Duration
2. Get current user (os/user)
3. Calculate end_time (start + duration)
4. Begin database transaction
5. Insert into inventory_blackouts
6. Update inventory_resources.status = 'blackout'
7. Commit transaction
8. Export active blackouts to JSON
9. Display confirmation

#### End Blackout
1. Query active blackout for hostname
2. Verify blackout exists and is active
3. Begin database transaction
4. Update blackout status = 'completed'
5. Set actual_end_time = now
6. Update inventory_resources.status = 'active'
7. Commit transaction
8. Export active blackouts to JSON
9. Display summary

#### Cleanup
1. Query blackouts where end_time < now and status = 'active'
2. Update status = 'expired'
3. Find hosts with no active blackouts
4. Update their status to 'active'
5. Export active blackouts to JSON
6. Display counts

## Code Organization

### main.go Structure

```go
// Constants
const (
    defaultDBHost, defaultDBPort, ...
    blackoutJSONPath
    version
)

// Types
type Blackout struct { ... }
type ActiveBlackoutExport struct { ... }
type DB struct { ... }

// Main entry point
func main()

// Command handlers
func handleStart(db, ticket, hostname, duration, reason)
func handleEnd(db, hostname)
func handleList(db, activeOnly, hostname)
func handleShow(db, hostname)
func handleExtend(db, hostname, duration)
func handleCleanup(db)

// Database operations
func connectDB() (*DB, error)
func (db *DB) ensureSchema() error
func (db *DB) exportActiveBlackouts() error

// Helper functions
func parseDuration(s string) (time.Duration, error)
func formatDuration(d time.Duration) string
func getCurrentUser() string
func getEnv(key, defaultValue) string
func printUsage()
```

## Dependencies

### Runtime Dependencies

- **PostgreSQL** (12+)
  - Database storage
  - ACID transactions
  - Time zone support

- **Go Runtime** (1.21+)
  - For compiled binary execution
  - Not needed after compilation

### Build Dependencies

- **Go** (1.21+)
  - Compiler and toolchain
  - Module management

- **github.com/lib/pq**
  - PostgreSQL driver for Go
  - Pure Go implementation

### Optional Dependencies

- **jq** - JSON validation in tests
- **systemd** - Auto-cleanup timer
- **psql** - Manual database operations
- **python3** - Monitoring integration example
- **ansible** - Automation example

## Configuration

### Environment Variables

```bash
DB_HOST=localhost           # PostgreSQL host
DB_PORT=5432                # PostgreSQL port
DB_NAME=apiproxy            # Database name
DB_USER=apiproxy            # Database user
DB_PASSWORD=secret          # Database password
```

### Files

- `/var/lib/adsops/active-blackouts.json` - JSON export (created automatically)
- `/etc/blackout/config.env` - Optional system config
- `~/.bashrc` - User environment variables

### Database Tables

- `inventory_resources` - Host inventory
- `inventory_blackouts` - Blackout history
- `active_blackouts` (view) - Active blackouts
- `blackout_summary` (view) - Statistics

## Monitoring Integration

### JSON Export Format

```json
[
  {
    "hostname": "api-server-1",
    "ticket": "CHG-2024-001",
    "end_time": "2024-01-13T18:30:00Z",
    "reason": "Database migration",
    "remaining_time": "1h 25m"
  }
]
```

### Integration Points

1. **File-based**: Read `/var/lib/adsops/active-blackouts.json`
2. **Database**: Query `active_blackouts` view
3. **CLI**: Use `blackout list --active` in scripts

## Building and Installation

### Quick Build

```bash
make build        # Build binary
./blackout help   # Test
```

### Full Installation

```bash
make install                     # Build and install to /usr/local/bin
sudo systemctl enable blackout-cleanup.timer  # Enable auto-cleanup
```

### Development

```bash
make deps         # Download dependencies
make test         # Run tests
make clean        # Clean build artifacts
```

## Testing

### Run Tests

```bash
./test.sh                # Run all tests
VERBOSE=1 ./test.sh      # Verbose output
```

### Test Coverage

- Command execution (start, end, list, show, extend)
- Database operations (create, update, query)
- JSON export validation
- Error handling
- Duration parsing
- Cleanup operations

## Future Enhancements

See CHANGELOG.md "Unreleased" section for planned features.

### Priorities

1. **Web UI** - User-friendly interface
2. **API Server** - RESTful API
3. **Notifications** - Slack, Teams integration
4. **Scheduling** - Future blackouts
5. **Metrics** - Prometheus export

## License

Copyright (c) 2024 After Dark Systems
Internal tool - All rights reserved

## Support

- GitHub: https://github.com/afterdarksystems/adsops-utils
- Slack: #infrastructure
- Email: ops@afterdarksys.com
