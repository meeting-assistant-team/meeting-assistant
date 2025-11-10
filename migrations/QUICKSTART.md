# Database Quick Start Guide

## 🚀 Setup from Scratch

### 1. Install Prerequisites

```bash
# Install PostgreSQL
brew install postgresql@15

# Install golang-migrate
brew install golang-migrate

# Or using Go
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 2. Create Database

```bash
# Start PostgreSQL
brew services start postgresql@15

# Create database
createdb meeting_assistant

# Or using psql
psql postgres
CREATE DATABASE meeting_assistant;
\q
```

### 3. Set Environment Variable

```bash
# Add to your .env file
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/meeting_assistant?sslmode=disable"

# Or for production (with SSL)
export DATABASE_URL="postgresql://user:password@host:5432/meeting_assistant?sslmode=require"
```

### 4. Run Migrations

```bash
# Using Makefile (recommended)
make migrate-up

# Or directly with golang-migrate
migrate -path migrations -database "$DATABASE_URL" up

# Check version
make migrate-version
```

### 5. Seed Test Data (Optional)

```bash
# Run seed file
psql -U postgres -d meeting_assistant -f migrations/seed.sql

# Or using Make
make db-seed
```

## 📋 Common Tasks

### Create New Migration

```bash
# Using Makefile
make migrate-create NAME=add_feature_x

# Or directly
migrate create -ext sql -dir migrations -seq add_feature_x
```

### Rollback Migration

```bash
# Rollback last migration
make migrate-down

# Rollback to specific version
migrate -path migrations -database "$DATABASE_URL" goto 3
```

### Reset Database (Development)

```bash
# Complete reset
make db-reset

# Or manually
make migrate-down  # Repeat until version 0
make migrate-up
make db-seed
```

### Check Database Status

```bash
# Current migration version
make migrate-version

# Database statistics
psql -U postgres -d meeting_assistant -c "SELECT * FROM get_database_statistics();"

# Storage usage
psql -U postgres -d meeting_assistant -c "SELECT * FROM get_storage_usage();"

# Active meetings
psql -U postgres -d meeting_assistant -c "SELECT * FROM active_meetings_view;"
```

## 🔧 Troubleshooting

### Migration Failed (Dirty State)

```bash
# Check current state
make migrate-version

# Output example: 
# 3/d (dirty)

# Force to clean state
migrate -path migrations -database "$DATABASE_URL" force 2

# Then re-run
make migrate-up
```

### Connection Refused

```bash
# Check if PostgreSQL is running
brew services list

# Start PostgreSQL
brew services start postgresql@15

# Check connection
psql postgres -c "SELECT version();"
```

### Permission Denied

```bash
# Grant permissions
psql postgres
GRANT ALL PRIVILEGES ON DATABASE meeting_assistant TO your_user;
\q
```

## 📊 Useful Queries

### View All Tables

```sql
SELECT tablename 
FROM pg_tables 
WHERE schemaname = 'public' 
ORDER BY tablename;
```

### Table Sizes

```sql
SELECT 
    tablename,
    pg_size_pretty(pg_total_relation_size(tablename::regclass)) as size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(tablename::regclass) DESC;
```

### Row Counts

```sql
SELECT 
    schemaname,
    tablename,
    n_live_tup as row_count
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC;
```

### Active Connections

```sql
SELECT 
    datname,
    count(*) as connections
FROM pg_stat_activity
GROUP BY datname;
```

## 🐳 Docker Setup

### Using Docker Compose

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: meeting_assistant
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/migrations

volumes:
  postgres_data:
```

```bash
# Start
docker-compose up -d

# Run migrations
docker-compose exec postgres sh -c "migrate -path /migrations -database postgresql://postgres:postgres@localhost:5432/meeting_assistant?sslmode=disable up"

# Or from host
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/meeting_assistant?sslmode=disable"
make migrate-up
```

## 🔐 Production Setup

### 1. Secure Database

```sql
-- Create dedicated user
CREATE USER meeting_app WITH PASSWORD 'strong_password_here';

-- Grant permissions
GRANT CONNECT ON DATABASE meeting_assistant TO meeting_app;
GRANT USAGE ON SCHEMA public TO meeting_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO meeting_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO meeting_app;

-- For future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public 
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO meeting_app;
```

### 2. Enable SSL

```bash
# Update connection string
export DATABASE_URL="postgresql://meeting_app:password@host:5432/meeting_assistant?sslmode=require"
```

### 3. Configure Backups

```bash
# Daily backup script
#!/bin/bash
BACKUP_DIR="/backups/postgres"
DATE=$(date +%Y%m%d_%H%M%S)
pg_dump -U postgres meeting_assistant | gzip > "$BACKUP_DIR/meeting_assistant_$DATE.sql.gz"

# Keep only last 30 days
find $BACKUP_DIR -name "meeting_assistant_*.sql.gz" -mtime +30 -delete
```

### 4. Monitoring

```sql
-- Create monitoring views
CREATE VIEW db_health AS
SELECT 
    'connections' as metric,
    count(*)::text as value
FROM pg_stat_activity
UNION ALL
SELECT 
    'database_size',
    pg_size_pretty(pg_database_size(current_database()))
UNION ALL
SELECT 
    'active_rooms',
    count(*)::text
FROM rooms 
WHERE status = 'active';
```

## 🎯 Next Steps

1. ✅ Database setup complete
2. 📝 Review schema: `migrations/SCHEMA_DIAGRAM.md`
3. 🔍 Explore views: `SELECT * FROM active_meetings_view;`
4. 🧪 Run tests with seed data
5. 🚀 Start building your application!

## 📚 Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [golang-migrate Guide](https://github.com/golang-migrate/migrate)
- [Database Schema](../docs/05-database-schema.md)
- [API Documentation](../docs/06-api-documentation.md)
