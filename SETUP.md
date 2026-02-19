# Rubxy Setup Guide

This guide will help you set up and run Rubxy on a new server or VM.

## Prerequisites

- Linux (Ubuntu/Debian) or macOS
- Go 1.19 or higher
- PostgreSQL 12 or higher (will be installed by setup script if missing)

## Quick Start (Automated Setup)

For a fresh installation on a new VM, use the automated setup script:

```bash
# 1. Clone the repository
git clone <repository-url>
cd Rubxy

# 2. Make scripts executable
chmod +x setup.sh build.sh

# 3. Run the setup script (installs PostgreSQL, creates DB, generates .env)
./setup.sh

# 4. Build the application
./build.sh

# 5. Run the application
./rubxy
```

The server will start on port 8080 by default.

## Manual Setup

If you prefer to set things up manually:

### 1. Install PostgreSQL

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

**macOS:**
```bash
brew install postgresql
brew services start postgresql
```

### 2. Create Database and User

```bash
# Access PostgreSQL
sudo -u postgres psql  # Linux
# or
psql postgres          # macOS

# Create user and database
CREATE USER rubxy_user WITH PASSWORD 'your_secure_password';
CREATE DATABASE rubxy OWNER rubxy_user;
GRANT ALL PRIVILEGES ON DATABASE rubxy TO rubxy_user;
\q
```

### 3. Configure Environment Variables

Copy the example environment file:
```bash
cp .env.example .env
```

Edit `.env` with your settings:
```bash
PORT=:8080
ACCESS_SECRET=<generate-random-secret>
REFRESH_SECRET=<generate-random-secret>
DATABASE_URL=postgres://rubxy_user:your_secure_password@localhost:5432/rubxy?sslmode=disable
```

**Generate secure secrets:**
```bash
openssl rand -hex 32  # Run twice for ACCESS_SECRET and REFRESH_SECRET
```

### 4. Build and Run

```bash
# Install dependencies
go mod download

# Build
go build -o rubxy main.go

# Run
./rubxy
```

## Configuration

All configuration is done via environment variables (loaded from `.env` file):

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `:8080` |
| `ACCESS_SECRET` | JWT access token secret | `your-access-secret` (change this!) |
| `REFRESH_SECRET` | JWT refresh token secret | `your-refresh-secret` (change this!) |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:password@localhost:5432/rubxy?sslmode=disable` |

## Running in Production

### Using systemd (Linux)

Create `/etc/systemd/system/rubxy.service`:

```ini
[Unit]
Description=Rubxy Proxy Server
After=network.target postgresql.service

[Service]
Type=simple
User=rubxy
WorkingDirectory=/opt/rubxy
ExecStart=/opt/rubxy/rubxy
Restart=on-failure
RestartSec=5s

# Environment
Environment="PATH=/usr/local/bin:/usr/bin:/bin"
EnvironmentFile=/opt/rubxy/.env

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable rubxy
sudo systemctl start rubxy
sudo systemctl status rubxy
```

### Using with Caddy

Example Caddyfile configuration:

```
your-domain.com {
    reverse_proxy localhost:8080

    # Optional: enable compression
    encode gzip

    # Optional: logging
    log {
        output file /var/log/caddy/rubxy.log
    }
}
```

## Troubleshooting

### Database Connection Issues

**Error: `pq: password authentication failed for user`**

- Check your DATABASE_URL in `.env`
- Verify PostgreSQL user exists: `sudo -u postgres psql -c "\du"`
- Test connection: `psql -U rubxy_user -d rubxy -h localhost`

**Error: `database "rubxy" does not exist`**

- Create it: `sudo -u postgres createdb rubxy -O rubxy_user`

### Port Already in Use

If port 8080 is already in use, change `PORT` in `.env`:
```bash
PORT=:8081
```

### Build Errors

Make sure Go modules are up to date:
```bash
go mod tidy
go mod download
```

## Development

To run in development mode:
```bash
go run main.go
```

To run tests:
```bash
go test ./...
```

## Scripts

- `setup.sh` - Automated setup (PostgreSQL + database + .env generation)
- `build.sh` - Build the application binary
- `.env.example` - Example environment configuration

## Security Notes

- **Never commit `.env` to git** - it contains secrets
- Always use strong, random secrets for `ACCESS_SECRET` and `REFRESH_SECRET`
- Use SSL/TLS in production (configure via reverse proxy like Caddy)
- Keep PostgreSQL updated with security patches
- Consider using environment-specific configurations

## Support

For issues or questions, check the main README or open an issue in the repository.
