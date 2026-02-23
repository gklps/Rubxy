#!/bin/bash
set -e

echo "==================================="
echo "  Rubxy Setup Script"
echo "==================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS
OS="unknown"
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
fi

echo "Detected OS: $OS"
echo ""

# Default configuration
DB_USER="${DB_USER:-rubxy_user}"
DB_PASSWORD="${DB_PASSWORD:-rubxy_pass_$(openssl rand -hex 8)}"
DB_NAME="${DB_NAME:-rubxy}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"

echo -e "${YELLOW}Configuration:${NC}"
echo "  DB_USER: $DB_USER"
echo "  DB_NAME: $DB_NAME"
echo "  DB_HOST: $DB_HOST"
echo "  DB_PORT: $DB_PORT"
echo ""

# Check if PostgreSQL is installed
echo -e "${YELLOW}[1/5] Checking PostgreSQL installation...${NC}"
if ! command -v psql &> /dev/null; then
    echo -e "${RED}PostgreSQL is not installed.${NC}"
    echo "Installing PostgreSQL..."

    if [[ "$OS" == "linux" ]]; then
        sudo apt update
        sudo apt install -y postgresql postgresql-contrib openssl
        sudo systemctl start postgresql
        sudo systemctl enable postgresql
    elif [[ "$OS" == "macos" ]]; then
        if command -v brew &> /dev/null; then
            brew install postgresql openssl
            brew services start postgresql
        else
            echo -e "${RED}Please install Homebrew first or manually install PostgreSQL${NC}"
            exit 1
        fi
    else
        echo -e "${RED}Unsupported OS. Please install PostgreSQL manually.${NC}"
        exit 1
    fi
    echo -e "${GREEN}PostgreSQL installed successfully!${NC}"
else
    echo -e "${GREEN}PostgreSQL is already installed.${NC}"
fi

# Ensure PostgreSQL is running
echo -e "${YELLOW}[2/5] Ensuring PostgreSQL is running...${NC}"
if [[ "$OS" == "linux" ]]; then
    # Check if the main cluster is running
    PG_VERSION=$(ls /etc/postgresql/ 2>/dev/null | head -n1)
    if [ -n "$PG_VERSION" ]; then
        echo "Found PostgreSQL version: $PG_VERSION"
        sudo systemctl start postgresql@${PG_VERSION}-main || sudo systemctl start postgresql
        sleep 2

        # Verify it's actually running
        if sudo systemctl is-active --quiet postgresql@${PG_VERSION}-main || sudo systemctl is-active --quiet postgresql; then
            echo -e "${GREEN}PostgreSQL service is active.${NC}"
        else
            echo -e "${YELLOW}PostgreSQL service status:${NC}"
            sudo systemctl status postgresql --no-pager -l
        fi
    else
        sudo systemctl start postgresql
        sleep 2
    fi

    # Double check with pg_isready
    if command -v pg_isready &> /dev/null; then
        if sudo -u postgres pg_isready; then
            echo -e "${GREEN}PostgreSQL is accepting connections.${NC}"
        else
            echo -e "${RED}PostgreSQL is not ready to accept connections yet.${NC}"
            echo "Trying to restart..."
            sudo systemctl restart postgresql
            sleep 3
            sudo -u postgres pg_isready
        fi
    fi
elif [[ "$OS" == "macos" ]]; then
    brew services start postgresql || true
    sleep 2
fi
echo -e "${GREEN}PostgreSQL is running.${NC}"

# Create database user and database
echo -e "${YELLOW}[3/5] Setting up database...${NC}"
if [[ "$OS" == "linux" ]]; then
    # Linux: use sudo -u postgres
    sudo -u postgres psql -tc "SELECT 1 FROM pg_user WHERE usename = '$DB_USER'" | grep -q 1 || \
        sudo -u postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"

    sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1 || \
        sudo -u postgres psql -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"

    sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
elif [[ "$OS" == "macos" ]]; then
    # macOS: direct psql access
    psql postgres -tc "SELECT 1 FROM pg_user WHERE usename = '$DB_USER'" | grep -q 1 || \
        psql postgres -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"

    psql postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1 || \
        psql postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"

    psql postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
fi
echo -e "${GREEN}Database setup complete!${NC}"

# Create .env file
echo -e "${YELLOW}[4/5] Creating .env file...${NC}"
cat > .env <<EOF
# Rubxy Configuration
PORT=:8080
ACCESS_SECRET=$(openssl rand -hex 32)
REFRESH_SECRET=$(openssl rand -hex 32)
DATABASE_URL=postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable
EOF
echo -e "${GREEN}.env file created!${NC}"

# Test database connection
echo -e "${YELLOW}[5/5] Testing database connection...${NC}"
export PGPASSWORD="$DB_PASSWORD"
if psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT version();" > /dev/null 2>&1; then
    echo -e "${GREEN}Database connection successful!${NC}"
else
    echo -e "${RED}Database connection failed!${NC}"
    echo ""
    echo "Debugging information:"
    echo "  Database URL: postgres://$DB_USER:****@$DB_HOST:$DB_PORT/$DB_NAME"

    # Try to connect as postgres user to verify PostgreSQL is working
    if sudo -u postgres psql -c "SELECT version();" > /dev/null 2>&1; then
        echo -e "${GREEN}  ✓ PostgreSQL is running (verified as postgres user)${NC}"

        # Check if user exists
        if sudo -u postgres psql -tc "SELECT 1 FROM pg_user WHERE usename = '$DB_USER'" | grep -q 1; then
            echo -e "${GREEN}  ✓ User '$DB_USER' exists${NC}"
        else
            echo -e "${RED}  ✗ User '$DB_USER' does not exist${NC}"
        fi

        # Check if database exists
        if sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1; then
            echo -e "${GREEN}  ✓ Database '$DB_NAME' exists${NC}"
        else
            echo -e "${RED}  ✗ Database '$DB_NAME' does not exist${NC}"
        fi

        # Check pg_hba.conf for authentication settings
        echo ""
        echo "PostgreSQL authentication settings (pg_hba.conf):"
        sudo grep -v "^#" /etc/postgresql/*/main/pg_hba.conf 2>/dev/null | grep -v "^$" | tail -5

        echo ""
        echo -e "${YELLOW}The database setup appears complete, but connection as '$DB_USER' failed.${NC}"
        echo "This might be a PostgreSQL authentication configuration issue."
        echo ""
        echo "Try running manually:"
        echo "  PGPASSWORD='$DB_PASSWORD' psql -h $DB_HOST -U $DB_USER -d $DB_NAME"

    else
        echo -e "${RED}  ✗ PostgreSQL is not responding${NC}"
        echo ""
        echo "Try these commands to diagnose:"
        echo "  sudo systemctl status postgresql"
        echo "  sudo -u postgres pg_isready"
    fi

    unset PGPASSWORD
    exit 1
fi
unset PGPASSWORD

echo ""
echo -e "${GREEN}==================================="
echo "  Setup Complete!"
echo "===================================${NC}"
echo ""
echo "Next steps:"
echo "  1. Review the .env file and update if needed"
echo "  2. Run './build.sh' to build the application"
echo "  3. Run './rubxy' to start the server"
echo ""
echo -e "${YELLOW}Note: Keep your .env file secure and don't commit it to git!${NC}"
