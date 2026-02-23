#!/bin/bash
# Quick fix script to grant schema permissions to existing database

set -e

# Load .env file to get database details
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Extract database details from DATABASE_URL
DB_USER="${DB_USER:-rubxy_user}"
DB_NAME="${DB_NAME:-rubxy}"

echo "====================================="
echo "  Fixing PostgreSQL Permissions"
echo "====================================="
echo ""
echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo ""

# Grant permissions on the public schema
echo "Granting schema permissions..."
sudo -u postgres psql -d $DB_NAME -c "GRANT ALL ON SCHEMA public TO $DB_USER;"
sudo -u postgres psql -d $DB_NAME -c "GRANT CREATE ON SCHEMA public TO $DB_USER;"
sudo -u postgres psql -d $DB_NAME -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $DB_USER;"
sudo -u postgres psql -d $DB_NAME -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $DB_USER;"

# If tables already exist, grant permissions on them
echo "Granting permissions on existing tables (if any)..."
sudo -u postgres psql -d $DB_NAME -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $DB_USER;" || true
sudo -u postgres psql -d $DB_NAME -c "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $DB_USER;" || true

echo ""
echo "✓ Permissions updated successfully!"
echo ""
echo "You can now run: ./rubxy"
