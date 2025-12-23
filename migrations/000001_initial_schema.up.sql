-- =====================================================
-- CORE SCHEMA: Change Management System
-- Compliance: HIPAA, SOX, GLBA, GDPR, BSA
-- =====================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- =====================================================
-- ENUM TYPES
-- =====================================================

CREATE TYPE industry_type AS ENUM (
    'healthcare',
    'it',
    'government',
    'insurance',
    'finance'
);

CREATE TYPE compliance_framework AS ENUM (
    'glba',
    'sox',
    'hipaa',
    'banking_secrecy_act',
    'gdpr',
    'custom'
);

CREATE TYPE ticket_status AS ENUM (
    'draft',
    'submitted',
    'in_review',
    'approved',
    'partially_approved',
    'denied',
    'update_requested',
    'implementing',
    'completed',
    'closed',
    'cancelled'
);

CREATE TYPE approval_type AS ENUM (
    'operations',
    'it',
    'risk',
    'change_management_board',
    'ai_ops',
    'security',
    'network_engineering',
    'cloud'
);

CREATE TYPE approval_status AS ENUM (
    'pending',
    'approved',
    'denied',
    'update_requested',
    'expired'
);

CREATE TYPE ticket_priority AS ENUM (
    'emergency',
    'urgent',
    'high',
    'normal',
    'low'
);

CREATE TYPE risk_level AS ENUM (
    'critical',
    'high',
    'medium',
    'low'
);

-- =====================================================
-- ORGANIZATIONS & TENANCY
-- =====================================================

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    industry industry_type NOT NULL,
    compliance_frameworks compliance_framework[] NOT NULL DEFAULT '{}',
    custom_compliance_spec JSONB,
    primary_region VARCHAR(50) NOT NULL DEFAULT 'us-east-1',
    data_residency_requirements JSONB,
    require_mfa BOOLEAN DEFAULT true,
    session_timeout_minutes INTEGER DEFAULT 30,
    password_policy JSONB,
    admin_email VARCHAR(255) NOT NULL,
    support_email VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    created_by UUID,
    updated_by UUID
);

CREATE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_industry ON organizations(industry);
CREATE INDEX idx_organizations_deleted_at ON organizations(deleted_at) WHERE deleted_at IS NULL;

-- =====================================================
-- USERS & AUTHENTICATION
-- =====================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    email VARCHAR(255) NOT NULL,
    username VARCHAR(100),
    full_name VARCHAR(255) NOT NULL,
    password_hash TEXT,
    require_password_change BOOLEAN DEFAULT false,
    mfa_enabled BOOLEAN DEFAULT false,
    mfa_secret TEXT,
    backup_codes TEXT[],
    webauthn_credentials JSONB DEFAULT '[]',
    oauth_provider VARCHAR(50),
    oauth_subject VARCHAR(255),
    roles VARCHAR(50)[] DEFAULT '{"user"}',
    is_approver BOOLEAN DEFAULT false,
    approval_types approval_type[] DEFAULT '{}',
    approval_delegate_id UUID REFERENCES users(id),
    is_active BOOLEAN DEFAULT true,
    email_verified BOOLEAN DEFAULT false,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    accepts_terms_version VARCHAR(20),
    accepts_terms_at TIMESTAMPTZ,
    data_retention_consent BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(organization_id, email),
    UNIQUE(organization_id, username)
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_organization_id ON users(organization_id);
CREATE INDEX idx_users_is_approver ON users(is_approver) WHERE is_approver = true;
CREATE INDEX idx_users_oauth_provider_subject ON users(oauth_provider, oauth_subject);

-- =====================================================
-- CHANGE TICKETS (IMMUTABLE AFTER SUBMISSION)
-- =====================================================

CREATE TABLE change_tickets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    ticket_number VARCHAR(50) NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    assigned_to UUID REFERENCES users(id),
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    status ticket_status NOT NULL DEFAULT 'draft',
    priority ticket_priority NOT NULL DEFAULT 'normal',
    risk_level risk_level NOT NULL DEFAULT 'medium',
    industry industry_type NOT NULL,
    compliance_frameworks compliance_framework[] NOT NULL,
    compliance_notes TEXT,
    change_type VARCHAR(100),
    affected_systems TEXT[],
    affected_data_types VARCHAR(100)[],
    impact_description TEXT,
    rollback_plan TEXT,
    testing_plan TEXT,
    requested_implementation_date TIMESTAMPTZ,
    scheduled_start TIMESTAMPTZ,
    scheduled_end TIMESTAMPTZ,
    actual_start TIMESTAMPTZ,
    actual_end TIMESTAMPTZ,
    requires_approval_types approval_type[] NOT NULL,
    approval_deadline TIMESTAMPTZ,
    attachment_urls TEXT[],
    custom_fields JSONB DEFAULT '{}',
    submitted_at TIMESTAMPTZ,
    submitted_snapshot JSONB,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    deletion_reason TEXT,
    UNIQUE(organization_id, ticket_number)
);

CREATE INDEX idx_tickets_org_status ON change_tickets(organization_id, status);
CREATE INDEX idx_tickets_created_by ON change_tickets(created_by);
CREATE INDEX idx_tickets_assigned_to ON change_tickets(assigned_to);
CREATE INDEX idx_tickets_priority ON change_tickets(priority);
CREATE INDEX idx_tickets_scheduled_start ON change_tickets(scheduled_start);
CREATE INDEX idx_tickets_compliance ON change_tickets USING GIN(compliance_frameworks);
CREATE INDEX idx_tickets_affected_systems ON change_tickets USING GIN(affected_systems);
CREATE INDEX idx_tickets_created_at ON change_tickets(created_at DESC);
CREATE INDEX idx_tickets_search ON change_tickets USING gin(
    to_tsvector('english', title || ' ' || description)
);
CREATE INDEX idx_tickets_org_status_priority
    ON change_tickets(organization_id, status, priority, created_at DESC);
CREATE INDEX idx_tickets_custom_fields
    ON change_tickets USING gin(custom_fields);

-- =====================================================
-- TICKET REVISIONS (AUDIT TRAIL)
-- =====================================================

CREATE TABLE ticket_revisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id UUID NOT NULL REFERENCES change_tickets(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    revision_number INTEGER NOT NULL,
    changed_by UUID NOT NULL REFERENCES users(id),
    change_reason TEXT,
    changes JSONB NOT NULL,
    ticket_snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    UNIQUE(ticket_id, revision_number)
);

CREATE INDEX idx_revisions_ticket_id ON ticket_revisions(ticket_id);
CREATE INDEX idx_revisions_changed_by ON ticket_revisions(changed_by);
CREATE INDEX idx_revisions_created_at ON ticket_revisions(created_at DESC);

-- =====================================================
-- APPROVALS
-- =====================================================

CREATE TABLE approvals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id UUID NOT NULL REFERENCES change_tickets(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    approval_type approval_type NOT NULL,
    sequence_order INTEGER DEFAULT 0,
    approver_id UUID NOT NULL REFERENCES users(id),
    delegated_from UUID REFERENCES users(id),
    status approval_status NOT NULL DEFAULT 'pending',
    approved_at TIMESTAMPTZ,
    denied_at TIMESTAMPTZ,
    decision_comment TEXT,
    conditions TEXT,
    approval_token VARCHAR(255) UNIQUE,
    token_expires_at TIMESTAMPTZ,
    approval_ip INET,
    approval_user_agent TEXT,
    notification_sent_at TIMESTAMPTZ,
    notification_read_at TIMESTAMPTZ,
    reminder_sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(ticket_id, approval_type, approver_id)
);

CREATE INDEX idx_approvals_ticket_id ON approvals(ticket_id);
CREATE INDEX idx_approvals_approver_id ON approvals(approver_id);
CREATE INDEX idx_approvals_status ON approvals(status);
CREATE INDEX idx_approvals_token ON approvals(approval_token) WHERE approval_token IS NOT NULL;
CREATE INDEX idx_approvals_org_pending ON approvals(organization_id, status)
    WHERE status = 'pending';
CREATE INDEX idx_approvals_pending_by_approver
    ON approvals(approver_id, status, created_at)
    WHERE status = 'pending';

-- =====================================================
-- COMMENTS & COLLABORATION
-- =====================================================

CREATE TABLE ticket_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id UUID NOT NULL REFERENCES change_tickets(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    author_id UUID NOT NULL REFERENCES users(id),
    comment TEXT NOT NULL,
    is_internal BOOLEAN DEFAULT false,
    mentioned_users UUID[],
    attachment_urls TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    edited BOOLEAN DEFAULT false,
    edit_history JSONB DEFAULT '[]'
);

CREATE INDEX idx_comments_ticket_id ON ticket_comments(ticket_id, created_at);
CREATE INDEX idx_comments_author_id ON ticket_comments(author_id);

-- =====================================================
-- AUDIT LOG (COMPREHENSIVE ACTIVITY TRACKING)
-- =====================================================

CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID REFERENCES users(id),
    username VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    description TEXT NOT NULL,
    changes JSONB,
    metadata JSONB,
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    compliance_relevant BOOLEAN DEFAULT false,
    compliance_frameworks compliance_framework[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    anonymized BOOLEAN DEFAULT false,
    anonymized_at TIMESTAMPTZ
);

CREATE INDEX idx_audit_org_created ON audit_log(organization_id, created_at DESC);
CREATE INDEX idx_audit_user ON audit_log(user_id, created_at DESC);
CREATE INDEX idx_audit_resource ON audit_log(resource_type, resource_id);
CREATE INDEX idx_audit_action ON audit_log(action);
CREATE INDEX idx_audit_compliance ON audit_log(compliance_relevant)
    WHERE compliance_relevant = true;

-- =====================================================
-- SESSIONS (FOR STATEFUL AUTH)
-- =====================================================

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    session_token VARCHAR(255) UNIQUE NOT NULL,
    refresh_token VARCHAR(255) UNIQUE,
    ip_address INET,
    user_agent TEXT,
    device_fingerprint VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked BOOLEAN DEFAULT false,
    revoked_at TIMESTAMPTZ,
    revoke_reason VARCHAR(255)
);

CREATE INDEX idx_sessions_token ON sessions(session_token);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
CREATE INDEX idx_sessions_cleanup ON sessions(expires_at)
    WHERE revoked = false;

-- =====================================================
-- NOTIFICATION QUEUE
-- =====================================================

CREATE TABLE notification_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID REFERENCES users(id),
    email VARCHAR(255) NOT NULL,
    notification_type VARCHAR(50) NOT NULL,
    subject VARCHAR(500) NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT NOT NULL,
    ticket_id UUID REFERENCES change_tickets(id),
    approval_id UUID REFERENCES approvals(id),
    status VARCHAR(50) DEFAULT 'pending',
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    sent_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    error_message TEXT,
    ses_message_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_for TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notifications_status ON notification_queue(status, scheduled_for);
CREATE INDEX idx_notifications_user ON notification_queue(user_id);
CREATE INDEX idx_notifications_ticket ON notification_queue(ticket_id);

-- =====================================================
-- COMPLIANCE TEMPLATES
-- =====================================================

CREATE TABLE compliance_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    framework compliance_framework NOT NULL,
    required_fields JSONB NOT NULL,
    approval_workflow approval_type[] NOT NULL,
    risk_questions JSONB,
    is_active BOOLEAN DEFAULT true,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_templates_org_framework ON compliance_templates(organization_id, framework);

-- =====================================================
-- TRIGGERS FOR AUDIT & IMMUTABILITY
-- =====================================================

-- Prevent deletion of tickets (compliance requirement)
CREATE OR REPLACE FUNCTION prevent_ticket_deletion()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Tickets cannot be deleted per compliance requirements. Use soft delete.';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER no_ticket_deletion
    BEFORE DELETE ON change_tickets
    FOR EACH ROW
    EXECUTE FUNCTION prevent_ticket_deletion();

-- Auto-update timestamps
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_organizations_timestamp
    BEFORE UPDATE ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_users_timestamp
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_tickets_timestamp
    BEFORE UPDATE ON change_tickets
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_approvals_timestamp
    BEFORE UPDATE ON approvals
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_comments_timestamp
    BEFORE UPDATE ON ticket_comments
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_templates_timestamp
    BEFORE UPDATE ON compliance_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

-- =====================================================
-- ROW LEVEL SECURITY (MULTI-TENANCY)
-- =====================================================

ALTER TABLE change_tickets ENABLE ROW LEVEL SECURITY;
ALTER TABLE approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE ticket_comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE ticket_revisions ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;

-- =====================================================
-- VIEWS FOR COMMON QUERIES
-- =====================================================

-- Pending approvals per user
CREATE VIEW v_pending_approvals AS
SELECT
    a.*,
    t.ticket_number,
    t.title as ticket_title,
    t.priority,
    t.risk_level,
    u.full_name as approver_name,
    u.email as approver_email
FROM approvals a
JOIN change_tickets t ON a.ticket_id = t.id
JOIN users u ON a.approver_id = u.id
WHERE a.status = 'pending'
  AND (a.token_expires_at IS NULL OR a.token_expires_at > NOW())
  AND t.deleted_at IS NULL;

-- =====================================================
-- FUNCTIONS FOR COMMON OPERATIONS
-- =====================================================

-- Generate next ticket number
CREATE OR REPLACE FUNCTION generate_ticket_number(org_id UUID, year INTEGER)
RETURNS VARCHAR AS $$
DECLARE
    next_num INTEGER;
    ticket_num VARCHAR;
BEGIN
    SELECT COALESCE(MAX(
        CAST(
            SUBSTRING(ticket_number FROM '[0-9]+$') AS INTEGER
        )
    ), 0) + 1
    INTO next_num
    FROM change_tickets
    WHERE organization_id = org_id
      AND EXTRACT(YEAR FROM created_at) = year;

    ticket_num := 'CHG-' || year || '-' || LPAD(next_num::TEXT, 5, '0');
    RETURN ticket_num;
END;
$$ LANGUAGE plpgsql;

-- Check if all approvals are complete
CREATE OR REPLACE FUNCTION check_approval_completion(ticket_uuid UUID)
RETURNS BOOLEAN AS $$
DECLARE
    pending_count INTEGER;
BEGIN
    SELECT COUNT(*)
    INTO pending_count
    FROM approvals
    WHERE ticket_id = ticket_uuid
      AND status = 'pending';

    RETURN pending_count = 0;
END;
$$ LANGUAGE plpgsql;

-- Function to anonymize user data (GDPR right to be forgotten)
CREATE OR REPLACE FUNCTION anonymize_user_data(user_uuid UUID)
RETURNS VOID AS $$
BEGIN
    -- Anonymize user record
    UPDATE users
    SET
        email = 'anonymized_' || user_uuid || '@deleted.local',
        username = 'anonymized_' || user_uuid,
        full_name = 'Deleted User',
        password_hash = NULL,
        mfa_secret = NULL,
        backup_codes = NULL,
        webauthn_credentials = '[]',
        deleted_at = NOW()
    WHERE id = user_uuid;

    -- Anonymize audit log
    UPDATE audit_log
    SET
        username = 'Deleted User',
        anonymized = true,
        anonymized_at = NOW()
    WHERE user_id = user_uuid;
END;
$$ LANGUAGE plpgsql;
