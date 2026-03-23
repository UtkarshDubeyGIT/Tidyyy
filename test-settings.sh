#!/bin/bash

# Debug script for testing Tidyyy file detection
# Usage: ./test-settings.sh

set -e

TIDYYY_DIR="/Users/dubeysmac/Developer/Go/tidy-app/Tidyyy"
TEST_DIR="/tmp/tidyyy-test"

echo "=== Tidyyy Debug Test ==="
echo ""

# Create test directory
mkdir -p "$TEST_DIR"
echo "✓ Created test directory: $TEST_DIR"

# Build the app
cd "$TIDYYY_DIR"
echo "✓ Building app..."
go build -o ./dist/tidyyy ./cmd/tidyyy

echo ""
echo "=== Step 1: Configure Settings ==="
echo "Opening settings window - please:"
echo "  1. Add $TEST_DIR as a watched folder"
echo "  2. Click 'Save and Close'"
echo ""
./dist/tidyyy --settings

echo ""
echo "=== Step 2: Check Config File ==="
CONFIG_FILE="$HOME/Library/Application Support/Tidyyy/config.json"
if [ -f "$CONFIG_FILE" ]; then
    echo "✓ Config file exists:"
    cat "$CONFIG_FILE"
else
    echo "✗ Config file not found at: $CONFIG_FILE"
    exit 1
fi

echo ""
echo "=== Step 3: Start Daemon ==="
echo "Daemon starting - watch for 'tidyyy started' message"
echo "You have 5 seconds to put a test file in: $TEST_DIR"
echo ""
timeout 5 ./dist/tidyyy || true

echo ""
echo "=== Step 4: Check Test Directory ==="
echo "Files in $TEST_DIR:"
ls -la "$TEST_DIR" 2>/dev/null || echo "Directory is empty"

echo ""
echo "=== Done ==="
