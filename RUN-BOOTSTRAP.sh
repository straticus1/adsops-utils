#!/bin/bash
# Quick script to bootstrap platform API keys
# This creates service accounts and generates API keys for backend services

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║      Bootstrap Platform API Keys for Changes API            ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Check if running in correct directory
if [ ! -f "migrations/bootstrap-platform-keys.js" ]; then
    echo "❌ Error: Run this script from adsops-utils root directory"
    echo "   cd ~/development/adsops-utils"
    echo "   ./RUN-BOOTSTRAP.sh"
    exit 1
fi

# Check for required environment variables
if [ -z "$DB_HOST" ] || [ -z "$PGPASSWORD" ]; then
    echo "⚠️  Database credentials not set in environment"
    echo ""
    echo "Please set:"
    echo "  export DB_HOST=your-db-host"
    echo "  export DB_USER=adsops_app  # (or your db user)"
    echo "  export PGPASSWORD=your-password"
    echo "  export DB_NAME=adsops_changes"
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Set defaults if not provided
export DB_USER="${DB_USER:-adsops_app}"
export DB_NAME="${DB_NAME:-adsops_changes}"
export DB_PORT="${DB_PORT:-5432}"

echo "📊 Configuration:"
echo "   DB_HOST: ${DB_HOST}"
echo "   DB_USER: ${DB_USER}"
echo "   DB_NAME: ${DB_NAME}"
echo "   DB_PORT: ${DB_PORT}"
echo ""

read -p "Proceed with bootstrap? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo "🚀 Running bootstrap..."
echo ""

# Run the bootstrap script
cd migrations
node bootstrap-platform-keys.js

if [ $? -eq 0 ]; then
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                   ✨ BOOTSTRAP COMPLETE ✨                   ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    echo "📝 Keys saved to:"
    echo "   • ~/changes_api_afterdark_key.txt (human-readable)"
    echo "   • .platform-keys.json (JSON format)"
    echo "   • .platform-keys.env (environment variables)"
    echo ""
    echo "⚠️  IMPORTANT:"
    echo "   1. Review ~/changes_api_afterdark_key.txt"
    echo "   2. Store keys in OCI Vault or secure key management"
    echo "   3. Delete temporary files after storing:"
    echo "      rm -f .platform-keys.* ~/changes_api_afterdark_key.txt"
    echo ""
    echo "📌 Next steps:"
    echo "   • Add keys to service .env files"
    echo "   • Test with: curl -H 'X-API-Key: chg_...' \\"
    echo "              https://api.changes.afterdarksys.com/v1/auth/me"
    echo ""
else
    echo ""
    echo "❌ Bootstrap failed. Check error messages above."
    exit 1
fi
