#!/bin/bash

# TurboScript Self-Hosted CI Setup Script
# This script helps you set up a self-hosted GitHub Actions runner

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

print_title() {
    echo ""
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
}

# Function to check prerequisites
check_prerequisites() {
    print_title "Checking Prerequisites"

    local missing_deps=()

    if ! command -v docker >/dev/null 2>&1; then
        missing_deps+=("docker")
    fi

    if ! command -v docker-compose >/dev/null 2>&1; then
        missing_deps+=("docker-compose")
    fi

    if ! command -v git >/dev/null 2>&1; then
        missing_deps+=("git")
    fi

    if [ ${#missing_deps[@]} -ne 0 ]; then
        print_error "Missing dependencies: ${missing_deps[*]}"
        print_status "Please install the missing dependencies:"
        for dep in "${missing_deps[@]}"; do
            case $dep in
                docker)
                    print_status "  - Docker: https://docs.docker.com/get-docker/"
                    ;;
                docker-compose)
                    print_status "  - Docker Compose: https://docs.docker.com/compose/install/"
                    ;;
                git)
                    print_status "  - Git: https://git-scm.com/downloads"
                    ;;
            esac
        done
        exit 1
    fi

    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running"
        print_status "Please start Docker and try again"
        exit 1
    fi

    print_success "All prerequisites satisfied"
}

# Function to create environment file
create_env_file() {
    print_title "Setting up Environment Configuration"

    local env_file=".github/runner/.env"

    if [ -f "$env_file" ]; then
        print_warning "Environment file already exists: $env_file"
        read -p "Do you want to recreate it? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_status "Using existing environment file"
            return 0
        fi
    fi

    # Copy template
    cp .github/runner/.env.example "$env_file"

    print_status "Environment file created: $env_file"
    print_warning "You need to configure the following variables:"
    echo ""
    echo "1. GITHUB_TOKEN - Your GitHub Personal Access Token"
    echo "   - Go to: https://github.com/settings/tokens"
    echo "   - Create a new token with 'repo' scope"
    echo ""
    echo "2. GITHUB_REPOSITORY - Your repository name"
    echo "   - Format: owner/repository (e.g., daison12006013/turboscript)"
    echo ""

    # Interactive configuration
    read -p "Enter your GitHub Personal Access Token: " -s github_token
    echo
    read -p "Enter your GitHub repository (owner/repo): " github_repo

    if [ -n "$github_token" ] && [ -n "$github_repo" ]; then
        # Update the .env file - using | as delimiter to avoid issues with / in repository names
        sed -i.bak "s|GITHUB_TOKEN=.*|GITHUB_TOKEN=$github_token|" "$env_file"
        sed -i.bak "s|GITHUB_REPOSITORY=.*|GITHUB_REPOSITORY=$github_repo|" "$env_file"
        rm "$env_file.bak"

        print_success "Environment file configured successfully"
    else
        print_warning "Skipping automatic configuration"
        print_status "Please edit $env_file manually with your settings"
    fi
}

# Function to build runner image
build_runner() {
    print_title "Building GitHub Actions Runner Image"

    cd .github/runner

    print_status "Building Docker image... (this may take a few minutes)"
    docker-compose build

    print_success "Runner image built successfully"

    cd - >/dev/null
}

# Function to start runner
start_runner() {
    print_title "Starting GitHub Actions Runner"

    cd .github/runner

    # Check if .env file is properly configured
    if ! grep -q "^GITHUB_TOKEN=\(ghp_\|github_pat_\)" .env 2>/dev/null; then
        print_error "GitHub token not configured in .env file"
        print_status "Please run: make ci-config"
        exit 1
    fi

    if ! grep -q "^GITHUB_REPOSITORY=.*/" .env 2>/dev/null; then
        print_error "GitHub repository not configured in .env file"
        print_status "Please run: make ci-config"
        exit 1
    fi

    print_status "Starting runner container..."
    docker-compose up -d

    # Wait for runner to start
    print_status "Waiting for runner to initialize..."
    sleep 10

    # Check if runner started successfully
    if docker-compose ps | grep -q "Up"; then
        print_success "GitHub Actions runner started successfully!"
        print_status "Runner logs:"
        docker-compose logs --tail 20 github-runner
    else
        print_error "Failed to start GitHub Actions runner"
        print_status "Check logs with: make ci-logs"
        exit 1
    fi

    cd - >/dev/null
}

# Function to verify setup
verify_setup() {
    print_title "Verifying Setup"

    cd .github/runner

    # Check container status
    if docker-compose ps | grep -q "Up"; then
        print_success "✅ Runner container is running"
    else
        print_error "❌ Runner container is not running"
        return 1
    fi

    # Check if runner process is active
    if docker-compose exec -T github-runner pgrep -f "Runner.Listener" >/dev/null 2>&1; then
        print_success "✅ Runner process is active"
    else
        print_warning "⚠️  Runner process status unknown"
    fi

    cd - >/dev/null

    print_status "Next steps:"
    echo "1. Go to your GitHub repository"
    echo "2. Navigate to Settings > Actions > Runners"
    echo "3. Verify your self-hosted runner appears as 'Online'"
    echo "4. Test the setup with: make ci-local"
}

# Function to show help
show_help() {
    cat << EOF
TurboScript Self-Hosted CI Setup

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    --skip-build            Skip building the runner image
    --skip-start            Skip starting the runner
    --config-only           Only create/update environment configuration

EXAMPLES:
    $0                      Complete setup (build and start)
    $0 --config-only        Only configure environment
    $0 --skip-build         Start with existing image

This script will:
1. Check prerequisites (Docker, git, etc.)
2. Create and configure .env file
3. Build the GitHub Actions runner Docker image
4. Start the self-hosted runner
5. Verify the setup

EOF
}

# Main function
main() {
    local skip_build=false
    local skip_start=false
    local config_only=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            --skip-build)
                skip_build=true
                shift
                ;;
            --skip-start)
                skip_start=true
                shift
                ;;
            --config-only)
                config_only=true
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    print_title "TurboScript Self-Hosted CI Setup"
    print_status "This script will help you set up a self-hosted GitHub Actions runner"

    # Check prerequisites
    check_prerequisites

    # Create/configure environment file
    create_env_file

    if [ "$config_only" = true ]; then
        print_success "Environment configuration complete"
        print_status "To build and start the runner, run: make ci-setup"
        exit 0
    fi

    # Build runner image
    if [ "$skip_build" != true ]; then
        build_runner
    fi

    # Start runner
    if [ "$skip_start" != true ]; then
        start_runner
        verify_setup
    fi

    print_title "Setup Complete!"
    print_success "Your self-hosted GitHub Actions runner is ready"

    echo ""
    print_status "Useful commands:"
    echo "  make ci-logs      - View runner logs"
    echo "  make ci-status    - Check runner status"
    echo "  make ci-restart   - Restart the runner"
    echo "  make ci-down      - Stop the runner"
    echo "  make ci-local     - Run CI locally"
    echo ""
    print_status "For more information, see: .github/runner/README.md"
}

main "$@"
