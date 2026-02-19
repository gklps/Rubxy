#!/bin/bash
set -e

echo "==================================="
echo "  Building Rubxy"
echo "==================================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go first."
    exit 1
fi

echo "Go version: $(go version)"
echo ""

# Clean previous builds
echo "[1/3] Cleaning previous builds..."
rm -f rubxy
echo "Done."

# Download dependencies
echo "[2/3] Downloading dependencies..."
go mod download
go mod tidy
echo "Done."

# Build the application
echo "[3/3] Building application..."
CGO_ENABLED=0 go build -ldflags="-s -w" -o rubxy main.go
echo "Done."

echo ""
echo "==================================="
echo "  Build Complete!"
echo "==================================="
echo ""
echo "Executable created: ./rubxy"
echo "Size: $(du -h rubxy | cut -f1)"
echo ""
echo "To run: ./rubxy"
