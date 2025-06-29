#!/bin/bash

# Real-Time Communication Test Runner
# This script runs all the WebSocket and SSE tests locally for validation

set -e

echo "🚀 TurboScript Real-Time Communication Test Suite"
echo "================================================="

# Check if server is running
echo "🔍 Checking if TurboScript server is running on port 7890..."
if ! curl -s http://localhost:7890/ > /dev/null; then
    echo "❌ Server is not running on port 7890"
    echo "Please start the server with: ./turboscript --config turboscript.dev.yml"
    exit 1
fi

echo "✅ Server is running"

# Test 1: JavaScript WebSocket Multi-Connection Tests
echo ""
echo "📡 Running JavaScript WebSocket Multi-Connection Tests..."
echo "--------------------------------------------------------"
if node scripts/test-multi-websocket.js; then
    echo "✅ JavaScript WebSocket tests passed"
else
    echo "❌ JavaScript WebSocket tests failed"
    exit 1
fi

# Test 2: JavaScript SSE Multi-Client Tests
echo ""
echo "📡 Running JavaScript SSE Multi-Client Tests..."
echo "-----------------------------------------------"
if node scripts/test-sse-multi.js; then
    echo "✅ JavaScript SSE tests passed"
else
    echo "❌ JavaScript SSE tests failed"
    exit 1
fi

# Test 3: Go WebSocket Tests
echo ""
echo "🧪 Running Go WebSocket Tests..."
echo "--------------------------------"
if E2E_TEST=true go test -v -run TestMultipleWebSocketConnections ./internal/tests/; then
    echo "✅ Go WebSocket connection tests passed"
else
    echo "❌ Go WebSocket connection tests failed"
    exit 1
fi

if E2E_TEST=true go test -v -run TestWebSocketMessageTypes ./internal/tests/; then
    echo "✅ Go WebSocket message type tests passed"
else
    echo "❌ Go WebSocket message type tests failed"
    exit 1
fi

# Test 4: Go SSE Tests
echo ""
echo "🧪 Running Go SSE Tests..."
echo "--------------------------"
if E2E_TEST=true go test -v -run TestMultipleSSEConnections ./internal/tests/; then
    echo "✅ Go SSE connection tests passed"
else
    echo "❌ Go SSE connection tests failed"
    exit 1
fi

if E2E_TEST=true go test -v -run TestSSEConnectionFlow ./internal/tests/; then
    echo "✅ Go SSE flow tests passed"
else
    echo "❌ Go SSE flow tests failed"
    exit 1
fi

# Test 5: Benchmarks (optional)
echo ""
echo "⚡ Running Performance Benchmarks (optional)..."
echo "-----------------------------------------------"
echo "Running WebSocket benchmarks..."
E2E_TEST=true go test -bench=BenchmarkWebSocketConnections -benchmem ./internal/tests/ || echo "WebSocket benchmarks completed (some may have failed)"

echo "Running SSE benchmarks..."
E2E_TEST=true go test -bench=BenchmarkSSEConnections -benchmem ./internal/tests/ || echo "SSE benchmarks completed (some may have failed)"

# Summary
echo ""
echo "🎉 All Real-Time Communication Tests Completed Successfully!"
echo "============================================================"
echo "✅ JavaScript WebSocket Tests"
echo "✅ JavaScript SSE Tests"
echo "✅ Go WebSocket Tests"
echo "✅ Go SSE Tests"
echo "⚡ Performance Benchmarks"
echo ""
echo "Your TurboScript real-time communication system is working correctly!"
echo ""
echo "To run individual test suites:"
echo "  WebSocket JS:  node scripts/test-multi-websocket.js"
echo "  SSE JS:        node scripts/test-sse-multi.js"
echo "  WebSocket Go:  E2E_TEST=true go test -v -run TestMultipleWebSocketConnections ./internal/tests/"
echo "  SSE Go:        E2E_TEST=true go test -v -run TestMultipleSSEConnections ./internal/tests/"
