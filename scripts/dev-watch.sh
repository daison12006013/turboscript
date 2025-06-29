#!/bin/bash

# Development watch script for TurboScript
# This script runs frontend and backend builds in parallel

echo "🚀 Starting TurboScript development environment..."

# Function to handle cleanup on exit
cleanup() {
    echo "🧹 Cleaning up processes..."
    pkill -f "npx tsx scripts/build-frontend.ts"
    pkill -f "air"
    exit 0
}

# Set up signal handling
trap cleanup SIGINT SIGTERM

# Build frontend initially
echo "🏗️  Initial frontend build..."
npx tsx scripts/build-frontend.ts

# Start frontend watch in background
echo "👀 Starting frontend watcher..."
npx tsx scripts/build-frontend.ts --watch &
FRONTEND_PID=$!

# Start Air for Go hot reloading
echo "🔥 Starting Go hot reloader (Air)..."
air -c .air.toml &
AIR_PID=$!

echo "✅ Development environment ready!"
echo "   - Frontend watcher PID: $FRONTEND_PID"
echo "   - Air (Go) watcher PID: $AIR_PID"
echo "   - Press Ctrl+C to stop all processes"

# Wait for any process to exit
wait $FRONTEND_PID $AIR_PID
