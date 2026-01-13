-- Add API keys table for programmatic access
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- API key metadata
    name VARCHAR(255) NOT NULL, -- User-friendly name (e.g., "Production CI/CD")
    key_hash VARCHAR(255) NOT NULL UNIQUE, -- bcrypt hash of the API key
    key_prefix VARCHAR(16) NOT NULL, -- First 8 chars for identification (e.g., "chg_1234")

    -- Permissions and scope
    scopes TEXT[] DEFAULT ARRAY['tickets:read', 'tickets:write'], -- What the key can do
    rate_limit_rpm INTEGER DEFAULT 60, -- Requests per minute

    -- Status and tracking
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ,
    last_used_ip INET,
    usage_count BIGINT DEFAULT 0,

    -- Expiration
    expires_at TIMESTAMPTZ, -- NULL = never expires

    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_ip INET,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id),
    revoke_reason TEXT,

    CONSTRAINT valid_name CHECK (LENGTH(name) >= 1),
    CONSTRAINT valid_scopes CHECK (array_length(scopes, 1) > 0)
);

-- Index for fast lookup by key_hash
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash) WHERE is_active = true AND (expires_at IS NULL OR expires_at > NOW());

-- Index for user's keys
CREATE INDEX idx_api_keys_user ON api_keys(user_id, organization_id) WHERE is_active = true;

-- Index for cleanup of expired keys
CREATE INDEX idx_api_keys_expired ON api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- Audit table for API key usage
CREATE TABLE IF NOT EXISTS api_key_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,

    -- Request details
    endpoint VARCHAR(500) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT,

    -- Timing
    request_time TIMESTAMPTZ DEFAULT NOW(),
    response_time_ms INTEGER,

    -- Error tracking
    error_message TEXT,

    -- Partitioning key (for future table partitioning)
    month_partition VARCHAR(7) GENERATED ALWAYS AS (to_char(request_time, 'YYYY-MM')) STORED
);

-- Index for audit queries
CREATE INDEX idx_api_key_audit_key ON api_key_audit(api_key_id, request_time DESC);
CREATE INDEX idx_api_key_audit_partition ON api_key_audit(month_partition);

-- Function to update last_used tracking
CREATE OR REPLACE FUNCTION update_api_key_usage()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE api_keys
    SET
        last_used_at = NEW.request_time,
        last_used_ip = NEW.ip_address,
        usage_count = usage_count + 1,
        updated_at = NOW()
    WHERE id = NEW.api_key_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_api_key_usage
    AFTER INSERT ON api_key_audit
    FOR EACH ROW
    EXECUTE FUNCTION update_api_key_usage();

-- Function to enforce 5 active keys per user limit
CREATE OR REPLACE FUNCTION enforce_api_key_limit()
RETURNS TRIGGER AS $$
DECLARE
    active_count INTEGER;
BEGIN
    -- Count active, non-expired keys for this user
    SELECT COUNT(*) INTO active_count
    FROM api_keys
    WHERE user_id = NEW.user_id
      AND is_active = true
      AND (expires_at IS NULL OR expires_at > NOW())
      AND revoked_at IS NULL;

    IF active_count >= 5 THEN
        RAISE EXCEPTION 'Maximum of 5 active API keys per user. Please revoke an existing key first.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_enforce_api_key_limit
    BEFORE INSERT ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION enforce_api_key_limit();

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON api_keys TO adsops_app;
GRANT SELECT, INSERT ON api_key_audit TO adsops_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO adsops_app;

-- Add OAuth2 integration columns to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS oauth_provider VARCHAR(50); -- 'afterdark', 'google', etc.
ALTER TABLE users ADD COLUMN IF NOT EXISTS oauth_subject VARCHAR(255); -- Subject from OAuth2 token
ALTER TABLE users ADD COLUMN IF NOT EXISTS oauth_picture_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS oauth_last_sync TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_oauth ON users(oauth_provider, oauth_subject) WHERE oauth_subject IS NOT NULL;

-- Comments for documentation
COMMENT ON TABLE api_keys IS 'API keys for programmatic access to the Changes API. Each AfterDark account holder gets up to 5 keys.';
COMMENT ON COLUMN api_keys.key_hash IS 'bcrypt hash of the API key. The actual key is never stored.';
COMMENT ON COLUMN api_keys.key_prefix IS 'First 8 characters of the key for user identification (e.g., chg_Ab12Cd34)';
COMMENT ON COLUMN api_keys.scopes IS 'Array of permissions: tickets:read, tickets:write, approvals:approve, etc.';
COMMENT ON COLUMN api_keys.rate_limit_rpm IS 'Requests per minute allowed for this key';
