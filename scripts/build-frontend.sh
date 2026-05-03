#!/bin/bash
set -e

echo "=== Building Anthill Frontend ==="

cd /home/sorc/test/test2/web

# Install dependencies
npm install

# Build
npm run build

# Copy to dist folder
cd /home/sorc/test/test2
rm -rf dist/frontend
mkdir -p dist/frontend
cp -r web/dist/* dist/frontend/

echo "Build complete: dist/frontend/"
ls -lh dist/frontend/