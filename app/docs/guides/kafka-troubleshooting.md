# Kafka Troubleshooting Guide

This guide covers common Kafka issues in the TurboScript development environment and their solutions.

## Common Issue: Kafka Exit Code 1 on Restart

### Problem Description

When running `make restart`, the Kafka container may exit with code 1 due to a `NodeExistsException`. This occurs because:

1. Kafka tries to register itself with Zookeeper using the same broker ID
2. Zookeeper still has ephemeral nodes from the previous Kafka session
3. Kafka fails to start with the error: `KeeperErrorCode = NodeExists`

### Error Example

```log
[2025-07-17 02:24:26,863] ERROR Exiting Kafka due to fatal exception during startup. (kafka.Kafka$)
org.apache.zookeeper.KeeperException$NodeExistsException: KeeperErrorCode = NodeExists
        at org.apache.zookeeper.KeeperException.create(KeeperException.java:126)
        at kafka.zk.KafkaZkClient$CheckedEphemeral.getAfterNodeExists(KafkaZkClient.scala:2189)
        ...
```

### Solution 1: Use Clean Restart

Instead of `make restart`, use the new clean restart command:

```bash
make restart-clean
```

This command:

- Stops all containers completely (`docker-compose down`)
- Waits for services to fully stop
- Starts services fresh (`docker-compose up -d`)

### Solution 2: Manual Restart with Container Removal

```bash
# Stop and remove all containers
docker-compose -f docker-compose.dev.yml down

# Wait a few seconds
sleep 3

# Start services
docker-compose -f docker-compose.dev.yml up -d
```

### Solution 3: Kafka-Specific Restart

If only Kafka is having issues:

```bash
# Stop Kafka and Zookeeper
docker-compose -f docker-compose.dev.yml stop kafka zookeeper

# Remove containers
docker-compose -f docker-compose.dev.yml rm -f kafka zookeeper

# Start Zookeeper first, then Kafka
docker-compose -f docker-compose.dev.yml up -d zookeeper
sleep 10
docker-compose -f docker-compose.dev.yml up -d kafka
```

## Configuration Improvements

The Docker Compose configuration has been updated with better Kafka restart handling:

```yaml
kafka:
  environment:
    # Fix for restart issues - allow Kafka to handle stale Zookeeper registrations
    KAFKA_ZOOKEEPER_SESSION_TIMEOUT_MS: 6000
    KAFKA_ZOOKEEPER_CONNECTION_TIMEOUT_MS: 6000
    KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0
    # Ensure proper cleanup on shutdown
    KAFKA_CONTROLLED_SHUTDOWN_ENABLE: true
    KAFKA_CONTROLLED_SHUTDOWN_MAX_RETRIES: 3
    KAFKA_CONTROLLED_SHUTDOWN_RETRY_BACKOFF_MS: 5000
  restart: unless-stopped
  healthcheck:
    start_period: 60s  # Allow more time for initial startup
    retries: 5         # More retry attempts
```

## Verifying Kafka Status

### Check Container Status

```bash
# Check all containers
docker-compose -f docker-compose.dev.yml ps

# Check Kafka specifically
docker ps --filter "name=kafka"
```

### Check Kafka Logs

```bash
# Recent logs
docker logs turboscript-kafka-1 --tail=20

# Follow logs in real-time
docker logs turboscript-kafka-1 -f
```

### Test Kafka Connectivity

```bash
# List topics (should return empty list if working)
docker exec turboscript-kafka-1 kafka-topics --bootstrap-server localhost:9092 --list

# Create a test topic
docker exec turboscript-kafka-1 kafka-topics --bootstrap-server localhost:9092 --create --topic test-topic --partitions 1 --replication-factor 1

# Delete test topic
docker exec turboscript-kafka-1 kafka-topics --bootstrap-server localhost:9092 --delete --topic test-topic
```

### Run Integration Tests

```bash
# Test Kafka integration in the application
docker exec turboscript-app-dev-1 /bin/bash -c "cd /app && DOCKER_ENV=true go test -v ./internal/server -run TestKafkaManager_Integration -timeout 60s"
```

## Prevention Best Practices

1. **Always use clean restart**: Use `make restart-clean` instead of `make restart` when you suspect state issues
2. **Monitor container health**: Check container status before assuming services are ready
3. **Use proper shutdown**: Use `make down` followed by `make up` for clean environment resets
4. **Check logs proactively**: Monitor Kafka logs during startup to catch issues early

## Related Services

### Zookeeper Health

Kafka depends on Zookeeper. If Kafka fails, check Zookeeper status:

```bash
# Check Zookeeper logs
docker logs turboscript-zookeeper-1 --tail=20

# Test Zookeeper connectivity
docker exec turboscript-zookeeper-1 nc -z localhost 2181 && echo "Zookeeper is running"
```

### Network Issues

If containers can't communicate:

```bash
# Check Docker network
docker network ls
docker network inspect turboscript_default

# Restart network if needed
docker-compose -f docker-compose.dev.yml down
docker network prune
docker-compose -f docker-compose.dev.yml up -d
```

## Troubleshooting Commands Reference

| Command | Purpose |
|---------|---------|
| `make restart-clean` | Clean restart all services |
| `make down && make up` | Manual clean restart |
| `docker-compose -f docker-compose.dev.yml ps` | Check container status |
| `docker logs turboscript-kafka-1` | View Kafka logs |
| `docker exec turboscript-kafka-1 kafka-topics --bootstrap-server localhost:9092 --list` | Test Kafka connectivity |

## When to Use Each Solution

- **Use `make restart-clean`**: First choice for any restart-related issues
- **Use manual restart**: When you need more control over the process
- **Use Kafka-specific restart**: When only Kafka is having issues and other services are stable
- **Use full rebuild**: When configuration changes require container rebuilds (`make rebuild`)
