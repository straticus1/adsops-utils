# Platform API Keys Bootstrap

## Overview

This script creates service account users and generates API keys for backend services to use with the Changes API.

## Platform Keys Created

1. **adsops-cli-default** - For adsops CLI tool
2. **ads-ai-staff-automation** - For AI staff automation
3. **changes-notifier-daemon** - For email notifications
4. **infrastructure-monitor** - For infrastructure monitoring
5. **deployment-ci-cd** - For CI/CD pipelines

## Usage

```bash
cd ~/development/adsops-utils/migrations

# Set database credentials
export DB_HOST="your-db-host"
export DB_USER="adsops_app"
export PGPASSWORD="your-password"
export DB_NAME="adsops_changes"

# Run bootstrap
node bootstrap-platform-keys.js
```

## Output

The script will:
1. Create a default organization (if needed): `afterdark-internal`
2. Create service account users for each platform
3. Generate API keys for each service
4. Save keys to:
   - `.platform-keys.json` - JSON format
   - `.platform-keys.env` - ENV variable format

**IMPORTANT:** These files contain sensitive API keys and are `.gitignore`d. Store them securely!

## Example Output

```
╔══════════════════════════════════════════════════════════════╗
║        Bootstrap Platform API Keys for Changes API          ║
╚══════════════════════════════════════════════════════════════╝

🏢 Setting up organization...
   Organization ID: abc123...

👥 Creating service accounts and API keys...

📦 AdsOps CLI Service Account
   ✅ Created user: adsops-service@afterdarksys.com
   ✅ Created API key: adsops-cli-default

📦 AfterDark AI Staff Service Account
   ✅ Created user: ads-ai-staff@afterdarksys.com
   ✅ Created API key: ads-ai-staff-automation

...

╔══════════════════════════════════════════════════════════════╗
║                  🔑 GENERATED API KEYS 🔑                    ║
╚══════════════════════════════════════════════════════════════╝

Service: adsops-cli-default
Email:   adsops-service@afterdarksys.com
Key:     chg_Ab12Cd34Ef56...
Scopes:  tickets:read, tickets:write, approvals:read, approvals:write
──────────────────────────────────────────────────────────────

...
```

## Storing Keys Securely

### Option 1: OCI Vault (Recommended)

```bash
# Store each key in OCI Vault
oci vault secret create-base64 \
  --compartment-id $COMPARTMENT_ID \
  --vault-id $VAULT_ID \
  --key-id $KEY_ID \
  --secret-name "changes-api-adsops-cli" \
  --secret-content-content "$(cat .platform-keys.json | jq -r '.[] | select(.service=="adsops-cli-default") | .apiKey' | base64)"
```

### Option 2: Environment Variables

```bash
# Load into environment
source .platform-keys.env

# Or add to service .env files
cat .platform-keys.env >> ~/development/adsops-utils/.env
cat .platform-keys.env >> ~/development/ads-ai-staff/.env
```

### Option 3: Kubernetes Secrets

```bash
# Create K8s secret
kubectl create secret generic changes-api-keys \
  --from-file=.platform-keys.json
```

## Using the Keys

### adsops CLI

```bash
# Add to ~/.adsops/config.yaml
changes_api:
  url: https://api.changes.afterdarksys.com
  api_key: chg_your_adsops_key_here
```

### ads-ai-staff

```bash
# Add to .env
CHANGES_API_URL=https://api.changes.afterdarksys.com
CHANGES_API_KEY=chg_your_ai_staff_key_here
```

### changes-notifier

```bash
# Add to .env
CHANGES_API_URL=https://api.changes.afterdarksys.com
CHANGES_API_KEY=chg_your_notifier_key_here
```

## Testing

```bash
# Test each key
curl -H "X-API-Key: chg_your_key_here" \
  https://api.changes.afterdarksys.com/v1/auth/me

# Should return service account details
```

## Regenerating Keys

To regenerate a key:

```sql
-- 1. Revoke old key
UPDATE api_keys
SET revoked_at = NOW(), is_active = false
WHERE name = 'adsops-cli-default';

-- 2. Re-run bootstrap script
node bootstrap-platform-keys.js
```

## Security Notes

- ✅ Keys are bcrypt hashed in database
- ✅ Keys are never logged or displayed after generation
- ✅ Output files are chmod 600 (owner read-only)
- ✅ Output files are .gitignore'd
- ⚠️ DELETE .platform-keys.* files after storing securely!
- ⚠️ Rotate keys every 90 days
- ⚠️ Never commit keys to git

## Cleanup

```bash
# After storing keys securely, delete local copies
rm -f .platform-keys.json .platform-keys.env

# Verify they're gone
ls -la .platform-keys.*
# Should show: No such file or directory
```

## Monitoring

```sql
-- Check platform key usage
SELECT
  u.email,
  k.name,
  k.usage_count,
  k.last_used_at,
  k.created_at
FROM api_keys k
JOIN users u ON u.id = k.user_id
WHERE u.email LIKE '%@afterdarksys.com'
  AND 'service_account' = ANY(u.roles)
ORDER BY k.last_used_at DESC;
```

## Troubleshooting

### "User already exists"
- Keys for existing users won't be regenerated
- Revoke old keys first, then re-run script

### "Permission denied"
- Ensure DB_USER has permission to create users and API keys
- Check database grants

### Keys not working
- Verify key is active: `SELECT * FROM api_keys WHERE key_prefix = 'chg_...'`
- Check scopes match required permissions
- Review audit log: `SELECT * FROM api_key_audit WHERE api_key_id = '...'`
