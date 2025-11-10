# Database Migrations

This directory contains database migration files for the Meeting Assistant application.

**Migration Tool**: [sql-migrate](https://github.com/rubenv/sql-migrate)

## 📁 Structure

```
migrations/
├── 001_create_initial_schema.sql
├── 002_create_recordings_and_transcripts.sql
├── 003_create_ai_summaries_and_actions.sql
├── 004_create_supporting_tables.sql
├── 005_create_views_and_maintenance.sql
├── seed.sql
├── README.md
├── SCHEMA_DIAGRAM.md
└── QUICKSTART.md
```

Each migration file contains both `-- +migrate Up` and `-- +migrate Down` sections.

## 📋 Migration List

### 001: Initial Schema
- ✅ Users table with OAuth support
- ✅ Rooms table with LiveKit integration
- ✅ Participants table with role management
- ✅ Auto-update participant count triggers
- ✅ Auto-calculate meeting duration

### 002: Recordings & Transcripts
- ✅ Recordings table with processing status
- ✅ Transcripts table with segments and words
- ✅ Full-text search on transcripts
- ✅ Speaker diarization support

### 003: AI Summaries & Actions
- ✅ Meeting summaries with structured data
- ✅ Action items with task management
- ✅ Participant reports with metrics
- ✅ Sentiment analysis support
- ✅ ClickUp integration fields

### 004: Supporting Tables
- ✅ Sessions table for refresh tokens
- ✅ Room invitations with expiry
- ✅ Notifications system
- ✅ Auto-expire invitation triggers

### 005: Views & Maintenance
- ✅ Active meetings view
- ✅ User statistics view
- ✅ Room summary view
- ✅ Pending action items view
- ✅ Cleanup functions (recordings, sessions, notifications)
- ✅ Database statistics functions

## 🚀 Running Migrations

### Method 1: Using sql-migrate CLI

#### Install sql-migrate

```bash
# Using Go
go install github.com/rubenv/sql-migrate/...@latest

# Or download binary from https://github.com/rubenv/sql-migrate/releases
```

#### Run migrations

```bash
# Apply all pending migrations
sql-migrate up

# Apply specific number of migrations
sql-migrate up -limit=1

# Check migration status
sql-migrate status

# Rollback last migration
sql-migrate down -limit=1

# Rollback all migrations
sql-migrate down
```

### Method 2: Using Make commands

```bash
make migrate-up        # Run all migrations
make migrate-down      # Rollback one migration
make migrate-status    # Check migration status
make migrate-redo      # Redo last migration
```

## 📝 Creating New Migrations

```bash
# Using sql-migrate
sql-migrate new add_user_preferences

# This creates: migrations/YYYYMMDDHHMMSS_add_user_preferences.sql
```

### Migration Template

```sql
-- +migrate Up
-- Description: Add user preferences

CREATE TABLE user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    preferences JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_user_preferences_user ON user_preferences(user_id);

-- +migrate Down
DROP TABLE IF EXISTS user_preferences;
```

## ✅ Best Practices

1. **Always use transactions** - Wrap changes in `BEGIN`/`COMMIT`
2. **Create both up and down** - Always provide rollback capability
3. **Test locally first** - Never test migrations in production
4. **Idempotent when possible** - Use `IF NOT EXISTS` and `IF EXISTS`
5. **Never modify existing** - Create new migrations instead

## 🔧 Troubleshooting

### Check Migration Status

```bash
sql-migrate status
```

### Skip Failed Migration

```bash
# Mark migration as applied without running
sql-migrate skip -limit=1
```

### Redo Last Migration

```bash
sql-migrate redo
```

## 📚 Additional Resources

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [Database Schema Design](../docs/05-database-schema.md)
