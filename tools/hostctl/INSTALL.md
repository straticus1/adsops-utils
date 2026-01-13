# hostctl Installation Guide

## Prerequisites

- Go 1.21 or later
- Access to the inventory PostgreSQL database at afterdarksys.com
- Database credentials (username and password)

## Quick Installation

### Option 1: Build and Install Locally

```bash
cd /Users/ryan/development/adsops-utils/tools/hostctl

# Download dependencies
make deps

# Build the binary
make build

# Install to /usr/local/bin (requires sudo)
make install

# OR install to ~/bin (no sudo required)
make install-user
```

### Option 2: Direct Go Install

```bash
cd /Users/ryan/development/adsops-utils/tools/hostctl
go install .
```

This will install the binary to `$GOPATH/bin` (usually `~/go/bin`).

### Option 3: Manual Build

```bash
cd /Users/ryan/development/adsops-utils/tools/hostctl
go mod download
go build -o hostctl .
sudo mv hostctl /usr/local/bin/
```

## Database Configuration

Create a `.env` file or add these to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
export INVENTORY_DB_HOST="afterdarksys.com"
export INVENTORY_DB_PORT="5432"
export INVENTORY_DB_NAME="inventory"
export INVENTORY_DB_USER="your_username"
export INVENTORY_DB_PASSWORD="your_password"
```

Then load the environment:

```bash
# If using .env file
source .env

# If added to shell profile
source ~/.bashrc  # or ~/.zshrc
```

## Verify Installation

```bash
# Check if hostctl is in PATH
which hostctl

# Test the command
hostctl --help

# Test database connection
hostctl list --limit 1
```

## Database Schema

The tool expects the following table structure:

```sql
CREATE TABLE inventory_resources (
    id SERIAL PRIMARY KEY,
    resource_name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    region VARCHAR(100),
    status VARCHAR(50) NOT NULL,
    environment VARCHAR(50) NOT NULL,
    owners TEXT[], -- Array of email addresses
    mailgroups TEXT[], -- Array of mail group names
    metadata JSONB, -- Flexible metadata storage
    average_daily_cost DECIMAL(10,2),
    average_monthly_cost DECIMAL(10,2),
    external_id VARCHAR(255),
    external_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Optional: Status change tracking table (created automatically)
CREATE TABLE status_changes (
    id SERIAL PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    old_status VARCHAR(50) NOT NULL,
    new_status VARCHAR(50) NOT NULL,
    changed_at TIMESTAMP NOT NULL,
    changed_by VARCHAR(255)
);

-- Indexes for better performance
CREATE INDEX idx_hostname ON inventory_resources(hostname);
CREATE INDEX idx_status ON inventory_resources(status);
CREATE INDEX idx_environment ON inventory_resources(environment);
CREATE INDEX idx_type ON inventory_resources(type);
CREATE INDEX idx_provider ON inventory_resources(provider);
CREATE INDEX idx_metadata ON inventory_resources USING gin(metadata);
```

## Troubleshooting

### "INVENTORY_DB_USER environment variable is required"

Make sure you've set all required environment variables:
```bash
echo $INVENTORY_DB_USER
echo $INVENTORY_DB_PASSWORD
```

If empty, source your environment file or shell profile.

### "failed to connect to database"

- Verify the database host is reachable: `ping afterdarksys.com`
- Check credentials are correct
- Ensure SSL/TLS is enabled on the database (the tool uses `sslmode=require`)
- Verify firewall rules allow connection on port 5432

### "host not found"

The hostname doesn't exist in the database. Use `hostctl list` to see available hosts.

### Permission Issues

If you get permission errors during installation:
- Use `make install-user` instead of `make install` to install to ~/bin
- Or manually move the binary: `mv hostctl ~/bin/`

## Updating

To update to the latest version:

```bash
cd /Users/ryan/development/adsops-utils/tools/hostctl
git pull
make clean
make install
```

## Uninstallation

```bash
# If installed to /usr/local/bin
sudo rm /usr/local/bin/hostctl

# If installed to ~/bin
rm ~/bin/hostctl

# If installed via go install
rm $GOPATH/bin/hostctl
```

## Next Steps

- Read [README.md](README.md) for detailed usage examples
- Try the example commands:
  ```bash
  hostctl list
  hostctl list --status active
  hostctl search web
  ```
