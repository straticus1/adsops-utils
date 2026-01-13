-- Drop triggers
DROP TRIGGER IF EXISTS trigger_enforce_api_key_limit ON api_keys;
DROP TRIGGER IF EXISTS trigger_update_api_key_usage ON api_key_audit;

-- Drop functions
DROP FUNCTION IF EXISTS enforce_api_key_limit();
DROP FUNCTION IF EXISTS update_api_key_usage();

-- Drop tables
DROP TABLE IF EXISTS api_key_audit CASCADE;
DROP TABLE IF EXISTS api_keys CASCADE;

-- Remove OAuth2 columns from users
ALTER TABLE users DROP COLUMN IF EXISTS oauth_provider;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_subject;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_picture_url;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_last_sync;
