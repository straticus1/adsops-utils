-- Blackout Tool Database Schema
-- This schema is automatically created by the tool on first run
-- You can also manually apply it with: psql -h localhost -U apiproxy -d apiproxy -f schema.sql

-- =============================================================================
-- INVENTORY RESOURCES TABLE
-- Tracks all infrastructure hosts and their current status
-- =============================================================================
CREATE TABLE IF NOT EXISTS inventory_resources (
    id SERIAL PRIMARY KEY,
    hostname VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    resource_type VARCHAR(100),           -- e.g., 'vm', 'container', 'physical'
    environment VARCHAR(50),              -- e.g., 'production', 'staging', 'development'
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indices for performance
CREATE INDEX IF NOT EXISTS idx_resources_hostname ON inventory_resources(hostname);
CREATE INDEX IF NOT EXISTS idx_resources_status ON inventory_resources(status);

COMMENT ON TABLE inventory_resources IS 'Infrastructure inventory with current status';
COMMENT ON COLUMN inventory_resources.status IS 'Current status: active, blackout, inactive';

-- =============================================================================
-- INVENTORY BLACKOUTS TABLE
-- Tracks all maintenance/blackout windows with full audit trail
-- =============================================================================
CREATE TABLE IF NOT EXISTS inventory_blackouts (
    id SERIAL PRIMARY KEY,
    ticket_number VARCHAR(50) NOT NULL,   -- Change ticket (CHG-*, INC-*)
    hostname VARCHAR(255) NOT NULL,
    start_time TIMESTAMP NOT NULL,        -- UTC timestamp
    end_time TIMESTAMP NOT NULL,          -- Scheduled end time (UTC)
    actual_end_time TIMESTAMP,            -- Actual end time if ended early
    reason TEXT,                          -- Description of maintenance
    created_by VARCHAR(255),              -- Unix username
    status VARCHAR(50) DEFAULT 'active',  -- active, completed, expired
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indices for common queries
CREATE INDEX IF NOT EXISTS idx_blackouts_hostname ON inventory_blackouts(hostname);
CREATE INDEX IF NOT EXISTS idx_blackouts_status ON inventory_blackouts(status);
CREATE INDEX IF NOT EXISTS idx_blackouts_end_time ON inventory_blackouts(end_time);
CREATE INDEX IF NOT EXISTS idx_blackouts_ticket ON inventory_blackouts(ticket_number);
CREATE INDEX IF NOT EXISTS idx_blackouts_created_by ON inventory_blackouts(created_by);

-- Composite index for active blackout lookups
CREATE INDEX IF NOT EXISTS idx_blackouts_active_lookup
    ON inventory_blackouts(hostname, status, end_time)
    WHERE status = 'active';

COMMENT ON TABLE inventory_blackouts IS 'Maintenance/blackout windows with full audit trail';
COMMENT ON COLUMN inventory_blackouts.status IS 'Blackout status: active, completed (manual end), expired (auto end)';
COMMENT ON COLUMN inventory_blackouts.ticket_number IS 'Change management ticket number (required for compliance)';

-- =============================================================================
-- HELPER VIEWS
-- =============================================================================

-- View: Active blackouts (currently in effect)
CREATE OR REPLACE VIEW active_blackouts AS
SELECT
    id,
    ticket_number,
    hostname,
    start_time,
    end_time,
    reason,
    created_by,
    EXTRACT(EPOCH FROM (end_time - NOW())) / 60 AS minutes_remaining,
    created_at
FROM inventory_blackouts
WHERE status = 'active'
  AND end_time > NOW()
ORDER BY end_time ASC;

COMMENT ON VIEW active_blackouts IS 'Currently active blackouts with time remaining';

-- View: Blackout history summary
CREATE OR REPLACE VIEW blackout_summary AS
SELECT
    hostname,
    COUNT(*) AS total_blackouts,
    COUNT(*) FILTER (WHERE status = 'active') AS active_count,
    MAX(start_time) AS last_blackout_start,
    SUM(EXTRACT(EPOCH FROM (COALESCE(actual_end_time, end_time) - start_time))) / 3600 AS total_hours,
    AVG(EXTRACT(EPOCH FROM (COALESCE(actual_end_time, end_time) - start_time))) / 3600 AS avg_hours
FROM inventory_blackouts
GROUP BY hostname
ORDER BY total_blackouts DESC;

COMMENT ON VIEW blackout_summary IS 'Summary statistics per hostname';

-- =============================================================================
-- MAINTENANCE FUNCTIONS
-- =============================================================================

-- Function: Auto-expire blackouts
CREATE OR REPLACE FUNCTION auto_expire_blackouts()
RETURNS INTEGER AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    WITH updated AS (
        UPDATE inventory_blackouts
        SET status = 'expired'
        WHERE status = 'active'
          AND end_time < NOW()
        RETURNING id
    )
    SELECT COUNT(*) INTO expired_count FROM updated;

    -- Restore host status if no active blackouts remain
    UPDATE inventory_resources
    SET status = 'active', updated_at = NOW()
    WHERE status = 'blackout'
      AND hostname NOT IN (
          SELECT hostname
          FROM inventory_blackouts
          WHERE status = 'active' AND end_time > NOW()
      );

    RETURN expired_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION auto_expire_blackouts() IS 'Automatically expire blackouts past their end_time';

-- Function: Get active blackout for a host
CREATE OR REPLACE FUNCTION get_active_blackout(p_hostname VARCHAR)
RETURNS TABLE(
    id INTEGER,
    ticket_number VARCHAR,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    reason TEXT,
    minutes_remaining NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        b.id,
        b.ticket_number,
        b.start_time,
        b.end_time,
        b.reason,
        EXTRACT(EPOCH FROM (b.end_time - NOW())) / 60 AS minutes_remaining
    FROM inventory_blackouts b
    WHERE b.hostname = p_hostname
      AND b.status = 'active'
      AND b.end_time > NOW()
    ORDER BY b.start_time DESC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_active_blackout(VARCHAR) IS 'Get current active blackout for a hostname';

-- =============================================================================
-- SAMPLE DATA (for testing)
-- =============================================================================

-- Uncomment to insert sample data:
/*
INSERT INTO inventory_resources (hostname, status, resource_type, environment) VALUES
    ('api-server-1', 'active', 'vm', 'production'),
    ('api-server-2', 'active', 'vm', 'production'),
    ('db-primary-1', 'active', 'vm', 'production'),
    ('web-server-1', 'active', 'vm', 'production')
ON CONFLICT (hostname) DO NOTHING;
*/

-- =============================================================================
-- SUCCESS MESSAGE
-- =============================================================================
DO $$
BEGIN
    RAISE NOTICE '✅ Blackout tool schema initialized successfully!';
    RAISE NOTICE '📊 Tables: inventory_resources, inventory_blackouts';
    RAISE NOTICE '👁️  Views: active_blackouts, blackout_summary';
    RAISE NOTICE '🔧 Functions: auto_expire_blackouts(), get_active_blackout()';
    RAISE NOTICE '';
    RAISE NOTICE 'Next steps:';
    RAISE NOTICE '  1. Build the tool: make build';
    RAISE NOTICE '  2. Install: make install';
    RAISE NOTICE '  3. Test: blackout help';
END $$;
