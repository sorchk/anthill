#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== Building Anthill ==="

echo "=== Building Backend ==="
cd "$PROJECT_ROOT/admin"
go build -o anthill-admin ./cmd/server
echo "Backend built successfully"

echo ""
echo "=== Building Frontend ==="
cd "$PROJECT_ROOT/web"
npm run build
echo "Frontend built successfully"

rm -f anthill-admin

echo ""
echo "=== All builds passed ==="
