# Meeting Assistant - Technical Documentation

> Comprehensive documentation for the AI-powered Meeting Assistant system

## 📚 Documentation Structure

### Core Documentation

1. **[System Architecture](01-system-architecture.md)**
   - Overall system design and components
   - Technology stack and decisions
   - Clean Architecture implementation
   - Backend folder structure

2. **[Authentication Flow](02-authentication-flow.md)**
   - Google OAuth2 integration
   - JWT token management with sessions
   - Security considerations

3. **[Room Management Flow](03-room-management-flow.md)**
   - Creating and joining rooms
   - LiveKit integration for WebRTC
   - Participant management
   - Room lifecycle

4. **[AI Analysis Flow](04-ai-analysis-flow.md)**
   - Real-time transcription with OpenAI Whisper
   - Meeting summarization with GPT-4
   - Action items extraction
   - Sentiment analysis

5. **[Database Schema](05-database-schema.md)**
   - Complete PostgreSQL schema
   - Table relationships and indexes
   - Migration strategy

6. **[API Documentation](06-api-documentation.md)**
   - REST API endpoints (40+)
   - Request/Response formats
   - Authentication requirements
   - Error codes

7. **[Deployment Guide](07-deployment-guide.md)**
   - Docker Compose setup
   - Development environment
   - Production deployment
   - MinIO storage configuration

### Additional Resources

- **[SECURITY.md](SECURITY.md)** - Security guidelines and best practices

## 🚀 Quick Start for Team Members

### 1. Clone Repository
```bash
git clone <repository-url>
cd meeting-assistant
```

### 2. Setup Environment
```bash
# Copy environment template
cp .env.example .env

# Edit with your credentials (DO NOT commit .env!)
nano .env
```

### 3. Start Reading Documentation
- **Backend Developers**: Start with [01-system-architecture.md](01-system-architecture.md) and [06-api-documentation.md](06-api-documentation.md)
- **Frontend Developers**: Start with [06-api-documentation.md](06-api-documentation.md) and [03-room-management-flow.md](03-room-management-flow.md)
- **DevOps**: Start with [07-deployment-guide.md](07-deployment-guide.md)
- **Project Managers**: Start with [01-system-architecture.md](01-system-architecture.md) for overview

## 🛠️ Technology Stack

- **Backend**: Golang 1.21+ with Echo v4 framework
- **Database**: PostgreSQL 15+ with Redis 7+
- **Storage**: MinIO (S3-compatible, self-hosted)
- **Real-time**: LiveKit Server (WebRTC SFU)
- **AI Services**: OpenAI Whisper API + GPT-4
- **Authentication**: Google OAuth2
- **Deployment**: Docker Compose

## 📖 Reading Tips

1. **For Understanding Architecture**: Read 01 → 02 → 03 → 04 in sequence
2. **For Implementation**: Start with 06 (API) and refer to specific flow documents
3. **For Deployment**: Jump directly to 07
4. **For Database Work**: Reference 05 for complete schema

---

**Last Updated**: November 3, 2025
