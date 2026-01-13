# Changelog

All notable changes to the Blackout Tool will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-01-13

### Added
- Initial release of blackout maintenance mode management tool
- Core commands: start, end, list, show, extend
- PostgreSQL database backend with automatic schema creation
- UTC timezone support for all timestamps
- JSON export for monitoring integration (`/var/lib/adsops/active-blackouts.json`)
- Host status management in `inventory_resources` table
- Audit trail with ticket numbers, timestamps, and user tracking
- Duration parsing supporting multiple formats (H:MM, H, MMm)
- Auto-expiration of blackouts past their end time
- Shorthand syntax support for quick blackout creation
- Comprehensive error handling and validation
- Database connection via environment variables
- Automatic table and index creation on first run

### Documentation
- README.md with full usage guide and examples
- INSTALL.md with detailed installation instructions
- QUICKSTART.md for rapid onboarding
- schema.sql with annotated database schema
- Inline help text with examples
- Example scripts for Ansible, monitoring, and maintenance workflows

### Integration Examples
- Python monitoring integration script
- Ansible playbook with blackout workflow
- Bash maintenance wrapper script
- Systemd service and timer for auto-cleanup

### Testing
- Comprehensive integration test suite (test.sh)
- Tests for all major commands and workflows
- JSON export validation
- Database connectivity tests

### Build System
- Go modules setup (go.mod, go.sum)
- Makefile with build, install, clean, test targets
- Binary optimization with stripped symbols

### Compliance
- Audit trail for SOX, HIPAA, GDPR, GLBA compliance
- Required ticket numbers for change management
- Reason tracking for all blackouts
- Username and timestamp recording
- Complete blackout history retention

## [Unreleased]

### Planned Features
- Web UI for blackout management
- Slack/Teams notifications on blackout start/end
- Blackout scheduling (future start times)
- Recurring blackout support (maintenance windows)
- Blackout templates for common scenarios
- API server for programmatic access
- Multi-host blackout operations (batch mode)
- Approval workflow integration
- Integration with PagerDuty, Opsgenie
- Metrics export (Prometheus format)
- Advanced reporting and analytics
- Role-based access control (RBAC)
- Blackout calendar view
- Email notifications
- SMS/voice call integration
- Mobile app

## Version History

### Version Numbering

- **Major (X.0.0)**: Breaking changes, major features
- **Minor (1.X.0)**: New features, backwards compatible
- **Patch (1.0.X)**: Bug fixes, minor improvements

### Support Policy

- **Latest version**: Full support
- **Previous minor**: Security fixes only
- **Older versions**: No support (please upgrade)

## Migration Guides

### Future Migrations

Migration guides will be provided here when breaking changes are introduced.

## Contributors

- After Dark Systems Infrastructure Team
- Ryan (Lead Developer)

## License

Copyright (c) 2024 After Dark Systems. All rights reserved.
Internal tool - proprietary license.
