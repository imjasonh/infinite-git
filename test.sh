#!/bin/bash
set -ex

# Enable Git packet tracing for debugging
export GIT_TRACE_PACKET=1

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    if [ -n "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
    fi
}

# Trap EXIT to ensure cleanup happens
trap cleanup EXIT

# Kill any existing server in case it was not cleaned up
lsof -ti:9876 | xargs kill -9 2>/dev/null || true

echo "Starting infinite-git server..."
PORT=9876 REPO_PATH=/tmp/test-infinite-git go run . &
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
git --no-pager log --oneline

echo "Test complete!"
