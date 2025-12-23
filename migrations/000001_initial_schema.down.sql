-- Rollback migration: 000001_initial_schema

-- Drop functions
DROP FUNCTION IF EXISTS anonymize_user_data(UUID);
DROP FUNCTION IF EXISTS check_approval_completion(UUID);
DROP FUNCTION IF EXISTS generate_ticket_number(UUID, INTEGER);
DROP FUNCTION IF EXISTS update_timestamp();
DROP FUNCTION IF EXISTS prevent_ticket_deletion();

-- Drop views
DROP VIEW IF EXISTS v_pending_approvals;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS compliance_templates CASCADE;
DROP TABLE IF EXISTS notification_queue CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS audit_log CASCADE;
DROP TABLE IF EXISTS ticket_comments CASCADE;
DROP TABLE IF EXISTS ticket_revisions CASCADE;
DROP TABLE IF EXISTS approvals CASCADE;
DROP TABLE IF EXISTS change_tickets CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;

-- Drop enum types
DROP TYPE IF EXISTS risk_level;
DROP TYPE IF EXISTS ticket_priority;
DROP TYPE IF EXISTS approval_status;
DROP TYPE IF EXISTS approval_type;
DROP TYPE IF EXISTS ticket_status;
DROP TYPE IF EXISTS compliance_framework;
DROP TYPE IF EXISTS industry_type;

-- Drop extensions (optional - may be used by other databases)
-- DROP EXTENSION IF EXISTS btree_gist;
-- DROP EXTENSION IF EXISTS pgcrypto;
-- DROP EXTENSION IF EXISTS "uuid-ossp";
