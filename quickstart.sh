#!/bin/bash
set -e

echo "==================================="
echo "  Rubxy Quick Start"
echo "==================================="
echo ""
echo "This script will:"
echo "  1. Set up PostgreSQL database"
echo "  2. Create configuration (.env)"
echo "  3. Build the application"
echo "  4. Start the server"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# Make scripts executable if not already
chmod +x setup.sh build.sh

# Run setup
echo ""
./setup.sh

# Run build
echo ""
./build.sh

# Run the application
echo ""
echo "==================================="
echo "  Starting Rubxy..."
echo "==================================="
echo ""
./rubxy
