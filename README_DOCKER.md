# CodexSecure API - Docker Deployment

This directory contains the CodexSecure API service with Docker containerization support.

## Overview

CodexSecure is a security and encryption microservice built in Go. It now runs as a Docker container on the `codex-net` network, making it accessible to other Codex services.

## Quick Start

### Build and Run

```bash
# Build the Docker image
docker-compose build

# Start the container
docker-compose up -d

# Check logs
docker logs codexsecure -f

# Check health
curl http://localhost:8084/health
```

### Stop and Remove

```bash
# Stop the container
docker-compose down

# Remove the container and image
docker-compose down --rmi all
```

## Architecture

### Port Configuration
- **Host Port**: 8084
- **Container Port**: 8084
- **Health Endpoint**: `/health`
- **Metrics Endpoint**: `/metrics`

### Network
- **Network**: `codex-net` (external)
- **Container Name**: `codexsecure`
- **Accessible via**: `http://codexsecure:8084` (from other containers)

### Database
- **Type**: PostgreSQL
- **Host**: 172.104.24.82 (configurable via `.env`)
- **Database**: `codex_main`
- **User**: `codex_admin`

## Environment Variables

All configuration is managed through the `.env` file. Key variables:

```bash
# Server
SERVER_PORT=8084
SERVER_HOST=0.0.0.0
ENV=development

# Database
DB_HOST=172.104.24.82
DB_PORT=5432
DB_NAME=codex_main
DB_USER=codex_admin
DB_PASSWORD=your_password_here

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

See `.env.example` for all available options.

## Health Checks

The container includes Docker health checks:
- **Interval**: 30 seconds
- **Timeout**: 3 seconds
- **Retries**: 3
- **Start Period**: 10 seconds

Check health status:
```bash
docker inspect codexsecure --format '{{.State.Health.Status}}'
```

## Service Monitoring

CodexSecure integrates with the CodexAdmin dashboard for monitoring:

```bash
# Check service status via API
curl http://localhost:8082/api/services/codexsecure
```

The dashboard displays:
- Running status (green = healthy, yellow = degraded, red = down)
- Health check results
- Container uptime
- Dependencies

## Development

### Build Locally

```bash
# Build without Docker
go build -o bin/codexsecure ./cmd/server

# Run locally
./bin/codexsecure
```

### Hot Reload (Development)

For development with hot reload, you can mount the source code:

```yaml
volumes:
  - .:/app
  - /app/bin  # Exclude bin directory
```

### Logs

```bash
# Follow container logs
docker logs -f codexsecure

# Last 100 lines
docker logs --tail 100 codexsecure

# Since timestamp
docker logs --since 10m codexsecure
```

## Troubleshooting

### Container won't start
```bash
# Check logs
docker logs codexsecure

# Inspect container
docker inspect codexsecure

# Check if port is in use
ss -tlnp | grep :8084
```

### Health check failing
```bash
# Test health endpoint directly
docker exec codexsecure wget --spider http://localhost:8084/health

# Check from another container
docker exec codex-admin curl http://codexsecure:8084/health
```

### Database connection issues
```bash
# Test database connectivity
docker exec codexsecure ping -c 3 172.104.24.82

# Check database credentials in .env
cat .env | grep DB_
```

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check (returns service status) |
| `/ready` | GET | Readiness check (database connection) |
| `/metrics` | GET | Service metrics and endpoint tracking |
| `/api/v1/status` | GET | Service status information |

## Service Tracking

CodexSecure includes endpoint hit tracking:

```bash
# Get metrics for last hour
curl "http://localhost:8084/metrics?hours=1"

# Get metrics for last 24 hours
curl "http://localhost:8084/metrics?hours=24"
```

Tracked data includes:
- Total hits per endpoint
- Average response time
- Last hit timestamp
- Stored in `service_endpoint_hits` table

## Integration with Other Services

CodexSecure is accessible to other services on the `codex-net` network:

```bash
# From CodexAdmin
http://codexsecure:8084

# From Web3Connected
http://codexsecure:8084

# From CodexAuth
http://codexsecure:8084
```

## Production Deployment

For production:

1. Update `.env` with production values:
   ```bash
   ENV=production
   SERVER_HOST=0.0.0.0
   LOG_LEVEL=info
   ```

2. Set `GIN_MODE=release`:
   ```bash
   export GIN_MODE=release
   ```

3. Use proper secrets management (not `.env` files)

4. Enable SSL/TLS for database connections:
   ```bash
   DB_SSL_MODE=require
   ```

## Version

- **Version**: 1.0.0
- **Go Version**: 1.23
- **Framework**: Gin
- **Database Driver**: pgx/v5
