# Documentation

High-level documentation for Meeting Assistant project.

## Quick Start

- **Architecture**: See `01-system-architecture.md` for system design and tech stack
- **Authentication**: See `02-authentication-flow.md` for OAuth2 and JWT token flow
- **Room Management**: See `03-room-management-flow.md` for meeting room operations
- **AI Processing**: See `04-ai-analysis-flow.md` for transcription and analysis pipeline
- **Database**: See `05-database-schema.md` for entity relationships
- **API Reference**: See `06-api-documentation.md` and `postman_testing.md` for endpoints
- **Deployment**: See `07-deployment-guide.md` for production deployment

## Project Overview

**Meeting Assistant** is a web-based meeting application with:

- **Real-time video/audio**: WebRTC via LiveKit
- **Automatic transcription**: AssemblyAI speech-to-text
- **AI analysis**: Groq LLM for summaries and insights
- **Meeting management**: Create, join, host meetings with participant approval workflow
- **Reporting**: Per-participant reports, action item tracking, meeting analytics

## Tech Stack Summary

- **Backend**: Go 1.24+ with Echo v4 framework (Clean Architecture)
- **Frontend**: React + TypeScript
- **Database**: PostgreSQL
- **Cache**: Redis
- **Real-time**: LiveKit WebRTC
- **AI Services**: AssemblyAI (STT), Groq (LLM)
- **Infrastructure**: Docker + Docker Compose

## Getting Started

1. Configure `.env` with required API keys and credentials
2. Run `docker-compose up -d` to start services
3. Run migrations with `make migrate-up`
4. Start backend with `go run cmd/api/main.go`
5. See `postman_testing.md` for API testing

## Documentation Philosophy

All documentation is high-level overview only:
- **Kept**: Architecture, design decisions, configuration, high-level flows
- **Removed**: Detailed code examples, SQL schemas, curl commands, step-by-step implementations
- **See Postman guide**: For comprehensive endpoint testing with examples

## Key Features

### Authentication
- Google OAuth2 login
- JWT token-based sessions
- Refresh token rotation

### Meeting Management
- Host-approved participant admission
- Real-time audio/video streaming
- Automatic recording capture

### AI Pipeline
- Speech-to-text with speaker diarization
- Meeting summary generation
- Action item extraction
- Sentiment analysis

### Reporting
- Per-participant meeting reports
- Action item tracking
- Meeting analytics dashboard
