# After Dark Systems Change Management

A comprehensive CLI toolkit and API for enterprise change management operations, featuring multi-industry compliance support, approval workflows, and audit trails.

## Features

### Core Capabilities
- **Ticket Management**: Create, View, List, Edit, Open, Close tickets
- **Multi-Industry Support**: Healthcare, IT, Government, Insurance, Finance
- **Regulatory Compliance**: GLBA, SOX, HIPAA, Banking Secrecy Act, GDPR, Custom

### Authentication
- After Dark Systems Central Auth (OAuth2/OIDC)
- Google OAuth2
- Passkeys/WebAuthn (FIDO2)
- Email/Password with MFA (TOTP)

### Approval Workflow
- Multiple approval types: Operations, IT, Risk, Change Management Board, AI Ops, Security, Network Engineering, Cloud
- Email notifications with one-click approval links
- Actions: Approve, Deny, Request Update
- Sequential or parallel approval workflows

### Compliance & Audit
- All work requests saved permanently (immutable)
- Complete audit trail with revision history
- GDPR data anonymization support
- Compliance-specific templates

## Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- AWS Account (for S3, SES, SQS)

### Installation

```bash
# Clone the repository
git clone https://github.com/afterdarksys/adsops-utils.git
cd adsops-utils

# Install dependencies
make deps

# Copy and configure environment
cp .env.example .env
# Edit .env with your configuration

# Run database migrations
make migrate-up

# Build all binaries
make build

# Or run directly
make run-api    # Start API server
make run-cli    # Run CLI commands
```

### CLI Usage

```bash
# Initialize CLI configuration
changes config init

# Login
changes auth login

# Create a ticket
changes ticket create

# List tickets
changes ticket list

# View a ticket
changes ticket view CHG-2025-00001

# Submit for approval
changes ticket submit CHG-2025-00001

# Manage approvals
changes approval list
changes approval approve CHG-2025-00001

# Close a ticket
changes ticket close CHG-2025-00001
```

## Project Structure

```
adsops-utils/
├── cmd/
│   ├── api/           # API server entry point
│   ├── cli/           # CLI tool entry point
│   ├── worker/        # Background worker entry point
│   └── migrate/       # Database migration tool
├── internal/
│   ├── config/        # Configuration management
│   ├── models/        # Domain models
│   ├── repository/    # Data access layer (PostgreSQL, Redis)
│   ├── service/       # Business logic
│   ├── api/           # HTTP handlers and middleware
│   ├── cli/           # CLI commands
│   └── pkg/           # Shared utilities
├── migrations/        # Database migrations
├── deployments/       # Infrastructure as Code (Terraform, Docker)
├── templates/         # Email templates
├── docs/              # Documentation
└── tests/             # Test suites
```

## API Endpoints

### Authentication
- `POST /v1/auth/login` - Email/password login
- `POST /v1/auth/login/mfa` - MFA verification
- `POST /v1/auth/login/oauth2/google` - Google OAuth2
- `POST /v1/auth/login/oauth2/afterdark` - After Dark Central Auth
- `POST /v1/auth/login/passkey/begin` - WebAuthn begin
- `POST /v1/auth/login/passkey/finish` - WebAuthn finish
- `POST /v1/auth/refresh` - Refresh token
- `POST /v1/auth/logout` - Logout
- `GET /v1/auth/me` - Current user

### Tickets
- `POST /v1/tickets` - Create ticket
- `GET /v1/tickets` - List tickets
- `GET /v1/tickets/:id` - Get ticket
- `PATCH /v1/tickets/:id` - Update ticket
- `POST /v1/tickets/:id/submit` - Submit for approval
- `POST /v1/tickets/:id/cancel` - Cancel ticket
- `POST /v1/tickets/:id/close` - Close ticket
- `POST /v1/tickets/:id/reopen` - Reopen ticket

### Approvals
- `GET /v1/approvals` - List pending approvals
- `GET /v1/approvals/:id` - Get approval
- `POST /v1/approvals/:id/approve` - Approve
- `POST /v1/approvals/:id/deny` - Deny
- `POST /v1/approvals/:id/request-update` - Request update
- `POST /v1/approvals/token/:token/approve` - Approve via email link
- `POST /v1/approvals/token/:token/deny` - Deny via email link

### Health & Metrics
- `GET /health` - Basic health check
- `GET /health/ready` - Readiness probe
- `GET /metrics` - Prometheus metrics (internal only)

## Configuration

Configuration can be provided via:
1. Environment variables (prefixed with `ADSOPS_`)
2. Config file (`config.yaml` or `~/.adsops-utils/config.yaml`)

See `.env.example` and `config.yaml.example` for all options.

## Development

```bash
# Install development tools
make install-tools

# Run with live reload
make dev

# Run tests
make test

# Run linter
make lint

# Generate coverage report
make test-coverage
```

## Deployment

### Docker

```bash
# Build Docker images
make docker-build

# Run with Docker Compose
make docker-compose-up
```

### Terraform (AWS)

```bash
cd deployments/terraform

# Initialize
terraform init

# Plan
terraform plan -var-file=environments/prod.tfvars

# Apply
terraform apply -var-file=environments/prod.tfvars
```

## Security

- TLS 1.3 for all connections
- JWT with short-lived access tokens (15 min)
- Row-level security for multi-tenancy
- Audit logging for all operations
- Encryption at rest (AWS KMS)
- Rate limiting per IP and user

## Compliance

This system is designed to support:
- **HIPAA** - Healthcare data protection
- **SOX** - Financial reporting controls
- **GLBA** - Financial customer data protection
- **GDPR** - EU data privacy (including right to be forgotten)
- **Banking Secrecy Act** - Financial transaction monitoring

## License

Proprietary - After Dark Systems

## Support

- Email: support@afterdarksys.com
- Portal: https://changes.afterdarksys.com
