# Authentication Flow

## Overview

System uses Google OAuth2 for user authentication and JWT tokens for session management.

## Authentication Stack

- **Provider**: Google OAuth2
- **Session Token**: JWT (JSON Web Token)
- **Token Storage**: Redis for session management
- **Architecture**: Clean Architecture with separate auth usecase

## OAuth2 Login Flow

**High-level process:**

1. User clicks "Login with Google" button
2. Backend generates OAuth state for validation
3. Frontend redirects to Google OAuth consent page
4. User approves permissions
5. Google redirects back with authorization code
6. Backend exchanges code for user information
7. System creates or updates user record
8. JWT tokens are generated (access + refresh)
9. Frontend stores tokens and redirects to dashboard

## Token Management

### Access Token
- **Duration**: 15 minutes
- **Purpose**: Authorize API requests
- **Contains**: User ID, email, role

### Refresh Token
- **Duration**: 7 days
- **Purpose**: Generate new access tokens
- **Storage**: Redis with automatic expiration

## Token Refresh

When access token expires:
1. Frontend detects 401 response
2. Sends refresh token to backend
3. Backend validates and generates new access token
4. Frontend retries original request

## Protected API Requests

Every authenticated request requires:
- Authorization header with Bearer token
- Valid (non-expired) JWT signature
- User must exist in database

## Logout

1. Frontend clears stored tokens
2. Backend blacklists token in Redis
3. User redirected to login page

## Security Features

- OAuth2 PKCE flow for web apps
- JWT signature verification
- Token expiration enforcement
- Refresh token rotation
- CORS configuration for allowed origins
- Secure cookie settings

## Implementation Details

See `postman_testing.md` for endpoint examples and API testing.
