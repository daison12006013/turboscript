# Deployment Guide

This guide covers deploying TurboScript applications to various production environments.

## Production Build Process

### Building for Distribution

TurboScript provides a complete build system for production deployment:

```bash
# Build complete distribution package
make build-dist
```

This creates a `dist/` folder containing:

- **Compiled JavaScript files**: Optimized TypeScript compilation via esbuild
- **Cross-platform binaries**: `turboscript` (current platform) and `turboscript-linux`
- **Production configuration**: Optimized `turboscript.yml`
- **Deployment script**: `runner.sh` with database configuration
- **Static assets**: All necessary files for production

### Distribution Structure

```text
dist/
├── app/                    # Compiled JavaScript from TypeScript
│   ├── routes/
│   ├── utils/
│   ├── queue/
│   └── global.d.ts
├── turboscript             # Native binary for current platform
├── turboscript-linux       # Cross-compiled Linux binary
├── turboscript.yml         # Production configuration
└── runner.sh               # Production deployment script
```

## Production Deployment Options

### 1. Docker Deployment (Recommended)

#### Production Dockerfile

```dockerfile
# Multi-stage build for optimized production image
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy Go modules and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build optimized binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-w -s -extldflags "-static"' \
    -o turboscript .

# Production stage
FROM alpine:latest

# Install certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary and distribution files
COPY --from=builder /app/turboscript .
COPY --from=builder /app/dist ./dist

# Create non-root user for security
RUN adduser -D -s /bin/sh turboscript
USER turboscript

EXPOSE 8080

CMD ["./turboscript"]
```

#### Production Docker Compose

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile.prod
    ports:
      - "8080:8080"
    environment:
      - JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET}
      - JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=${DB_NAME}
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
    depends_on:
      - postgres
    restart: unless-stopped
    volumes:
      - ./logs:/app/logs
    networks:
      - turboscript-network

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=${DB_NAME}
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    restart: unless-stopped
    networks:
      - turboscript-network

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - app
    restart: unless-stopped
    networks:
      - turboscript-network

volumes:
  postgres_data:

networks:
  turboscript-network:
    driver: bridge
```

#### Production Environment File

```bash
# .env.prod
JWT_ACCESS_SECRET=your_super_secure_access_secret_64_characters_minimum
JWT_REFRESH_SECRET=your_super_secure_refresh_secret_64_characters_minimum

# Database
DB_NAME=turboscript_prod
DB_USER=turboscript_user
DB_PASSWORD=secure_database_password

# Email
SMTP_HOST=smtp.yourdomain.com
SMTP_PORT=587
SMTP_USER=noreply@yourdomain.com
SMTP_PASSWORD=secure_email_password

# Application
APP_ENV=production
APP_DEBUG=false
```

#### Deployment Commands

```bash
# 1. Build and deploy
make build-dist
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d

# 2. View logs
docker-compose -f docker-compose.prod.yml logs -f app

# 3. Update deployment
docker-compose -f docker-compose.prod.yml pull
docker-compose -f docker-compose.prod.yml up -d

# 4. Database backup
docker exec turboscript_postgres_1 pg_dump -U $DB_USER $DB_NAME > backup.sql
```

### 2. Direct Server Deployment

#### Server Prerequisites

```bash
# Ubuntu 20.04+ server setup
sudo apt update && sudo apt upgrade -y

# Install required packages
sudo apt install -y postgresql postgresql-contrib nginx certbot python3-certbot-nginx

# Install Go (if building on server)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### Database Setup

```bash
# Create production database
sudo -u postgres psql
CREATE DATABASE turboscript_prod;
CREATE USER turboscript_user WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE turboscript_prod TO turboscript_user;
\q

# Import schema
sudo -u postgres psql -d turboscript_prod -f /path/to/init.sql
```

#### Application Deployment

```bash
# 1. Deploy application files
sudo mkdir -p /opt/turboscript
sudo chown $USER:$USER /opt/turboscript

# Copy distribution files
cp -r dist/* /opt/turboscript/

# Make binary executable
chmod +x /opt/turboscript/turboscript-linux

# 2. Create systemd service
sudo tee /etc/systemd/system/turboscript.service > /dev/null <<EOF
[Unit]
Description=TurboScript Application
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/turboscript
ExecStart=/opt/turboscript/turboscript-linux
Restart=always
RestartSec=5
Environment=JWT_ACCESS_SECRET=your_secret
Environment=JWT_REFRESH_SECRET=your_secret
Environment=DB_HOST=localhost
Environment=DB_PORT=5432
Environment=DB_NAME=turboscript_prod
Environment=DB_USER=turboscript_user
Environment=DB_PASSWORD=secure_password

[Install]
WantedBy=multi-user.target
EOF

# 3. Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable turboscript
sudo systemctl start turboscript

# 4. Check status
sudo systemctl status turboscript
```

#### Nginx Configuration

```nginx
# /etc/nginx/sites-available/turboscript
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com www.yourdomain.com;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Rate Limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req zone=api burst=20 nodelay;

    # Proxy to TurboScript application
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Static file serving (if needed)
    location /static/ {
        alias /opt/turboscript/static/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # Health check endpoint
    location /health {
        proxy_pass http://127.0.0.1:8080/health;
        access_log off;
    }
}
```

#### SSL Certificate Setup

```bash
# Install SSL certificate with Let's Encrypt
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# Test automatic renewal
sudo certbot renew --dry-run

# Set up automatic renewal
echo "0 12 * * * /usr/bin/certbot renew --quiet" | sudo crontab -
```

### 3. Cloud Platform Deployment

#### AWS Deployment with ECS

```yaml
# ecs-task-definition.json
{
  "family": "turboscript-app",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::account:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "turboscript",
      "image": "your-ecr-repo/turboscript:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "DB_HOST",
          "value": "your-rds-endpoint.rds.amazonaws.com"
        }
      ],
      "secrets": [
        {
          "name": "JWT_ACCESS_SECRET",
          "valueFrom": "arn:aws:ssm:region:account:parameter/turboscript/jwt-access-secret"
        },
        {
          "name": "JWT_REFRESH_SECRET",
          "valueFrom": "arn:aws:ssm:region:account:parameter/turboscript/jwt-refresh-secret"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/turboscript",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

#### Google Cloud Run Deployment

```yaml
# cloudrun.yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: turboscript-app
  annotations:
    run.googleapis.com/ingress: all
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/maxScale: "10"
        run.googleapis.com/cpu-throttling: "false"
    spec:
      containerConcurrency: 100
      containers:
      - image: gcr.io/your-project/turboscript:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "your-cloud-sql-connection"
        - name: JWT_ACCESS_SECRET
          valueFrom:
            secretKeyRef:
              name: turboscript-secrets
              key: jwt-access-secret
        resources:
          limits:
            cpu: "1"
            memory: "512Mi"
```

## Production Configuration

### Optimized turboscript.yml

```yaml
# Production configuration
server:
  port: 8080
  timeout: "30s"
  max_body_size: "10MB"
  cors:
    enabled: true
    origins:
      - "https://yourdomain.com"
      - "https://www.yourdomain.com"
    methods: ["GET", "POST", "PUT", "DELETE"]
    headers: ["Content-Type", "Authorization"]

database:
  host: "${DB_HOST}"
  port: 5432
  name: "${DB_NAME}"
  user: "${DB_USER}"
  password: "${DB_PASSWORD}"
  ssl: true
  max_connections: 50
  connection_timeout: "10s"
  idle_timeout: "30m"
  max_lifetime: "2h"
  allowed_tables:
    - users
    - user_sessions
    - orders
    - order_items
    - products
    - audit_logs

security:
  jwt_access_secret: "${JWT_ACCESS_SECRET}"
  jwt_refresh_secret: "${JWT_REFRESH_SECRET}"
  bcrypt_cost: 12

logging:
  level: "warn"
  format: "json"

email:
  driver: "smtp"
  smtp_host: "${SMTP_HOST}"
  smtp_port: 587
  smtp_user: "${SMTP_USER}"
  smtp_password: "${SMTP_PASSWORD}"
  from_address: "noreply@yourdomain.com"
  from_name: "Your App Name"

jobs:
  max_workers: 20
  queue_size: 5000
  retry_attempts: 3
  retry_delay: "30s"
  data_retention:
    jobs_days: 30
    history_days: 30
    auto_cleanup: true

debug: false
```

## Monitoring and Maintenance

### Health Check Endpoint

```typescript
// app/routes/health.ts
export const handle = async (_event: Event): Promise<TurboScriptResponse> => {
    try {
        // Test database connection
        await turboQuery('SELECT 1');

        // Test email service (optional)
        // await testEmailConnection();

        return {
            code: 200,
            response: {
                status: "healthy",
                timestamp: new Date().toISOString(),
                version: process.env.APP_VERSION || "unknown",
                services: {
                    database: "connected",
                    email: "available"
                }
            }
        };
    } catch (error) {
        return {
            code: 503,
            response: {
                status: "unhealthy",
                timestamp: new Date().toISOString(),
                error: error instanceof Error ? error.message : "Health check failed"
            }
        };
    }
};
```

### Logging Configuration

```bash
# Log rotation with logrotate
sudo tee /etc/logrotate.d/turboscript > /dev/null <<EOF
/var/log/turboscript/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 www-data www-data
    postrotate
        systemctl reload turboscript
    endscript
}
EOF
```

### Backup Strategy

```bash
#!/bin/bash
# backup.sh - Database backup script

BACKUP_DIR="/opt/backups/turboscript"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="turboscript_backup_${DATE}.sql"

# Create backup directory
mkdir -p $BACKUP_DIR

# Perform database backup
docker exec turboscript_postgres_1 pg_dump \
    -U $DB_USER \
    -h localhost \
    $DB_NAME > $BACKUP_DIR/$BACKUP_FILE

# Compress backup
gzip $BACKUP_DIR/$BACKUP_FILE

# Remove backups older than 30 days
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete

# Upload to cloud storage (optional)
# aws s3 cp $BACKUP_DIR/${BACKUP_FILE}.gz s3://your-backup-bucket/
```

### Monitoring Setup

```bash
# Install monitoring tools
sudo apt install -y prometheus node-exporter

# Prometheus configuration for TurboScript
cat > /etc/prometheus/turboscript.yml <<EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'turboscript'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'node'
    static_configs:
      - targets: ['localhost:9100']
EOF
```

## Scaling and Performance

### Horizontal Scaling

```yaml
# docker-compose.scale.yml
version: '3.8'

services:
  app:
    build: .
    deploy:
      replicas: 3
    environment:
      - JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET}
      - JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}
    depends_on:
      - postgres
      - redis

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx-lb.conf:/etc/nginx/nginx.conf
    depends_on:
      - app

  redis:
    image: redis:alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

volumes:
  redis_data:
```

### Load Balancer Configuration

```nginx
# nginx-lb.conf
upstream turboscript_backend {
    least_conn;
    server app_1:8080 weight=1 max_fails=3 fail_timeout=30s;
    server app_2:8080 weight=1 max_fails=3 fail_timeout=30s;
    server app_3:8080 weight=1 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    location / {
        proxy_pass http://turboscript_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Deployment Checklist

### Pre-deployment

- [ ] Build and test distribution package locally
- [ ] Verify all environment variables are set
- [ ] Test database migrations
- [ ] Review security configuration
- [ ] Backup existing production data
- [ ] Prepare rollback plan

### Deployment

- [ ] Deploy during low-traffic period
- [ ] Monitor application logs during deployment
- [ ] Verify health check endpoints
- [ ] Test critical user flows
- [ ] Monitor performance metrics
- [ ] Verify SSL certificate validity

### Post-deployment

- [ ] Monitor error rates and response times
- [ ] Verify background jobs are processing
- [ ] Test email functionality
- [ ] Check database connection pools
- [ ] Verify log collection and rotation
- [ ] Update monitoring dashboards

---

## Navigation

**Previous:** [← Security Guide](guides/security.md)
**Next:** [Troubleshooting →](guides/troubleshooting.md)

## Related Topics

- [Performance Guide](guides/performance.md)
- [Security Guidelines](guides/security.md)
- [Configuration Guide](guides/configuration.md)
- [Development Workflow](guides/development.md)
