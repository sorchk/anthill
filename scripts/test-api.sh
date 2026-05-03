#!/bin/bash
set -e

BASE_URL="${BASE_URL:-http://localhost:8080/api}"
TOKEN_FILE="/tmp/anthill-token.txt"

echo "=== Anthill API Test ==="

echo "1. Testing login..."
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo $LOGIN_RESP | jq -r '.token')
if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
  echo "FAIL: Login failed"
  echo $LOGIN_RESP | jq .
  exit 1
fi
echo $TOKEN > $TOKEN_FILE
echo "OK: Logged in"

echo "2. Testing /me..."
ME=$(curl -s "$BASE_URL/me" -H "Authorization: Bearer $TOKEN")
echo $ME | jq .

echo "3. Testing /stats..."
curl -s "$BASE_URL/stats" -H "Authorization: Bearer $TOKEN" | jq .

echo "4. Testing /nodes..."
curl -s "$BASE_URL/nodes" -H "Authorization: Bearer $TOKEN" | jq .

echo ""
echo "=== All API tests passed ==="