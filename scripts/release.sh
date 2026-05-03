#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== Building Anthill Release ==="

cd "$PROJECT_ROOT"

rm -rf dist
mkdir -p dist

echo "Building admin backend..."
bash scripts/build-admin.sh

echo "Building runtime..."
bash scripts/build-runtime.sh

echo "Building frontend..."
bash scripts/build-frontend.sh

cp admin/build/*.tar.gz dist/
cp runtime/build/*.tar.gz dist/
cp -r web/dist dist/web

cat > dist/RELEASE_INFO.txt << EOF
Anthill Platform v1.0.0
=======================

Components:
- anthill-admin.tar.gz  - Admin Panel Backend (Go+Gin)
- anthill-runtime.tar.gz - Runtime Node Agent
- web/                  - Admin Panel Web UI

Quick Start:
1. Extract backend: tar -xzf anthill-admin.tar.gz
2. Run: ./anthill-admin-linux-amd64
3. Access: http://localhost:8080

Runtime Node:
1. Extract: tar -xzf anthill-runtime.tar.gz
2. Run: ./anthill-runtime-linux-amd64 -admin wss://localhost:8080/ws

Default credentials: admin / admin123

EOF

echo ""
echo "=== Release Build Complete ==="
echo "Output: dist/"
ls -lh dist/
