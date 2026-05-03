#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== Runtime Node E2E Test ==="

cd "$PROJECT_ROOT/runtime"

echo "1. Building runtime..."
go build -o anthill-runtime ./cmd/node
echo "OK: Runtime binary built"

echo "2. Building CLI..."
go build -o anthill-cli ./cmd/cli
echo "OK: CLI binary built"

echo "3. Testing version..."
./anthill-runtime -version
echo "OK: Version works"

echo "4. Testing CLI help..."
./anthill-cli -h | head -20
echo "OK: CLI help works"

rm -f anthill-runtime anthill-cli

echo ""
echo "=== All Runtime tests passed ==="
