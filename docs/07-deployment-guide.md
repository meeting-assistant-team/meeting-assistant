# Deployment Guide

## Overview

High-level guidance for deploying Meeting Assistant to production.

## Deployment Architecture

**Production setup:**
- Nginx or similar reverse proxy (SSL/TLS termination)
- Docker containers for all services
- PostgreSQL database (managed or self-hosted)
- Redis cache (managed or self-hosted)
- External services: LiveKit, AssemblyAI, Groq (API-based)

## Environment Configuration

Production requires:
- `.env` file with all configuration
- Database connection string
- Redis connection details
- OAuth2 credentials
- API keys (AssemblyAI, Groq)
- LiveKit server/credentials
- JWT secret key
- CORS allowed origins

## Database Setup

1. Create PostgreSQL database
2. Run migrations: `go run scripts/migrate.go`
3. Verify schema created successfully
4. Set up backups

## Service Deployment

### Option 1: Docker Compose
```
docker-compose -f docker-compose.prod.yml up -d
```

Includes:
- Backend (Go API)
- PostgreSQL database
- Redis cache
- Nginx reverse proxy

### Option 2: Kubernetes
Use provided Helm charts for:
- Horizontal scaling
- Auto-recovery
- Load balancing
- Rolling updates

## SSL/TLS Configuration

- Use Let's Encrypt certificates
- Auto-renewal via certbot
- Configure Nginx for HTTPS
- Redirect HTTP → HTTPS

## Monitoring & Logging

- Structured logging to stdout (Docker-compatible)
- Log aggregation (ELK, CloudWatch, etc.)
- Health check endpoints
- Performance monitoring
- Error tracking

## Scaling Considerations

- Stateless backend allows horizontal scaling
- Database connections should be pooled
- Redis for distributed caching
- CDN for static assets
- Load balancer in front of API servers

## Backup Strategy

- Daily PostgreSQL backups
- Retention: 30 days minimum
- Test restoration regularly
- Store backups off-site

## Pre-deployment Checklist

- [ ] All environment variables configured
- [ ] Database migrated successfully
- [ ] SSL certificate installed
- [ ] OAuth credentials verified
- [ ] External API keys tested
- [ ] Backups configured
- [ ] Monitoring set up
- [ ] Log aggregation configured
- [ ] Firewall rules configured
- [ ] Security headers enabled
