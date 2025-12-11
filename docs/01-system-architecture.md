# System Architecture

## Overview

**Meeting Assistant** is a web-based meeting application with integrated AI for automatic recording, transcription, analysis, and summarization.

## Tech Stack

### Backend
- **Language**: Golang 1.24+
- **Framework**: Echo v4
- **Architecture**: Clean Architecture
- **Database**: PostgreSQL
- **Cache**: Redis

### Frontend
- **Framework**: React + TypeScript
- **UI**: Material-UI / Tailwind CSS

### Real-time Communication
- **Solution**: LiveKit (WebRTC)

### AI Services
- **Speech-to-Text**: AssemblyAI
- **Analysis**: Groq (Llama 3.1 70B)

### Infrastructure
- **Containerization**: Docker
- **Orchestration**: Docker Compose (development) / Kubernetes (production)

## Architecture Overview

**Three-tier architecture:**

1. **Frontend** - React TypeScript UI
2. **Backend** - Go API server with Clean Architecture
3. **External Services** - LiveKit, AssemblyAI, Groq

**Data Storage:**
- PostgreSQL for persistent data
- Redis for caching and sessions

## Core Services

### Authentication Service
- Google OAuth2 integration
- JWT token generation and validation
- User session management via Redis

### Room Management Service
- Create and manage meeting rooms
- Participant admission workflow
- Room status lifecycle (scheduled → active → ended)

### Recording Service
- Capture audio/video via LiveKit
- Store recordings in object storage
- Handle recording lifecycle

### AI Service
- Transcription via AssemblyAI speech-to-text
- Analysis via Groq LLM
- Generate summaries, key points, action items

### Reporting Service
- Generate per-participant reports
- Meeting analytics and dashboards
- Notification delivery

## Clean Architecture Pattern

Backend follows Clean Architecture with clear separation:

- **Domain Layer**: Business logic and entities
- **Use Case Layer**: Application-specific rules
- **Adapter Layer**: HTTP handlers and repository implementations
- **Infrastructure Layer**: External services and database drivers
