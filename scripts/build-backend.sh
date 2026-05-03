#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== Building Anthill Backend ==="

cd "$PROJECT_ROOT/admin"

rm -rf build
mkdir -p build

echo "Building for linux/amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/anthill-admin-linux-amd64 ./cmd/server

echo "Building for linux/arm64..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/anthill-admin-linux-arm64 ./cmd/server

echo "Building for windows/amd64..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o build/anthill-admin-windows-amd64.exe ./cmd/server

echo "Building for darwin/amd64..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o build/anthill-admin-darwin-amd64 ./cmd/server

echo "Building for darwin/arm64..."
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o build/anthill-admin-darwin-arm64 ./cmd/server

cd build
tar -czf anthill-admin.tar.gz anthill-admin-*-*
rm -f anthill-admin-linux-amd64 anthill-admin-linux-arm64 anthill-admin-darwin-amd64 anthill-admin-darwin-arm64 anthill-admin-windows-amd64.exe

echo "Build complete: build/anthill-admin.tar.gz"
ls -lh .
