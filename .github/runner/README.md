# Self-Hosted GitHub Actions Runner for TurboScript

This directory contains the setup for a self-hosted GitHub Actions runner using Docker. This allows you to run your CI/CD pipeline locally, saving costs and providing more control over the execution environment.

## 🚀 Quick Setup

### 1. Prerequisites

- Docker and Docker Compose installed
- GitHub Personal Access Token with `repo` scope
- Repository admin access

### 2. Create GitHub Personal Access Token

1. Go to [GitHub Settings > Developer settings > Personal access tokens](https://github.com/settings/tokens)
2. Click "Generate new token (classic)"
3. Select scopes:
   - `repo` (Full control of private repositories)
   - `workflow` (Update GitHub Action workflows)
   - `admin:repo_hook` (Admin access to repository hooks)
4. Copy the generated token

### 3. Configure Environment

```bash
# Copy the example environment file
cp .env.example .env

# Edit the .env file with your settings
vim .env
```

**Required settings in `.env`:**

```bash
GITHUB_TOKEN=ghp_your_personal_access_token_here
GITHUB_REPOSITORY=your-username/turboscript
```

### 4. Start the Runner

```bash
# Navigate to the runner directory
cd .github/runner

# Start the self-hosted runner
docker-compose up -d

# View logs
docker-compose logs -f github-runner
```

### 5. Verify Runner Registration

1. Go to your repository on GitHub
2. Navigate to Settings > Actions > Runners
3. You should see your self-hosted runner listed as "Online"

## 🛠️ Makefile Integration

Add these commands to your project's Makefile for easy CI execution:

```bash
# In the project root Makefile
ci-setup: ## Setup self-hosted GitHub runner
 cd .github/runner && docker-compose up -d

ci-down: ## Stop self-hosted GitHub runner
 cd .github/runner && docker-compose down

ci-logs: ## View GitHub runner logs
 cd .github/runner && docker-compose logs -f github-runner

ci-local: ## Run CI pipeline locally using act
 act --platform ubuntu-latest=catthehacker/ubuntu:act-latest
```

## 📋 Available Commands

### Runner Management

```bash
# Start the runner
docker-compose up -d

# Stop the runner
docker-compose down

# View logs
docker-compose logs -f github-runner

# Restart the runner
docker-compose restart github-runner

# Rebuild the runner image
docker-compose build --no-cache

# Remove everything (including volumes)
docker-compose down -v
```

### Debug and Troubleshooting

```bash
# Connect to the runner container
docker-compose exec github-runner bash

# Check runner status
docker-compose exec github-runner ps aux | grep Runner

# View runner configuration
docker-compose exec github-runner cat .runner

# Check Docker socket access
docker-compose exec github-runner docker ps
```

## 🔧 Configuration Options

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITHUB_TOKEN` | Yes | - | Personal Access Token |
| `GITHUB_REPOSITORY` | Yes | - | Repository in owner/repo format |
| `RUNNER_NAME` | No | `turboscript-docker-runner` | Custom runner name |
| `RUNNER_LABELS` | No | `self-hosted,linux,docker,turboscript,local` | Runner labels |
| `RUNNER_GROUP` | No | `default` | Runner group |

### Runner Labels

The runner is configured with these labels by default:

- `self-hosted`: Indicates it's a self-hosted runner
- `linux`: Operating system
- `docker`: Has Docker capabilities
- `turboscript`: Project-specific label
- `local`: Indicates local development runner

### Docker Configuration

The runner container includes:

- ✅ Ubuntu 22.04 base image
- ✅ Go 1.23.10
- ✅ Node.js 20 LTS
- ✅ Docker CLI (Docker-in-Docker support)
- ✅ PostgreSQL client
- ✅ All CI tools (newman, hey, gosec, etc.)

## 🔄 Using with CI Workflows

### Targeting Self-Hosted Runner

Modify your `.github/workflows/ci.yml` to use your self-hosted runner:

```yaml
jobs:
  test:
    runs-on: [self-hosted, linux, turboscript]  # Use your runner labels
    steps:
      # Your existing workflow steps
```

### Conditional Runner Usage

Use different runners based on conditions:

```yaml
jobs:
  test:
    runs-on: ${{ github.event_name == 'push' && 'ubuntu-latest' || 'self-hosted' }}
```

## 🐛 Troubleshooting

### Common Issues

**1. Runner not appearing in GitHub**

- Check GitHub token permissions
- Verify repository name format
- Check runner logs: `docker-compose logs github-runner`

**2. Docker socket permission errors**

```bash
# Fix Docker socket permissions
sudo chmod 666 /var/run/docker.sock
```

**3. Runner fails to start**

```bash
# Check environment variables
docker-compose config

# Rebuild with latest dependencies
docker-compose build --no-cache
```

**4. CI jobs fail with "No space left on device"**

```bash
# Clean up Docker resources
docker system prune -af
docker volume prune -f
```

### Logs and Debugging

```bash
# View detailed runner logs
docker-compose logs -f github-runner

# Check runner health
docker-compose ps

# Monitor resource usage
docker stats turboscript-github-runner

# Access runner filesystem
docker-compose exec github-runner bash
```

## 🔒 Security Considerations

### Token Security

- Use a dedicated GitHub account for CI
- Limit token scope to required permissions only
- Rotate tokens regularly
- Store tokens securely (use `.env` file, never commit)

### Network Security

- Runner runs in isolated Docker network
- Only required ports exposed
- Docker socket access controlled

### Resource Limits

```yaml
# Add to docker-compose.yml service
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 4G
    reservations:
      memory: 2G
```

## 📊 Monitoring and Maintenance

### Health Checks

- Built-in health check monitors runner process
- Automatic restart on failure
- PostgreSQL health monitoring for CI tests

### Log Rotation

```bash
# Configure log rotation in docker-compose.yml
logging:
  driver: "json-file"
  options:
    max-size: "100m"
    max-file: "3"
```

### Updates

```bash
# Update runner to latest version
docker-compose pull
docker-compose up -d
```

## 💡 Best Practices

1. **Resource Management**
   - Monitor disk space regularly
   - Clean up Docker images periodically
   - Set appropriate resource limits

2. **Security**
   - Use least-privilege GitHub tokens
   - Regularly update base images
   - Monitor runner activity

3. **Reliability**
   - Use health checks
   - Configure automatic restarts
   - Monitor runner logs

4. **Cost Optimization**
   - Use self-hosted runner for heavy workloads
   - Fall back to GitHub-hosted for simple tasks
   - Implement conditional runner selection

## 🚀 Advanced Usage

### Multiple Runners

```bash
# Scale to multiple runners
docker-compose up -d --scale github-runner=3
```

### Custom Runner Images

```dockerfile
# Extend the base runner image
FROM turboscript-github-runner:latest
RUN apt-get update && apt-get install -y custom-tools
```

### Integration with Local Development

```bash
# Mount local repository for faster development
volumes:
  - ${PWD}:/home/runner/_work/turboscript/turboscript
```

This setup provides a complete self-hosted CI/CD solution that can run all your GitHub Actions workflows locally while maintaining compatibility with your existing CI configuration.
