#!/bin/bash
set -e

# Ynet Space Sync Import Script
# Usage: ./sync-to-prod.sh <export.zip> [--auto-confirm]

PROD_API="${PROD_API:-http://localhost:8080}"
SPACE_ID="${SPACE_ID:?'SPACE_ID is required. Set via: export SPACE_ID=200'}"
ZIP_FILE="${1:?'Usage: sync-to-prod.sh <export.zip> [--auto-confirm]'}"
AUTO_CONFIRM="${2:-}"

if [ ! -f "$ZIP_FILE" ]; then
    echo "Error: File not found: $ZIP_FILE"
    exit 1
fi

echo "=== Ynet Space Sync Import ==="
echo "API:   $PROD_API"
echo "Space: $SPACE_ID"
echo "File:  $ZIP_FILE ($(du -h "$ZIP_FILE" | cut -f1))"
echo ""

# Step 1: Preview
echo "--- Step 1: Preview ---"
PREVIEW=$(curl -sf -X POST "$PROD_API/api/space/$SPACE_ID/sync/import/preview" \
  -F "file=@$ZIP_FILE")

if [ $? -ne 0 ]; then
    echo "Error: Preview request failed"
    exit 1
fi

TOKEN=$(echo "$PREVIEW" | jq -r '.import_token')
if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "Error: Failed to get import token"
    echo "$PREVIEW" | jq .
    exit 1
fi

echo ""
echo "Import Plan:"
echo "$PREVIEW" | jq '.plan'

WARNINGS=$(echo "$PREVIEW" | jq -r '.warnings[]? // empty')
if [ -n "$WARNINGS" ]; then
    echo ""
    echo "Warnings:"
    echo "$WARNINGS"
fi

# Step 2: Confirm
if [ "$AUTO_CONFIRM" != "--auto-confirm" ]; then
    echo ""
    read -p "Proceed with import? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 0
    fi
fi

echo ""
echo "--- Step 2: Importing ---"
RESULT=$(curl -sf -X POST "$PROD_API/api/space/$SPACE_ID/sync/import/confirm" \
  -H "Content-Type: application/json" \
  -d "{\"import_token\": \"$TOKEN\"}")

if [ $? -ne 0 ]; then
    echo "Error: Import failed"
    exit 1
fi

echo ""
echo "Import Result:"
echo "$RESULT" | jq '.statistics'

HISTORY_ID=$(echo "$RESULT" | jq -r '.sync_history_id // empty')
if [ -n "$HISTORY_ID" ]; then
    echo ""
    echo "Sync History ID: $HISTORY_ID"
fi

echo ""
echo "=== Sync Complete ==="
