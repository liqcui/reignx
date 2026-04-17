#!/bin/bash

# ReignX Database Setup Script
# This script initializes the PostgreSQL database for ReignX

set -e

echo "=== ReignX Database Setup ==="

# Configuration
DB_NAME="reignx"
DB_USER="reignx"
DB_PASSWORD="reignx"
DB_HOST="localhost"
DB_PORT="5432"

# Check if PostgreSQL is running
if ! pg_isready -h $DB_HOST -p $DB_PORT > /dev/null 2>&1; then
    echo "ERROR: PostgreSQL is not running on $DB_HOST:$DB_PORT"
    echo "Please start PostgreSQL first:"
    echo "  brew services start postgresql@16"
    exit 1
fi

echo "✓ PostgreSQL is running"

# Check if database exists
if psql -h $DB_HOST -p $DB_PORT -U $(whoami) -lqt | cut -d \| -f 1 | grep -qw $DB_NAME; then
    echo "✓ Database '$DB_NAME' already exists"
else
    echo "Creating database '$DB_NAME'..."
    createdb -h $DB_HOST -p $DB_PORT -U $(whoami) $DB_NAME
    echo "✓ Database created"
fi

# Check if user exists
if psql -h $DB_HOST -p $DB_PORT -U $(whoami) -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'" | grep -q 1; then
    echo "✓ User '$DB_USER' already exists"
else
    echo "Creating user '$DB_USER'..."
    psql -h $DB_HOST -p $DB_PORT -U $(whoami) -d postgres -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"
    echo "✓ User created"
fi

# Grant privileges
echo "Granting privileges..."
psql -h $DB_HOST -p $DB_PORT -U $(whoami) -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
psql -h $DB_HOST -p $DB_PORT -U $(whoami) -d $DB_NAME -c "GRANT ALL ON SCHEMA public TO $DB_USER;"
echo "✓ Privileges granted"

# Run migrations
echo "Running migrations..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="$SCRIPT_DIR/../migrations"

if [ -f "$MIGRATIONS_DIR/000001_initial_schema.up.sql" ]; then
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$MIGRATIONS_DIR/000001_initial_schema.up.sql"
    echo "✓ Migrations applied"
else
    echo "WARNING: Migration file not found at $MIGRATIONS_DIR/000001_initial_schema.up.sql"
fi

echo ""
echo "=== Database Setup Complete ==="
echo ""
echo "Database connection details:"
echo "  Host:     $DB_HOST"
echo "  Port:     $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User:     $DB_USER"
echo "  Password: $DB_PASSWORD"
echo ""
echo "Connection string:"
echo "  postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable"
echo ""
echo "You can now start reignxd:"
echo "  ./bin/reignxd"
