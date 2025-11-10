# Deployment Guide

## Overview

Hướng dẫn deploy Meeting Assistant từ development environment đến production.

## Development Environment

### Prerequisites

```bash
# System Requirements
- CPU: 4+ cores
- RAM: 8GB minimum, 16GB recommended
- Storage: 50GB SSD
- OS: macOS, Linux, or Windows with WSL2

# Software Requirements
- Go 1.21+
- Node.js 18+ & npm 9+
- Docker 24+ & Docker Compose 2.20+
- PostgreSQL 15+ (via Docker or local)
- Redis 7+ (via Docker or local)
- Git 2.40+

# API Keys (required)
- LiveKit Cloud account or self-hosted server
- OpenAI API key (GPT-4 + Whisper access)
- Google OAuth2 credentials (Google Cloud Console)
```

### Local Setup

#### 1. Clone Repository

```bash
git clone https://github.com/yourorg/meeting-assistant.git
cd meeting-assistant
```

#### 2. Environment Configuration

**Backend (.env)**
```bash
cd backend
cp .env.example .env
```

Edit `backend/.env`:
```env
# Server
PORT=8080
ENV=development
DEBUG=true

# Database
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/meeting_assistant?sslmode=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# Redis
REDIS_URL=redis://localhost:6379/0
REDIS_PASSWORD=

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=7d

# OAuth2 (Google only)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback

# LiveKit
LIVEKIT_API_KEY=your-livekit-api-key
LIVEKIT_API_SECRET=your-livekit-api-secret
LIVEKIT_URL=wss://your-livekit-server.com
LIVEKIT_WEBHOOK_SECRET=your-webhook-secret

# OpenAI
OPENAI_API_KEY=sk-your-openai-api-key
OPENAI_ORG_ID=org-your-org-id (optional)

# Storage (MinIO - Self-hosted Object Storage)
MINIO_ENDPOINT=http://localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=meeting-recordings
MINIO_USE_SSL=false

# Email (Optional - for notifications)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
EMAIL_FROM=noreply@meetingassistant.com

# ClickUp (Optional)
CLICKUP_CLIENT_ID=your-clickup-client-id
CLICKUP_CLIENT_SECRET=your-clickup-client-secret

# CORS
CORS_ORIGINS=http://localhost:3000,http://localhost:3001

# Rate Limiting
RATE_LIMIT_REQUESTS_PER_MINUTE=100

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json
```

**Frontend (.env)**
```bash
cd ../frontend
cp .env.example .env
```

Edit `frontend/.env`:
```env
REACT_APP_API_URL=http://localhost:8080/api/v1
REACT_APP_WS_URL=ws://localhost:8080/ws
REACT_APP_LIVEKIT_URL=wss://your-livekit-server.com
REACT_APP_ENV=development

# OAuth (Google only)
REACT_APP_GOOGLE_CLIENT_ID=your-google-client-id

# Feature Flags
REACT_APP_ENABLE_RECORDING=true
REACT_APP_ENABLE_TRANSCRIPTION=true
REACT_APP_ENABLE_CLICKUP=false

# Analytics (Optional)
REACT_APP_GA_TRACKING_ID=
REACT_APP_SENTRY_DSN=
```

#### 3. Start Infrastructure with Docker

```bash
# From project root
docker-compose -f docker-compose.dev.yml up -d

# Check services are running
docker-compose ps
```

**docker-compose.dev.yml**
```yaml
version: '3.9'

services:
  postgres:
    image: postgres:15-alpine
    container_name: meeting-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: meeting_assistant
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: meeting-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    container_name: meeting-minio
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    command: server /data --console-address ":9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  # Optional: LiveKit (if not using cloud)
  livekit:
    image: livekit/livekit-server:latest
    container_name: meeting-livekit
    ports:
      - "7880:7880"
      - "7881:7881"
      - "7882:7882/udp"
    volumes:
      - ./livekit.yaml:/etc/livekit.yaml
    command: --config /etc/livekit.yaml

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

#### 4. Database Migration

```bash
cd backend

# Install migration tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
make migrate-up

# Or manually
migrate -path ./migrations -database "${DATABASE_URL}" up
```

#### 5. Start Backend

```bash
cd backend

# Download dependencies
go mod download
go mod tidy

# Run
go run cmd/server/main.go

# Or with hot reload (install air first)
go install github.com/cosmtrek/air@latest
air
```

Backend will start on `http://localhost:8080`

API Documentation: `http://localhost:8080/swagger`

#### 6. Start Frontend

```bash
cd frontend

# Install dependencies
npm install

# Start dev server
npm start
```

Frontend will start on `http://localhost:3000`

#### 7. Create MinIO Bucket

```bash
# Access MinIO Console: http://localhost:9001
# Login: minioadmin / minioadmin

# Or use CLI
docker exec -it meeting-minio mc alias set myminio http://localhost:9000 minioadmin minioadmin
docker exec -it meeting-minio mc mb myminio/meeting-recordings
docker exec -it meeting-minio mc anonymous set download myminio/meeting-recordings
```

### Development Workflow

```bash
# Backend hot reload with Air
cd backend && air

# Frontend hot reload (default)
cd frontend && npm start

# Run tests
cd backend && go test ./...
cd frontend && npm test

# Database operations
make migrate-create name=add_users_table
make migrate-up
make migrate-down
make migrate-force version=1

# Docker logs
docker-compose logs -f postgres
docker-compose logs -f redis

# Reset database
docker-compose down -v
docker-compose up -d
make migrate-up
```

## Production Deployment

### Server Requirements

**Minimum (Single Server):**
- CPU: 4 cores
- RAM: 8GB
- Storage: 100GB SSD
- Network: 100Mbps
- OS: Ubuntu 22.04 LTS

**Recommended (For 10+ concurrent rooms):**
- CPU: 8 cores
- RAM: 16GB
- Storage: 500GB SSD
- Network: 1Gbps
- OS: Ubuntu 22.04 LTS

### Deployment with Docker Compose

Best for: Small teams, single server, MVP (recommended for this project)

```bash
# 1. Server Setup
ssh user@your-server.com

# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo apt install docker-compose-plugin

# 2. Clone repository
git clone https://github.com/yourorg/meeting-assistant.git
cd meeting-assistant

# 3. Configure environment
cp .env.example .env
nano .env  # Edit with production values

# 4. Start services
docker-compose -f docker-compose.prod.yml up -d

# 5. Check status
docker-compose ps
docker-compose logs -f
```

**docker-compose.prod.yml**
```yaml
version: '3.9'

services:
  postgres:
    image: postgres:15-alpine
    restart: always
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backups:/backups
    networks:
      - backend

  redis:
    image: redis:7-alpine
    restart: always
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    networks:
      - backend

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile.prod
    restart: always
    depends_on:
      - postgres
      - redis
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
      - JWT_SECRET=${JWT_SECRET}
      - LIVEKIT_API_KEY=${LIVEKIT_API_KEY}
      - LIVEKIT_API_SECRET=${LIVEKIT_API_SECRET}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    volumes:
      - ./logs:/app/logs
    networks:
      - backend
      - frontend

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile.prod
      args:
        - REACT_APP_API_URL=${API_URL}
        - REACT_APP_LIVEKIT_URL=${LIVEKIT_URL}
    restart: always
    depends_on:
      - backend
    networks:
      - frontend

  nginx:
    image: nginx:alpine
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf
      - ./nginx/ssl:/etc/nginx/ssl
      - ./frontend/build:/usr/share/nginx/html
    depends_on:
      - backend
      - frontend
    networks:
      - frontend

  livekit:
    image: livekit/livekit-server:latest
    restart: always
    ports:
      - "7880:7880"
      - "7881:7881"
      - "7882:7882/udp"
    volumes:
      - ./livekit-prod.yaml:/etc/livekit.yaml
    command: --config /etc/livekit.yaml
    networks:
      - backend

networks:
  backend:
    driver: bridge
  frontend:
    driver: bridge

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

**Note:** Production cũng cần thêm MinIO service trong docker-compose.prod.yml:

```yaml
  minio:
    image: minio/minio:latest
    container_name: meeting-minio-prod
    restart: always
    environment:
      MINIO_ROOT_USER: ${MINIO_ACCESS_KEY}
      MINIO_ROOT_PASSWORD: ${MINIO_SECRET_KEY}
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    command: server /data --console-address ":9001"
    networks:
      - backend
```

### SSL/TLS Setup (Let's Encrypt)

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d your-domain.com -d www.your-domain.com

# Auto-renewal
sudo certbot renew --dry-run

# Add to crontab for auto-renewal
0 12 * * * /usr/bin/certbot renew --quiet
```

**nginx.conf with SSL**
```nginx
upstream backend {
    server backend:8080;
}

upstream livekit {
    server livekit:7880;
}

server {
    listen 80;
    server_name your-domain.com www.your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com www.your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Frontend
    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }

    # Backend API
    location /api/ {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # WebSocket
    location /ws {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400;
    }

    # LiveKit
    location /livekit/ {
        proxy_pass http://livekit/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;

    # Gzip
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/x-javascript application/xml+rss application/json;
}
```

## Monitoring & Logging

### Prometheus + Grafana

```bash
# Deploy monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d
```

**docker-compose.monitoring.yml**
```yaml
version: '3.9'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
    depends_on:
      - prometheus

  node_exporter:
    image: prom/node-exporter:latest
    container_name: node_exporter
    ports:
      - "9100:9100"

volumes:
  prometheus_data:
  grafana_data:
```

### Application Logs

```bash
# Backend logs
docker-compose logs -f backend

# Aggregate logs
docker-compose logs -f > logs/app.log

# Use log aggregation tools
# - ELK Stack (Elasticsearch, Logstash, Kibana)
# - Loki + Grafana
# - CloudWatch (AWS)
```

## Backup & Recovery

### Database Backup

```bash
# Automated backup script
#!/bin/bash
# backup.sh

BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
FILENAME="meeting_assistant_${DATE}.sql.gz"

# PostgreSQL backup
docker exec meeting-postgres pg_dump -U postgres meeting_assistant | gzip > ${BACKUP_DIR}/${FILENAME}

# Keep last 30 days
find ${BACKUP_DIR} -name "meeting_assistant_*.sql.gz" -mtime +30 -delete

# Upload to MinIO (optional, for off-site backup)
# Install mc (MinIO Client) first: https://min.io/docs/minio/linux/reference/minio-mc.html
mc alias set minio-backup https://backup-server.com BACKUP_ACCESS_KEY BACKUP_SECRET_KEY
mc cp ${BACKUP_DIR}/${FILENAME} minio-backup/database-backups/
```

Add to crontab:
```bash
0 2 * * * /path/to/backup.sh
```

### Recovery

```bash
# Restore from backup
gunzip -c /backups/meeting_assistant_20251103.sql.gz | \
docker exec -i meeting-postgres psql -U postgres meeting_assistant
```

## Performance Optimization

### Database Optimization

```sql
-- Add indexes
CREATE INDEX CONCURRENTLY idx_rooms_status ON rooms(status) WHERE status = 'active';
CREATE INDEX CONCURRENTLY idx_participants_room_user ON participants(room_id, user_id);

-- Analyze tables
ANALYZE rooms;
ANALYZE participants;
ANALYZE recordings;

-- Vacuum
VACUUM ANALYZE;
```

### Redis Caching Strategy

```go
// Cache frequently accessed data
// - User sessions (TTL: 7 days)
// - Room state (TTL: 24 hours)
// - API responses (TTL: 5 minutes)

// Cache invalidation on updates
// - Room updates → Clear room cache
// - User updates → Clear user cache
```

### CDN Configuration (Optional)

Use CloudFlare (Free plan available) for:
- Static assets (JS, CSS, images)
- SSL/TLS termination
- DDoS protection
- Caching

For recorded files, use MinIO with Nginx reverse proxy:
```nginx
# Add to nginx.conf
location /recordings/ {
    proxy_pass http://minio:9000/meeting-recordings/;
    proxy_set_header Host $host;
    proxy_buffering off;
    client_max_body_size 500M;
}
```

## Troubleshooting

### Common Issues

**1. Database connection failed**
```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Check connection
psql -h localhost -U postgres -d meeting_assistant

# Reset password
docker-compose exec postgres psql -U postgres -c "ALTER USER postgres PASSWORD 'newpassword';"
```

**2. LiveKit connection issues**
```bash
# Check LiveKit server
curl https://your-livekit-server.com/health

# Verify API credentials
lk room list --url wss://your-server.com --api-key KEY --api-secret SECRET

# Check firewall rules (ports 7880, 7881, 7882)
sudo ufw allow 7880/tcp
sudo ufw allow 7881/tcp
sudo ufw allow 7882/udp
```

**3. OpenAI API errors**
```bash
# Test API key
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"

# Check quota
curl https://api.openai.com/v1/usage \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

**4. High memory usage**
```bash
# Check Docker resources
docker stats

# Limit container memory
docker-compose.yml:
  backend:
    mem_limit: 2g
    memswap_limit: 2g
```

**5. MinIO connection issues**
```bash
# Check MinIO is running
docker-compose ps minio

# Test connection
curl http://localhost:9000/minio/health/live

# Access MinIO Console
open http://localhost:9001

# Check bucket exists
docker exec -it meeting-minio mc ls minio/meeting-recordings
```

## Scaling Strategy (Future)

**Note:** Cho MVP với 5-10 users, không cần scaling ngay. Khi cần scale lên:

### Horizontal Scaling

```bash
# Add more backend instances
docker-compose up -d --scale backend=3

# Add load balancer
# nginx.conf:
upstream backend {
    least_conn;
    server backend1:8080;
    server backend2:8080;
    server backend3:8080;
}
```

### Storage Scaling

```bash
# MinIO distributed mode (4+ servers)
# Cung cấp high availability và auto-healing
# Docs: https://min.io/docs/minio/linux/operations/install-deploy-manage/deploy-minio-multi-node-multi-drive.html
```

### Database Scaling

```bash
# PostgreSQL read replicas
# - Master: Write operations
# - Replicas: Read operations

# Connection pooling
# - PgBouncer for PostgreSQL
# - Redis Cluster for caching
```

## Security Checklist

- [ ] Change all default passwords
- [ ] Use strong JWT secret (32+ characters)
- [ ] Enable HTTPS/TLS with valid certificate
- [ ] Configure firewall (UFW/iptables)
- [ ] Enable rate limiting
- [ ] Set up fail2ban for SSH
- [ ] Regular security updates
- [ ] Database encryption at rest
- [ ] Secure API keys in secrets manager
- [ ] Enable audit logging
- [ ] Configure CORS properly
- [ ] Implement CSP headers
- [ ] Regular backups
- [ ] Disaster recovery plan

## Maintenance

### Regular Tasks

**Daily:**
- Monitor error logs
- Check disk space
- Verify backups

**Weekly:**
- Review performance metrics
- Update dependencies
- Clean old recordings (30+ days)

**Monthly:**
- Security patches
- Database optimization
- Review access logs
- Test disaster recovery

---

**Last Updated:** November 3, 2025
