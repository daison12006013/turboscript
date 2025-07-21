#!/bin/bash

# TurboScript Act Runner Script
# This script uses 'act' to run GitHub Actions locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if act is installed
check_act() {
    if ! command -v act >/dev/null 2>&1; then
        print_error "Act is not installed"
        print_status "Install act using one of these methods:"
        print_status "  macOS: brew install act"
        print_status "  Linux: curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash"
        print_status "  Manual: https://github.com/nektos/act/releases"
        exit 1
    fi
}

# Check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running"
        print_status "Please start Docker and try again"
        exit 1
    fi
}

# Function to show help
show_help() {
    cat << EOF
TurboScript Act Runner - Run GitHub Actions locally using Act

Usage: $0 [OPTIONS] [JOB_NAME]

OPTIONS:
    -h, --help              Show this help message
    -l, --list              List available jobs/workflows
    -n, --dry-run           Dry run (show what would be executed)
    -v, --verbose           Enable verbose output
    --pull                  Pull latest runner images
    --platform PLATFORM    Specify platform (default: ubuntu-latest=catthehacker/ubuntu:act-latest)

JOB_NAME:
    If specified, run only the specific job. Available jobs:
    - lint                  Lint and format checks
    - test                  Unit tests
    - build                 Build tests
    - e2e                   End-to-end tests
    - postman-contract      Postman API contract tests
    - security              Security scan
    - performance           Performance tests (only on main branch)
    - docker                Docker build test
    - summary               CI summary report

EXAMPLES:
    $0                      Run all jobs
    $0 test                 Run only unit tests
    $0 --list               List available workflows and jobs
    $0 --dry-run            Show what would be executed
    $0 lint test build      Run specific jobs

NOTES:
    - Act runs GitHub Actions workflows locally using Docker
    - Some features may not work exactly like GitHub Actions
    - Set GITHUB_TOKEN environment variable for API access if needed

EOF
}

# Main function
main() {
    local jobs=()
    local list_mode=false
    local dry_run=false
    local verbose=false
    local pull_images=false
    local platform="ubuntu-latest=catthehacker/ubuntu:act-latest"

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -l|--list)
                list_mode=true
                shift
                ;;
            -n|--dry-run)
                dry_run=true
                shift
                ;;
            -v|--verbose)
                verbose=true
                shift
                ;;
            --pull)
                pull_images=true
                shift
                ;;
            --platform)
                platform="$2"
                shift 2
                ;;
            lint|test|build|e2e|postman-contract|security|performance|docker|summary)
                jobs+=("$1")
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Check prerequisites
    check_act
    check_docker

    # List mode
    if [ "$list_mode" = true ]; then
        print_status "Available workflows and jobs:"
        act -l
        exit 0
    fi

    # Pull images if requested
    if [ "$pull_images" = true ]; then
        print_status "Pulling latest runner images..."
        docker pull catthehacker/ubuntu:act-latest
    fi

    # Build act command
    local act_cmd="act"

    # Add platform
    act_cmd="$act_cmd --platform $platform"

    # Add dry run
    if [ "$dry_run" = true ]; then
        act_cmd="$act_cmd --dry-run"
    fi

    # Add verbose
    if [ "$verbose" = true ]; then
        act_cmd="$act_cmd --verbose"
    fi

    # Add environment variables
    act_cmd="$act_cmd --env GO_VERSION=1.23.10"
    act_cmd="$act_cmd --env POSTGRES_VERSION=16"

    # If specific jobs are requested
    if [ ${#jobs[@]} -gt 0 ]; then
        for job in "${jobs[@]}"; do
            print_status "Running job: $job"
            eval "$act_cmd --job $job"
        done
    else
        # Run all jobs
        print_status "Running complete CI pipeline with act..."
        eval "$act_cmd"
    fi

    print_success "Act execution completed"
}

main "$@"
