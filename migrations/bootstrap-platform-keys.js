#!/usr/bin/env node
/**
 * Bootstrap Platform API Keys
 * Creates service account users and generates API keys for backend services
 */

const bcrypt = require('bcryptjs');
const crypto = require('crypto');
const { Pool } = require('pg');

const DB_CONFIG = {
  host: process.env.DB_HOST || 'localhost',
  port: process.env.DB_PORT || 5432,
  database: process.env.DB_NAME || 'adsops_changes',
  user: process.env.DB_USER || 'adsops_app',
  password: process.env.DB_PASSWORD || process.env.PGPASSWORD,
  ssl: process.env.DB_SSL !== 'false' ? { rejectUnauthorized: false } : false,
};

// Platform service accounts to create
const PLATFORM_ACCOUNTS = [
  {
    email: 'adsops-service@afterdarksys.com',
    name: 'AdsOps CLI Service Account',
    description: 'API key for adsops CLI tool',
    keyName: 'adsops-cli-default',
    scopes: ['tickets:read', 'tickets:write', 'approvals:read', 'approvals:write'],
  },
  {
    email: 'ads-ai-staff@afterdarksys.com',
    name: 'AfterDark AI Staff Service Account',
    description: 'API key for ads-ai-staff automation',
    keyName: 'ads-ai-staff-automation',
    scopes: ['tickets:read', 'tickets:write', 'approvals:read'],
  },
  {
    email: 'changes-notifier@afterdarksys.com',
    name: 'Changes Notifier Service Account',
    description: 'API key for email notification daemon',
    keyName: 'changes-notifier-daemon',
    scopes: ['tickets:read', 'approvals:read'],
  },
  {
    email: 'infrastructure-monitor@afterdarksys.com',
    name: 'Infrastructure Monitor Service Account',
    description: 'API key for infrastructure monitoring integration',
    keyName: 'infrastructure-monitor',
    scopes: ['tickets:read', 'tickets:write'],
  },
  {
    email: 'deployment-pipeline@afterdarksys.com',
    name: 'Deployment Pipeline Service Account',
    description: 'API key for CI/CD deployment automation',
    keyName: 'deployment-ci-cd',
    scopes: ['tickets:read', 'tickets:write', 'approvals:read'],
  },
];

/**
 * Generate API key with format: chg_<base64url>
 */
function generateAPIKey() {
  const randomBytes = crypto.randomBytes(32);
  const encoded = Buffer.from(randomBytes).toString('base64url');
  const apiKey = `chg_${encoded}`;

  const keyPrefix = apiKey.substring(0, 16);

  return { apiKey, keyPrefix };
}

/**
 * Get or create default organization
 */
async function getOrCreateOrganization(pool) {
  let result = await pool.query(`
    SELECT id FROM organizations WHERE slug = 'afterdark-internal' LIMIT 1
  `);

  if (result.rows.length > 0) {
    return result.rows[0].id;
  }

  // Create default organization
  result = await pool.query(`
    INSERT INTO organizations (
      name,
      slug,
      industry,
      compliance_frameworks,
      admin_email,
      require_mfa
    ) VALUES (
      'AfterDark Internal Services',
      'afterdark-internal',
      'it',
      ARRAY['sox', 'gdpr']::compliance_framework[],
      'admin@afterdarksys.com',
      false
    )
    RETURNING id
  `);

  console.log('✨ Created default organization: afterdark-internal');
  return result.rows[0].id;
}

/**
 * Create service account user
 */
async function createServiceAccount(pool, orgID, account) {
  // Check if user exists
  let result = await pool.query(
    'SELECT id FROM users WHERE email = $1',
    [account.email]
  );

  if (result.rows.length > 0) {
    console.log(`   ⚠️  User already exists: ${account.email}`);
    return result.rows[0].id;
  }

  // Create user
  result = await pool.query(`
    INSERT INTO users (
      organization_id,
      email,
      full_name,
      password_hash,
      roles,
      is_active,
      email_verified,
      created_at
    ) VALUES (
      $1, $2, $3, $4,
      ARRAY['service_account']::text[],
      true,
      true,
      NOW()
    )
    RETURNING id
  `, [
    orgID,
    account.email,
    account.name,
    '', // No password for service accounts
  ]);

  console.log(`   ✅ Created user: ${account.email}`);
  return result.rows[0].id;
}

/**
 * Create API key for service account
 */
async function createAPIKey(pool, userID, orgID, account) {
  const { apiKey, keyPrefix } = generateAPIKey();
  const keyHash = await bcrypt.hash(apiKey, 10);

  // Check if key already exists for this user with this name
  const existing = await pool.query(
    `SELECT id FROM api_keys
     WHERE user_id = $1 AND name = $2 AND revoked_at IS NULL`,
    [userID, account.keyName]
  );

  if (existing.rows.length > 0) {
    console.log(`   ⚠️  API key already exists: ${account.keyName}`);
    console.log(`      (Revoke old key first to regenerate)`);
    return null;
  }

  // Create API key
  const result = await pool.query(`
    INSERT INTO api_keys (
      user_id,
      organization_id,
      name,
      key_hash,
      key_prefix,
      scopes,
      rate_limit_rpm,
      is_active,
      created_at
    ) VALUES (
      $1, $2, $3, $4, $5, $6, 300, true, NOW()
    )
    RETURNING id, created_at
  `, [
    userID,
    orgID,
    account.keyName,
    keyHash,
    keyPrefix,
    account.scopes,
  ]);

  console.log(`   ✅ Created API key: ${account.keyName}`);

  return {
    id: result.rows[0].id,
    key: apiKey,
    prefix: keyPrefix,
    created: result.rows[0].created_at,
  };
}

/**
 * Save keys to secure file
 */
function saveKeysToFile(keys) {
  const fs = require('fs');
  const path = require('path');
  const os = require('os');

  const outputFile = path.join(__dirname, '../.platform-keys.json');
  const envFile = path.join(__dirname, '../.platform-keys.env');
  const homeKeyFile = path.join(os.homedir(), 'changes_api_afterdark_key.txt');

  // JSON format
  fs.writeFileSync(outputFile, JSON.stringify(keys, null, 2), { mode: 0o600 });

  // ENV format
  const envContent = keys.map(k =>
    `${k.service.toUpperCase().replace(/-/g, '_')}_API_KEY="${k.apiKey}"`
  ).join('\n');

  fs.writeFileSync(envFile, envContent + '\n', { mode: 0o600 });

  // Human-readable format for home directory
  const readableContent = [
    '═'.repeat(80),
    'AFTER DARK SYSTEMS - CHANGES API PLATFORM KEYS',
    '═'.repeat(80),
    '',
    '⚠️  KEEP THESE KEYS SECURE - DO NOT COMMIT TO GIT',
    '',
    ...keys.map(k => [
      '─'.repeat(80),
      `Service: ${k.service}`,
      `Email:   ${k.email}`,
      `Description: ${k.description}`,
      `API Key: ${k.apiKey}`,
      `Scopes:  ${k.scopes.join(', ')}`,
      `Created: ${k.created}`,
      '',
      'Usage:',
      `  curl -H "X-API-Key: ${k.apiKey}" \\`,
      `    https://api.changes.afterdarksys.com/v1/tickets`,
      '',
    ]).flat(),
    '─'.repeat(80),
    '',
    'SECURITY NOTES:',
    '• Store these keys in OCI Vault or secure key management',
    '• Never commit these keys to git',
    '• Rotate keys every 90 days',
    '• Delete this file after storing keys securely',
    '',
    'Generated: ' + new Date().toISOString(),
    '═'.repeat(80),
  ].join('\n');

  fs.writeFileSync(homeKeyFile, readableContent, { mode: 0o600 });

  console.log(`\n💾 Keys saved to:`);
  console.log(`   JSON: ${outputFile}`);
  console.log(`   ENV:  ${envFile}`);
  console.log(`   TXT:  ${homeKeyFile}`);
  console.log(`   (chmod 600 - only you can read)`);
}

/**
 * Main execution
 */
async function main() {
  console.log('╔══════════════════════════════════════════════════════════════╗');
  console.log('║        Bootstrap Platform API Keys for Changes API          ║');
  console.log('╚══════════════════════════════════════════════════════════════╝\n');

  const pool = new Pool(DB_CONFIG);
  const generatedKeys = [];

  try {
    // Get or create organization
    console.log('🏢 Setting up organization...');
    const orgID = await getOrCreateOrganization(pool);
    console.log(`   Organization ID: ${orgID}\n`);

    // Create service accounts and API keys
    console.log('👥 Creating service accounts and API keys...\n');

    for (const account of PLATFORM_ACCOUNTS) {
      console.log(`📦 ${account.name}`);

      const userID = await createServiceAccount(pool, orgID, account);
      const keyData = await createAPIKey(pool, userID, orgID, account);

      if (keyData) {
        generatedKeys.push({
          service: account.keyName,
          email: account.email,
          description: account.description,
          apiKey: keyData.key,
          keyPrefix: keyData.prefix,
          scopes: account.scopes,
          created: keyData.created,
        });
      }

      console.log('');
    }

    // Display results
    if (generatedKeys.length > 0) {
      console.log('╔══════════════════════════════════════════════════════════════╗');
      console.log('║                  🔑 GENERATED API KEYS 🔑                    ║');
      console.log('╚══════════════════════════════════════════════════════════════╝\n');

      generatedKeys.forEach(k => {
        console.log(`Service: ${k.service}`);
        console.log(`Email:   ${k.email}`);
        console.log(`Key:     ${k.apiKey}`);
        console.log(`Scopes:  ${k.scopes.join(', ')}`);
        console.log('─'.repeat(70));
      });

      console.log('\n⚠️  IMPORTANT: Save these keys securely!');
      console.log('   These keys will NOT be shown again.\n');

      // Save to file
      saveKeysToFile(generatedKeys);

      // Show usage examples
      console.log('\n📝 Usage Examples:\n');
      console.log('# adsops CLI');
      console.log('export CHANGES_API_KEY="' + generatedKeys.find(k => k.service === 'adsops-cli-default')?.apiKey + '"');
      console.log('curl -H "X-API-Key: $CHANGES_API_KEY" https://api.changes.afterdarksys.com/v1/tickets\n');

      console.log('# ads-ai-staff');
      console.log('export AI_STAFF_API_KEY="' + generatedKeys.find(k => k.service === 'ads-ai-staff-automation')?.apiKey + '"');
      console.log('# Add to ads-ai-staff .env file\n');

      console.log('📌 Next Steps:');
      console.log('   1. Store keys in OCI Vault or secure key management');
      console.log('   2. Add keys to respective service .env files');
      console.log('   3. Update service configurations to use Changes API');
      console.log('   4. Test connectivity with each service');
      console.log('   5. DELETE .platform-keys.json and .platform-keys.env after storing');
    } else {
      console.log('ℹ️  No new keys generated (all already exist)');
      console.log('   To regenerate, first revoke existing keys in the database');
    }

  } catch (error) {
    console.error('\n❌ Error:', error.message);
    console.error(error.stack);
    process.exit(1);
  } finally {
    await pool.end();
  }

  console.log('\n✨ Bootstrap complete!\n');
}

main();
