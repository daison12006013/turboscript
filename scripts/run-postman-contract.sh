#!/bin/bash

# TurboScript Postman Contract Runner
# Executes Postman collection after E2E tests to validate API contract

set -e

echo "🚀 TurboScript Complete API Test Suite"
echo "======================================="

# Configuration
COLLECTION_FILE="postman/TurboScript-Complete-API.postman_collection.json"
ENVIRONMENT_FILE="postman/TurboScript-Complete.postman_environment.json"
REPORT_DIR="postman/reports"
BASE_URL="${BASE_URL:-http://127.0.0.1:7890}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."

    # Check if newman is available locally or globally
    if ! npx newman --version &> /dev/null && ! command -v newman &> /dev/null; then
        log_error "Newman is not available. Installing via npm..."
        npm install newman

        if ! npx newman --version &> /dev/null; then
            log_error "Failed to install Newman. Please install manually: npm install newman"
            exit 1
        fi
    fi

    log_success "Dependencies checked"
}

# Check if server is running
check_server() {
    log_info "Checking if TurboScript server is running at $BASE_URL..."

    if curl -s "$BASE_URL" > /dev/null 2>&1; then
        log_success "Server is running at $BASE_URL"
    else
        log_error "Server is not running at $BASE_URL"
        log_info "Please start the server with: make up"
        exit 1
    fi
}

# Create reports directory
setup_reports() {
    log_info "Setting up reports directory..."
    mkdir -p "$REPORT_DIR"
    log_success "Reports directory ready: $REPORT_DIR"
}

# Run Postman collection
run_collection() {
    log_info "Running Postman collection..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local json_report="$REPORT_DIR/report_$timestamp.json"

    # Run newman with CLI and JSON reporters using npx
    npx newman run "$COLLECTION_FILE" \
        --environment "$ENVIRONMENT_FILE" \
        --reporters cli,json \
        --reporter-json-export "$json_report" \
        --env-var "base_url=$BASE_URL" \
        --timeout-request 30000 \
        --timeout-script 60000 \
        --bail || {
            log_error "Postman collection execution failed"
            log_info "Check the reports for details:"
            log_info "  HTML Report: $report_file"
            log_info "  JSON Report: $json_report"
            exit 1
        }

    log_success "Postman collection executed successfully"
    log_info "Reports generated:"
    log_info "  JSON Report: $json_report"
}

# Main execution
main() {
    echo ""
    log_info "Starting Postman contract execution..."
    echo ""

    check_dependencies
    check_server
    setup_reports
    run_collection

    echo ""
    log_success "Postman contract execution completed successfully! 🎉"
    echo ""
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --install      Install Newman and dependencies only"
        echo ""
        echo "Environment Variables:"
        echo "  BASE_URL       TurboScript server URL (default: http://localhost:7890)"
        echo ""
        echo "Examples:"
        echo "  $0                                    # Run with default settings"
        echo "  BASE_URL=http://staging.example.com $0  # Run against staging server"
        exit 0
        ;;
    --install)
        log_info "Installing Newman and dependencies..."
        npm install newman
        log_success "Installation completed"
        exit 0
        ;;
    *)
        main
        ;;
esac
