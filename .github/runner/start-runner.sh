#!/bin/bash

# GitHub Actions Self-Hosted Runner Startup Script

set -e

echo "🚀 Starting GitHub Actions Self-Hosted Runner for TurboScript"

# Check if required environment variables are set
if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ Error: GITHUB_TOKEN environment variable is required"
    echo "Please set your GitHub Personal Access Token with 'repo' scope"
    exit 1
fi

if [ -z "$GITHUB_REPOSITORY" ]; then
    echo "❌ Error: GITHUB_REPOSITORY environment variable is required"
    echo "Please set it to 'owner/repo' format (e.g., 'daison12006013/turboscript')"
    exit 1
fi

# Default values
RUNNER_NAME="${RUNNER_NAME:-turboscript-runner-$(hostname)}"
RUNNER_LABELS="${RUNNER_LABELS:-self-hosted,linux,docker,turboscript}"
RUNNER_GROUP="${RUNNER_GROUP:-default}"

echo "📋 Configuration:"
echo "  Repository: $GITHUB_REPOSITORY"
echo "  Runner Name: $RUNNER_NAME"
echo "  Runner Labels: $RUNNER_LABELS"
echo "  Runner Group: $RUNNER_GROUP"

# Get registration token from GitHub API
echo "🔑 Getting registration token from GitHub..."
REGISTRATION_TOKEN=$(curl -s -X POST \
  -H "Authorization: token $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  "https://api.github.com/repos/$GITHUB_REPOSITORY/actions/runners/registration-token" | jq -r .token)

if [ "$REGISTRATION_TOKEN" == "null" ] || [ -z "$REGISTRATION_TOKEN" ]; then
    echo "❌ Failed to get registration token. Please check:"
    echo "  - GitHub token has correct permissions (repo scope)"
    echo "  - Repository exists and is accessible"
    echo "  - Token is not expired"
    exit 1
fi

echo "✅ Registration token obtained successfully"

# Configure the runner
echo "⚙️  Configuring GitHub Actions runner..."
./config.sh \
    --url "https://github.com/$GITHUB_REPOSITORY" \
    --token "$REGISTRATION_TOKEN" \
    --name "$RUNNER_NAME" \
    --labels "$RUNNER_LABELS" \
    --runnergroup "$RUNNER_GROUP" \
    --work "_work" \
    --replace \
    --unattended

echo "✅ Runner configured successfully"

# Cleanup function for graceful shutdown
cleanup() {
    echo "🛑 Stopping runner..."

    # Remove the runner from GitHub
    echo "🗑️  Removing runner from GitHub..."
    REMOVE_TOKEN=$(curl -s -X POST \
      -H "Authorization: token $GITHUB_TOKEN" \
      -H "Accept: application/vnd.github.v3+json" \
      "https://api.github.com/repos/$GITHUB_REPOSITORY/actions/runners/remove-token" | jq -r .token)

    if [ "$REMOVE_TOKEN" != "null" ] && [ -n "$REMOVE_TOKEN" ]; then
        ./config.sh remove --token "$REMOVE_TOKEN"
        echo "✅ Runner removed from GitHub"
    else
        echo "⚠️  Could not get removal token, runner may need manual cleanup"
    fi

    exit 0
}

# Set up signal handlers for graceful shutdown
trap cleanup SIGTERM SIGINT

# Start the runner
echo "🏃 Starting GitHub Actions runner..."
echo "Runner is ready to accept jobs from: https://github.com/$GITHUB_REPOSITORY/actions"
echo "Press Ctrl+C to stop the runner"

# Run the runner with proper signal handling
./run.sh & wait $!
