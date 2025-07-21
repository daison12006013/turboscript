# Self-Hosted GitHub Actions Runner Setup

This guide walks you through setting up a self-hosted GitHub Actions runner for TurboScript development and CI/CD workflows.

## Overview

The self-hosted runner provides several advantages:

- **Local Development**: Test CI workflows locally before pushing
- **Resource Control**: Use your own hardware resources
- **Docker Access**: Full Docker-in-Docker capabilities for container testing
- **Custom Environment**: Pre-configured with TurboScript dependencies

## Prerequisites

Before starting, ensure you have:

- [Docker](https://docs.docker.com/get-docker/) installed and running
- [Docker Compose](https://docs.docker.com/compose/install/) installed
- [Git](https://git-scm.com/downloads) installed
- Repository admin access to create self-hosted runners

## Quick Start

```bash
# Start the setup process
make ci-setup
```

This command will:

1. Check prerequisites
2. Guide you through environment configuration
3. Build the runner Docker image
4. Start the self-hosted runner
5. Verify the setup

## Step-by-Step Setup

### 1. Create GitHub Personal Access Token

1. Go to [GitHub Settings > Personal Access Tokens](https://github.com/settings/tokens)
2. Click "Generate new token (classic)"
3. Set an expiration date (recommended: 90 days or custom)
4. Select the following scopes:
   - `repo` (Full control of private repositories)
   - `admin:repo_hook` (Repository hooks administration)
   - `workflow` (Update GitHub Action workflows)

5. Click "Generate token"
6. **Important**: Copy the token immediately - you won't see it again!

### 2. Configure Environment Variables

Run the configuration command:

```bash
make ci-config
```

Or manually edit `.github/runner/.env`:

```bash
# Required: GitHub Authentication
GITHUB_TOKEN=your_complete_github_personal_access_token_here
GITHUB_REPOSITORY=daison12006013/turboscript

# Optional: Runner Configuration
RUNNER_NAME=turboscript-local-runner
RUNNER_LABELS=self-hosted,linux,docker,turboscript,local,dev
```

### 3. Build and Start Runner

```bash
# Complete setup with build and start
make ci-setup

# Or build only
make ci-build

# Or start with existing image
make ci-start
```

### 4. Verify Setup

Check that the runner is registered:

1. Go to your repository on GitHub
2. Navigate to **Settings** > **Actions** > **Runners**
3. Verify your runner appears as "Online"

## Troubleshooting

### Token Authentication Failed

**Error**: `❌ Failed to get registration token`

**Solutions**:

1. **Check token validity**: Ensure your GitHub token hasn't expired
2. **Verify permissions**: Token must have `repo` and `admin:repo_hook` scopes
3. **Complete token**: Ensure the full token is copied (usually 40+ characters)
4. **Repository access**: Verify you have admin access to the repository

**Fix the token**:

```bash
# Edit the environment file
nano .github/runner/.env

# Update the GITHUB_TOKEN line with your complete token
GITHUB_TOKEN=github_pat_11ABC6QJY0...your_complete_token_here

# Restart the runner
make ci-restart
```

### Docker Permission Issues

**Error**: `usermod: group 'docker' does not exist`

This has been fixed in the latest version. If you encounter this:

```bash
# Rebuild the image
make ci-build
```

### Container Startup Issues

**Error**: Runner containers not starting

```bash
# Check Docker status
docker info

# Check container logs
make ci-logs

# Check container status
make ci-status

# Restart if needed
make ci-restart
```

### Port Conflicts

**Error**: PostgreSQL port already in use

The CI PostgreSQL runs on port 5433 to avoid conflicts. If this port is still in use:

```bash
# Check what's using the port
lsof -i :5433

# Or change the port in docker-compose.yml
nano .github/runner/docker-compose.yml
```

## Available Commands

| Command | Description |
|---------|-------------|
| `make ci-setup` | Complete setup (build and start) |
| `make ci-config` | Configure environment variables |
| `make ci-build` | Build runner image only |
| `make ci-start` | Start existing runner |
| `make ci-restart` | Restart the runner |
| `make ci-stop` | Stop the runner |
| `make ci-down` | Stop and remove containers |
| `make ci-logs` | View runner logs |
| `make ci-status` | Check runner status |
| `make ci-local` | Run CI workflows locally |

## CI Environment

The runner environment includes:

### Pre-installed Tools

- **Go 1.23.10** with module support
- **Node.js 20 LTS** with npm
- **Docker CLI** for container operations
- **PostgreSQL Client** for database testing
- **Newman** for Postman API testing
- **Security Tools**: gosec, nancy, govulncheck
- **Performance Tools**: hey for load testing

### Available Services

- **PostgreSQL 16** (port 5433)
- **Docker-in-Docker** capabilities
- **Persistent workspace** storage

### Environment Variables

```bash
GO_VERSION=1.23.10
POSTGRES_VERSION=16
NODE_ENV=test
CI=true
DOCKER_ENV=true
```

## Security Considerations

1. **Token Security**: Store tokens securely and rotate regularly
2. **Repository Access**: Only grant necessary permissions
3. **Network Isolation**: Runner uses isolated Docker network
4. **Resource Limits**: Consider setting Docker resource constraints

## Architecture

```
┌─────────────────────────────────────────┐
│ GitHub Actions Self-Hosted Runner       │
├─────────────────────────────────────────┤
│ • Ubuntu 22.04 base                    │
│ • Go 1.23.10 + Node.js 20               │
│ • Docker-in-Docker support             │
│ • Security scanning tools              │
│ • Performance testing tools            │
└─────────────────────────────────────────┘
                    │
                    ├── Docker Socket
                    ├── Persistent Workspace
                    └── PostgreSQL (CI Testing)
```

## Next Steps

After successful setup:

1. **Test Local CI**: `make ci-local`
2. **Review Workflows**: Check `.github/workflows/`
3. **Configure Alerts**: Set up notifications for CI failures
4. **Monitor Performance**: Use built-in performance tools

## Best Practices

1. **Regular Updates**: Keep the runner image updated
2. **Token Rotation**: Rotate GitHub tokens every 90 days
3. **Resource Monitoring**: Monitor Docker resource usage
4. **Backup Configuration**: Keep `.env` file backed up securely
5. **CI Optimization**: Optimize workflows for faster feedback

## Support

For issues or questions:

- Check the logs: `make ci-logs`
- Review this documentation
- Check GitHub Actions documentation
- Open an issue in the repository

---

*This setup provides a robust, local CI environment that mirrors production GitHub Actions while giving you full control over the execution environment.*
