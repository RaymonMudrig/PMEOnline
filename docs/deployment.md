# PME Online - Deployment Guide

This guide covers deploying PME Online services to production environments where each service runs on separate servers/VMs.

## Prerequisites

- Docker installed on each server
- Kafka cluster accessible from all servers
- PostgreSQL database accessible from DB Exporter server
- Network connectivity between services and infrastructure

## Architecture Overview

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Server 1    │     │  Server 2    │     │  Server 3    │     │  Server 4    │
│              │     │              │     │              │     │              │
│  eClear API  │     │     OMS      │     │  APME API    │     │ DB Exporter  │
│  :8081       │     │              │     │  :8080       │     │              │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                    │                    │
       └────────────────────┴────────────────────┴────────────────────┘
                                     │
                          ┌──────────┴──────────┐
                          │                     │
                     ┌────▼─────┐         ┌─────▼──────┐
                     │  Kafka   │         │ PostgreSQL │
                     │  Cluster │         │  Database  │
                     └──────────┘         └────────────┘
```

## Building Docker Images

### Build All Images

From the project root:

```bash
# Build eClear API
docker build -t pme-eclearapi:latest -f cmd/eclearapi/Dockerfile .

# Build OMS
docker build -t pme-oms:latest -f cmd/pmeoms/Dockerfile .

# Build APME API
docker build -t pme-api:latest -f cmd/pmeapi/Dockerfile .

# Build DB Exporter
docker build -t pme-dbexporter:latest -f cmd/dbexporter/Dockerfile .
```

### Tag for Registry

If using a private Docker registry:

```bash
# Tag images
docker tag pme-eclearapi:latest registry.example.com/pme-eclearapi:v1.0.0
docker tag pme-oms:latest registry.example.com/pme-oms:v1.0.0
docker tag pme-api:latest registry.example.com/pme-api:v1.0.0
docker tag pme-dbexporter:latest registry.example.com/pme-dbexporter:v1.0.0

# Push to registry
docker push registry.example.com/pme-eclearapi:v1.0.0
docker push registry.example.com/pme-oms:v1.0.0
docker push registry.example.com/pme-api:v1.0.0
docker push registry.example.com/pme-dbexporter:v1.0.0
```

## Deployment

### Environment Variables

Each service requires the following environment variables:

#### Common Variables (All Services)
```bash
KAFKA_URL=kafka-server:9092          # Kafka broker address
KAFKA_TOPIC=pme-ledger               # Kafka topic name
```

#### eClear API (Server 1)
```bash
API_PORT=8081                        # HTTP port
ECLEAR_BASE_URL=http://eclear:9000   # eClear system URL
```

#### OMS (Server 2)
```bash
# No additional variables
```

#### APME API (Server 3)
```bash
API_PORT=8080                        # HTTP port
```

#### DB Exporter (Server 4)
```bash
DB_HOST=postgres-server              # PostgreSQL host
DB_PORT=5432                         # PostgreSQL port
DB_USER=pmeuser                      # Database username
DB_PASSWORD=pmepass                  # Database password
DB_NAME=pmedb                        # Database name
DB_SSLMODE=require                   # SSL mode (disable/require/verify-full)
```

### Deployment Commands

#### Server 1: eClear API Service

```bash
docker run -d \
  --name pme-eclearapi \
  --restart unless-stopped \
  -p 8081:8081 \
  -e KAFKA_URL=10.0.1.100:9092 \
  -e KAFKA_TOPIC=pme-ledger \
  -e API_PORT=8081 \
  -e ECLEAR_BASE_URL=http://eclear.example.com:9000 \
  pme-eclearapi:latest
```

**Health Check:**
```bash
curl http://localhost:8081/health
```

#### Server 2: OMS Service

```bash
docker run -d \
  --name pme-oms \
  --restart unless-stopped \
  -e KAFKA_URL=10.0.1.100:9092 \
  -e KAFKA_TOPIC=pme-ledger \
  pme-oms:latest
```

**Logs Monitoring:**
```bash
docker logs -f pme-oms
```

#### Server 3: APME API Service

```bash
docker run -d \
  --name pme-api \
  --restart unless-stopped \
  -p 8080:8080 \
  -e KAFKA_URL=10.0.1.100:9092 \
  -e KAFKA_TOPIC=pme-ledger \
  -e API_PORT=8080 \
  pme-api:latest
```

**Health Check:**
```bash
curl http://localhost:8080/health
```

**Access Dashboard:**
```
http://server3-ip:8080
```

#### Server 4: DB Exporter Service

```bash
docker run -d \
  --name pme-dbexporter \
  --restart unless-stopped \
  -e KAFKA_URL=10.0.1.100:9092 \
  -e KAFKA_TOPIC=pme-ledger \
  -e DB_HOST=10.0.1.200 \
  -e DB_PORT=5432 \
  -e DB_USER=pmeuser \
  -e DB_PASSWORD='your-secure-password' \
  -e DB_NAME=pmedb \
  -e DB_SSLMODE=require \
  pme-dbexporter:latest
```

**Verify Database Connection:**
```bash
docker logs pme-dbexporter | grep "Database connected"
```

## Using Environment Files

For better security and management, use `.env` files:

### Example: eclearapi.env
```bash
KAFKA_URL=10.0.1.100:9092
KAFKA_TOPIC=pme-ledger
API_PORT=8081
ECLEAR_BASE_URL=http://eclear.example.com:9000
```

### Run with env file:
```bash
docker run -d \
  --name pme-eclearapi \
  --restart unless-stopped \
  -p 8081:8081 \
  --env-file eclearapi.env \
  pme-eclearapi:latest
```

## Docker Compose (Alternative - All on One Server)

If deploying all services on a single server for testing:

```bash
# Uncomment services in docker-compose.yml
docker-compose up -d
```

## Monitoring and Logging

### View Logs

```bash
# eClear API
docker logs -f pme-eclearapi

# OMS
docker logs -f pme-oms

# APME API
docker logs -f pme-api

# DB Exporter
docker logs -f pme-dbexporter
```

### Check Container Status

```bash
docker ps -a | grep pme-
```

### Resource Usage

```bash
docker stats pme-eclearapi pme-oms pme-api pme-dbexporter
```

## Stopping and Updating Services

### Stop a Service

```bash
docker stop pme-eclearapi
docker rm pme-eclearapi
```

### Update a Service

```bash
# Pull new image
docker pull registry.example.com/pme-eclearapi:v1.1.0

# Stop old container
docker stop pme-eclearapi
docker rm pme-eclearapi

# Start new container
docker run -d \
  --name pme-eclearapi \
  --restart unless-stopped \
  -p 8081:8081 \
  --env-file eclearapi.env \
  registry.example.com/pme-eclearapi:v1.1.0
```

### Rolling Update Script

```bash
#!/bin/bash
# update-service.sh

SERVICE=$1
VERSION=$2
IMAGE="registry.example.com/pme-${SERVICE}:${VERSION}"

echo "Updating ${SERVICE} to ${VERSION}..."

# Pull new image
docker pull $IMAGE

# Stop and remove old container
docker stop pme-${SERVICE}
docker rm pme-${SERVICE}

# Start new container
docker run -d \
  --name pme-${SERVICE} \
  --restart unless-stopped \
  --env-file ${SERVICE}.env \
  $IMAGE

echo "${SERVICE} updated successfully"
```

Usage:
```bash
./update-service.sh eclearapi v1.1.0
```

## Backup and Recovery

### Backup PostgreSQL

```bash
docker exec pme-postgres pg_dump -U pmeuser pmedb > pmedb-backup-$(date +%Y%m%d).sql
```

### Restore PostgreSQL

```bash
docker exec -i pme-postgres psql -U pmeuser pmedb < pmedb-backup-20251122.sql
```

### Backup Kafka Topic

```bash
# Using Kafka console consumer
docker exec pme-kafka kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic pme-ledger \
  --from-beginning \
  --max-messages 1000000 > kafka-backup.json
```

## Security Considerations

### 1. Network Security

- Use firewall rules to restrict access
- Only expose necessary ports (8080, 8081)
- Use VPN or private network for inter-service communication

### 2. Secrets Management

- Never commit `.env` files with passwords
- Use Docker secrets or external secret managers
- Rotate credentials regularly

### 3. SSL/TLS

- Enable PostgreSQL SSL: `DB_SSLMODE=require`
- Use HTTPS for API endpoints (reverse proxy)
- Use SSL for Kafka connections

### 4. Container Security

- Run containers as non-root user (already configured)
- Keep images updated
- Scan images for vulnerabilities

## Troubleshooting

### Service Won't Start

```bash
# Check logs
docker logs pme-eclearapi

# Check environment variables
docker inspect pme-eclearapi | grep -A 20 Env

# Verify network connectivity
docker exec pme-eclearapi ping kafka-server
```

### Cannot Connect to Kafka

```bash
# Test Kafka connectivity
docker run --rm -it apache/kafka:3.8.1 \
  kafka-broker-api-versions.sh \
  --bootstrap-server 10.0.1.100:9092
```

### Database Connection Issues

```bash
# Test PostgreSQL connectivity
docker run --rm -it postgres:16-alpine \
  psql -h 10.0.1.200 -U pmeuser -d pmedb
```

### High Memory Usage

```bash
# Check resource usage
docker stats

# Restart service if needed
docker restart pme-eclearapi
```

## Performance Tuning

### Container Resources

```bash
# Limit CPU and memory
docker run -d \
  --name pme-oms \
  --restart unless-stopped \
  --cpus="2.0" \
  --memory="4g" \
  --memory-swap="4g" \
  -e KAFKA_URL=10.0.1.100:9092 \
  pme-oms:latest
```

### Kafka Configuration

- Increase partitions for higher throughput
- Adjust consumer group settings
- Configure appropriate retention policies

### PostgreSQL Tuning

- Adjust `shared_buffers` and `work_mem`
- Configure connection pooling
- Create appropriate indexes

## Production Checklist

- [ ] All services built and pushed to registry
- [ ] Environment files configured for each server
- [ ] Kafka cluster accessible from all servers
- [ ] PostgreSQL database accessible from DB Exporter
- [ ] Firewall rules configured
- [ ] SSL/TLS enabled for database and APIs
- [ ] Monitoring and alerting configured
- [ ] Backup strategy implemented
- [ ] Log aggregation configured
- [ ] Health checks passing
- [ ] Load testing completed
- [ ] Disaster recovery plan documented

## Support

For issues or questions:
- Check service logs: `docker logs -f <container-name>`
- Review service README files in `cmd/*/readme.md`
- Check system design: `docs/design.md`
