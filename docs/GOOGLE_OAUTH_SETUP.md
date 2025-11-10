# Google OAuth2 Setup Guide

## 📋 Overview

Hướng dẫn chi tiết để setup Google OAuth2 authentication cho Meeting Assistant application.

## 🎯 Prerequisites

- Google Account
- Meeting Assistant application đã setup database và Docker services
- Go 1.21+ đã cài đặt

## 📝 Step-by-Step Setup

### 1. Tạo Google Cloud Project

1. Truy cập [Google Cloud Console](https://console.cloud.google.com/)
2. Click **Select a project** → **New Project**
3. Nhập tên project: `meeting-assistant` (hoặc tên bạn muốn)
4. Click **Create**

### 2. Enable Google+ API

1. Trong Google Cloud Console, vào **APIs & Services** → **Library**
2. Tìm kiếm "Google+ API"
3. Click vào **Google+ API** và click **Enable**
4. Hoặc enable "Google Identity" API (recommended)

### 3. Tạo OAuth Consent Screen

1. Vào **APIs & Services** → **OAuth consent screen**
2. Chọn **External** (cho testing) hoặc **Internal** (nếu có Google Workspace)
3. Click **Create**

**App Information:**
- App name: `Meeting Assistant`
- User support email: Your email
- Developer contact email: Your email

**Scopes:**
- Click **Add or Remove Scopes**
- Thêm các scopes sau:
  - `.../auth/userinfo.email`
  - `.../auth/userinfo.profile`
- Click **Update**

**Test Users (cho External):**
- Click **Add Users**
- Thêm email addresses của bạn và team members
- Click **Save and Continue**

### 4. Tạo OAuth Credentials

1. Vào **APIs & Services** → **Credentials**
2. Click **Create Credentials** → **OAuth client ID**
3. Chọn Application type: **Web application**

**Configuration:**
- Name: `Meeting Assistant Web Client`
- **Authorized JavaScript origins:**
  - `http://localhost:8080`
  - `http://localhost:3000` (nếu có frontend)
- **Authorized redirect URIs:**
  - `http://localhost:8080/api/v1/auth/google/callback`
  - `https://yourdomain.com/api/v1/auth/google/callback` (production)

4. Click **Create**
5. **Copy** Client ID và Client Secret

### 5. Configure Application

1. Copy file `.env.example` thành `.env`:
```bash
cp .env.example .env
```

2. Điền thông tin Google OAuth vào `.env`:
```bash
GOOGLE_CLIENT_ID=your-actual-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-actual-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback
```

### 6. Start Services

```bash
# Start Docker services
docker-compose up -d

# Run migrations
make migrate-up

# Start application
go run cmd/api/main.go
```

## 🧪 Testing OAuth Flow

### Test với cURL

**1. Get Google Login URL:**
```bash
curl http://localhost:8080/api/v1/auth/google/login
```

Response:
```json
{
  "url": "https://accounts.google.com/o/oauth2/auth?client_id=...",
  "state": "random-state-token"
}
```

**2. Open URL in Browser:**
- Copy `url` từ response
- Paste vào browser
- Login với Google account
- Grant permissions
- Browser sẽ redirect về callback URL với code

**3. Exchange Code for Token:**
Browser sẽ tự động call callback endpoint:
```
http://localhost:8080/api/v1/auth/google/callback?code=xxx&state=xxx
```

Response:
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "User Name",
    "role": "participant",
    "avatar_url": "https://...",
    "is_email_verified": true,
    "created_at": "2024-01-01T00:00:00Z"
  },
  "access_token": "session-token-uuid",
  "refresh_token": "google-refresh-token",
  "expires_in": 604800
}
```

**4. Test Protected Endpoint:**
```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  http://localhost:8080/api/v1/auth/me
```

Response:
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "User Name",
    ...
  }
}
```

**5. Refresh Token:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "google-refresh-token"}'
```

**6. Logout:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## 🔐 Security Best Practices

### 1. Environment Variables
- ❌ **NEVER** commit `.env` file
- ✅ Keep `.env.example` updated
- ✅ Use different credentials for dev/staging/prod

### 2. State Parameter
- State token được generate randomly cho mỗi request
- Validates OAuth callback to prevent CSRF attacks
- Automatically handled by `StateManager`

### 3. Token Storage
- Access tokens stored as SHA256 hash in database
- Refresh tokens encrypted before storage
- Sessions expire after 7 days (configurable)

### 4. HTTPS in Production
- **ALWAYS** use HTTPS in production
- Update redirect URIs to use `https://`
- Set `Secure` flag on cookies

## 🏗️ Architecture

```
┌─────────────┐
│   Browser   │
└──────┬──────┘
       │ 1. GET /auth/google/login
       ▼
┌─────────────────────────────────────┐
│         Auth Handler                │
│  - GenerateState()                  │
│  - GetGoogleAuthURL()               │
└──────┬──────────────────────────────┘
       │ 2. Return Google Auth URL
       ▼
┌─────────────┐
│   Browser   │ 3. Redirect to Google
└──────┬──────┘
       │ 4. User logs in & grants permissions
       ▼
┌─────────────┐
│   Google    │ 5. Redirect to callback with code
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│    Auth Handler (Callback)          │
│  - ValidateState()                  │
│  - ExchangeCode()                   │
│  - GetUserInfo()                    │
│  - FindOrCreateUser()               │
│  - CreateSession()                  │
└──────┬──────────────────────────────┘
       │ 6. Return access token
       ▼
┌─────────────┐
│   Browser   │ 7. Store token & use for API calls
└─────────────┘
```

## 📊 Database Schema

**Users Table:**
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    oauth_provider VARCHAR(50),      -- 'google'
    oauth_id VARCHAR(255),            -- Google user ID
    oauth_refresh_token TEXT,         -- Encrypted refresh token
    avatar_url TEXT,
    is_email_verified BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

**Sessions Table:**
```sql
CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    token_hash VARCHAR(64) UNIQUE,    -- SHA256 hash
    expires_at TIMESTAMP,
    created_at TIMESTAMP,
    revoked_at TIMESTAMP
);
```

## 🔧 Troubleshooting

### Error: "redirect_uri_mismatch"
- Check redirect URI in Google Console matches exactly
- Include protocol (`http://` or `https://`)
- No trailing slashes

### Error: "Access blocked: This app's request is invalid"
- Configure OAuth consent screen
- Add test users (for External type)
- Enable required APIs

### Error: "invalid_client"
- Check Client ID and Secret are correct
- No extra spaces in `.env` file
- Credentials match the project

### Error: "state mismatch"
- State tokens expire after use
- Check StateManager is working
- Consider using Redis for distributed systems

## 🚀 Next Steps

1. **Frontend Integration:**
   - Create login button linking to `/api/v1/auth/google/login`
   - Handle callback and store token
   - Add token to all API requests

2. **Implement Other Features:**
   - Room management
   - Meeting recording
   - AI transcription
   - Report generation

3. **Production Deployment:**
   - Update redirect URIs
   - Enable HTTPS
   - Use production credentials
   - Configure proper CORS

4. **Monitoring:**
   - Log OAuth events
   - Track failed login attempts
   - Monitor token refresh rates

## 📚 References

- [Google OAuth2 Documentation](https://developers.google.com/identity/protocols/oauth2)
- [OAuth 2.0 RFC](https://tools.ietf.org/html/rfc6749)
- [OWASP OAuth Security](https://cheatsheetseries.owasp.org/cheatsheets/OAuth2_Cheat_Sheet.html)
