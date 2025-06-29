#!/bin/bash

# TurboScript Development Build Script
# This script builds both frontend and Go binary for hot reloading

set -e  # Exit on any error

echo "🔄 Starting TurboScript development build..."

# Build frontend assets
echo "🎨 Building frontend assets..."
if npm run build:frontend:dev; then
    echo "✅ Frontend build completed successfully"
else
    echo "❌ Frontend build failed"
    exit 1
fi

# Build Go binary
echo "🏗️  Building Go binary..."
if go build -o ./.tmp/main .; then
    echo "✅ Go build completed successfully"
else
    echo "❌ Go build failed"
    exit 1
fi

echo "🚀 TurboScript development build completed!"
