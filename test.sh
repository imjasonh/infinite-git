#!/bin/bash
set -e

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    if [ -n "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
    fi
}

# Trap EXIT to ensure cleanup happens
trap cleanup EXIT

echo "Starting infinite-git server..."
go run . -addr :9876 -repo /tmp/test-infinite-git &
SERVER_PID=$!

# Give server time to start
sleep 2

# Create test directory
TEST_DIR=$(mktemp -d)
cd "$TEST_DIR"

echo "Cloning repository..."
git clone http://localhost:9876 test-repo
cd test-repo

echo "Initial clone complete. Files:"
ls -la

echo "Pulling again..."
git pull

echo "Files after first pull:"
ls -la

echo "Pulling once more..."
git pull

echo "Files after second pull:"
ls -la

echo "Checking commits..."
git log --oneline

# Test directory cleanup
cd /
rm -rf "$TEST_DIR"

echo "Test complete!"