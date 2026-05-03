#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== Building Anthill Runtime Node ==="

cd "$PROJECT_ROOT/runtime"

rm -rf build
mkdir -p build

echo "Building for linux/amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/anthill-runtime-linux-amd64 ./cmd/node

echo "Building for linux/arm64..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/anthill-runtime-linux-arm64 ./cmd/node

echo "Building for windows/amd64..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o build/anthill-runtime-windows-amd64.exe ./cmd/node

echo "Building for darwin/amd64..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o build/anthill-runtime-darwin-amd64 ./cmd/node

echo "Building for darwin/arm64..."
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o build/anthill-runtime-darwin-arm64 ./cmd/node

echo "Building CLI..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/anthill-cli-linux-amd64 ./cmd/cli
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/anthill-cli-linux-arm64 ./cmd/cli
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o build/anthill-cli-windows-amd64.exe ./cmd/cli
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o build/anthill-cli-darwin-arm64 ./cmd/cli

cd build

echo "Build complete: build/anthill-runtime.tar.gz"
ls -lh .
