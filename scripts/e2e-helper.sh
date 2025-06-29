#!/bin/bash

# E2E Test Helper Script for TurboScript
# This script helps manage server lifecycle for e2e testing

set -e

# Configuration
SERVER_HOST="localhost"
SERVER_PORT="7890"
BASE_URL="http://${SERVER_HOST}:${SERVER_PORT}"

# Database configuration will be read from turboscript.dev.yml

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if server is running
check_server() {
    if curl -s -f "$BASE_URL/" > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to wait for server
wait_for_server() {
    log_info "Waiting for TurboScript server to be ready..."
    local attempts=0
    local max_attempts=30

    while [ $attempts -lt $max_attempts ]; do
        if check_server; then
            log_info "Server is ready!"
            return 0
        fi

        attempts=$((attempts + 1))
        echo -n "."
        sleep 1
    done

    log_error "Server failed to start within timeout"
    return 1
}

# Function to start server
start_server() {
    log_info "Building TurboScript..."
    go build -o turboscript .

    log_info "Starting TurboScript server..."
    ./turboscript &
    echo $! > server.pid

    if wait_for_server; then
        log_info "TurboScript server started successfully (PID: $(cat server.pid))"
        return 0
    else
        log_error "Failed to start TurboScript server"
        return 1
    fi
}

# Function to stop server
stop_server() {
    if [ -f server.pid ]; then
        local pid=$(cat server.pid)
        log_info "Stopping TurboScript server (PID: $pid)..."
        kill $pid 2>/dev/null || true
        rm server.pid
        log_info "Server stopped"
    else
        log_warn "No server PID file found"
    fi
}

# Function to run e2e tests
run_e2e_tests() {
    log_info "Running Go-based E2E tests..."
    E2E_TEST=true go test -v -run TestE2EEndpoints ./...
    log_info "✅ E2E tests completed!"
}

# Function to run e2e benchmarks
run_e2e_benchmarks() {
    log_info "Running E2E performance benchmarks..."
    E2E_TEST=true go test -v -bench=BenchmarkE2EEndpoints -benchmem ./...
    log_info "✅ E2E benchmarks completed!"
}

# Main command handling
case "${1:-}" in
    "start")
        start_server
        ;;
    "stop")
        stop_server
        ;;
    "test")
        if ! check_server; then
            log_warn "Server not running, starting it first..."
            start_server
            trap stop_server EXIT
        fi
        run_e2e_tests
        ;;
    "bench")
        if ! check_server; then
            log_warn "Server not running, starting it first..."
            start_server
            trap stop_server EXIT
        fi
        run_e2e_benchmarks
        ;;
    "full")
        log_info "Running full E2E test cycle..."
        start_server
        trap stop_server EXIT
        run_e2e_tests
        run_e2e_benchmarks
        ;;
    "restart")
        stop_server
        start_server
        ;;
    "status")
        if check_server; then
            log_info "Server is running at $BASE_URL"
        else
            log_warn "Server is not running"
        fi
        ;;
    *)
        echo "Usage: $0 {start|stop|test|bench|full|restart|status}"
        echo ""
        echo "Commands:"
        echo "  start    - Start the TurboScript server"
        echo "  stop     - Stop the TurboScript server"
        echo "  test     - Run E2E tests (starts server if needed)"
        echo "  bench    - Run E2E benchmarks (starts server if needed)"
        echo "  full     - Start server, run tests and benchmarks, then stop"
        echo "  restart  - Restart the server"
        echo "  status   - Check if server is running"
        echo ""
        echo "Prerequisites:"
        echo "  - PostgreSQL database running with init.sql applied"
        echo "  - Go and Node.js dependencies installed"
        echo "  - Database configuration in turboscript.dev.yml"
        echo ""
        echo "Configuration:"
        echo "  Database connections are configured in turboscript.dev.yml"
        exit 1
        ;;
esac
