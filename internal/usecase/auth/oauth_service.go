package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/external/oauth"
	"github.com/johnquangdev/meeting-assistant/pkg/jwt"
)

// OAuthService handles OAuth authentication
type OAuthService struct {
	userRepo        repositories.UserRepository
	sessionRepo     repositories.SessionRepository
	tokenFamilyRepo repositories.TokenFamilyRepository
	google          *oauth.GoogleProvider
	stateManager    *oauth.StateManager
	pkceManager     *oauth.PKCEManager
	jwtManager      *jwt.Manager
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(
	userRepo repositories.UserRepository,
	sessionRepo repositories.SessionRepository,
	tokenFamilyRepo repositories.TokenFamilyRepository,
	google *oauth.GoogleProvider,
	stateManager *oauth.StateManager,
	pkceManager *oauth.PKCEManager,
	jwtManager *jwt.Manager,
) *OAuthService {
	return &OAuthService{
		userRepo:        userRepo,
		sessionRepo:     sessionRepo,
		tokenFamilyRepo: tokenFamilyRepo,
		google:          google,
		stateManager:    stateManager,
		pkceManager:     pkceManager,
		jwtManager:      jwtManager,
	}
}

// GoogleAuthURLResponse represents the response for auth URL request
type GoogleAuthURLResponse struct {
	URL   string `json:"url"`
	State string `json:"state"`
}

// GetGoogleAuthURL generates Google OAuth URL with PKCE (RFC 7636)
func (s *OAuthService) GetGoogleAuthURL(ctx context.Context) (*GoogleAuthURLResponse, error) {
	// Generate state for CSRF protection
	state, err := s.stateManager.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Generate PKCE parameters for enhanced security
	pkceParams, err := s.pkceManager.GeneratePKCEParams()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKCE params: %w", err)
	}

	// Store code_verifier associated with this state (needed for token exchange)
	s.pkceManager.StorePKCEVerifier(state, pkceParams.CodeVerifier)

	// Generate OAuth URL with PKCE code_challenge
	url := s.google.GetAuthURLWithPKCE(state, pkceParams.CodeChallenge)

	return &GoogleAuthURLResponse{
		URL:   url,
		State: state,
	}, nil
}

// ValidateState validates OAuth state from in-memory store (one-time use)
func (s *OAuthService) ValidateState(state string) bool {
	return s.stateManager.ValidateState(state)
}

// GoogleCallbackRequest represents the callback request
type GoogleCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	User         *entities.User `json:"user"`
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token,omitempty"` // Returned in OAuth2 standard flow
	ExpiresIn    int64          `json:"expires_in"`
	SessionID    string         `json:"session_id,omitempty"` // Deprecated - for backwards compatibility only
}

// HandleGoogleCallback handles the OAuth callback from Google
func (s *OAuthService) HandleGoogleCallback(ctx context.Context, req *GoogleCallbackRequest) (*AuthResponse, error) {
	// Check if repositories are initialized
	if s.userRepo == nil || s.sessionRepo == nil {
		return nil, fmt.Errorf("database not initialized: userRepo=%v, sessionRepo=%v", s.userRepo != nil, s.sessionRepo != nil)
	}

	// Validate state
	if !s.stateManager.ValidateState(req.State) {
		return nil, entities.ErrOAuthStateMismatch
	}

	// Get PKCE code_verifier for this state
	codeVerifier, found := s.pkceManager.GetPKCEVerifier(req.State)
	if !found {
		return nil, fmt.Errorf("PKCE code_verifier not found for state - possible CSRF attack or expired session")
	}

	// Exchange code for token WITH PKCE verification
	token, err := s.google.ExchangeCodeWithPKCE(ctx, req.Code, codeVerifier)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code with PKCE: %w", err)
	}

	// Get user info from Google
	googleUser, err := s.google.GetUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Find or create user
	user, err := s.userRepo.FindByOAuth(ctx, "google", googleUser.ID)
	if err != nil {
		if err == entities.ErrUserNotFound {
			// Check if user with this email already exists
			existingUser, err := s.userRepo.FindByEmail(ctx, googleUser.Email)
			if err == nil {
				// User exists with different auth method, link accounts
				provider := "google"
				existingUser.OAuthProvider = &provider
				existingUser.OAuthID = &googleUser.ID
				existingUser.AvatarURL = &googleUser.Picture

				if token.RefreshToken != "" {
					existingUser.OAuthRefreshToken = &token.RefreshToken
				}

				if err := s.userRepo.Update(ctx, existingUser); err != nil {
					return nil, fmt.Errorf("failed to link accounts: %w", err)
				}
				user = existingUser
			} else {
				// Create new user
				user = entities.NewOAuthUser(googleUser.Email, googleUser.Name, "google", googleUser.ID)
				user.AvatarURL = &googleUser.Picture

				if token.RefreshToken != "" {
					user.OAuthRefreshToken = &token.RefreshToken
				}

				// Set language from locale
				if googleUser.Locale != "" {
					user.Language = googleUser.Locale
				}

				if err := s.userRepo.Create(ctx, user); err != nil {
					return nil, fmt.Errorf("failed to create user: %w", err)
				}
			}
		} else {
			return nil, fmt.Errorf("failed to find user: %w", err)
		}
	} else {
		// Update existing OAuth user
		user.UpdateLastLogin()
		user.AvatarURL = &googleUser.Picture

		if token.RefreshToken != "" {
			user.OAuthRefreshToken = &token.RefreshToken
		}

		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create token family for rotation tracking (OAuth2 security best practice)
	tokenFamily := entities.NewTokenFamily(
		user.ID,
		refreshToken,
		time.Now().Add(s.jwtManager.GetRefreshExpiry()),
	)

	if err := s.tokenFamilyRepo.Create(ctx, tokenFamily); err != nil {
		return nil, fmt.Errorf("failed to create token family: %w", err)
	}

	// DEPRECATED: Also create session for backwards compatibility (will be removed)
	session := entities.NewSession(
		user.ID,
		refreshToken,
		time.Now().Add(s.jwtManager.GetRefreshExpiry()),
	)
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		// Non-fatal - session is deprecated
		fmt.Printf("⚠️  [DEPRECATED] Failed to create session: %v\n", err)
	}

	// Return both access_token and refresh_token per OAuth2 RFC 6749
	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // OAuth2 standard - client stores this
		ExpiresIn:    int64(s.jwtManager.GetAccessExpiry().Seconds()),
		SessionID:    session.ID.String(), // Deprecated
	}, nil
}

// RefreshAccessToken refreshes the access token using refresh token with rotation
// Implements OAuth2 RFC 6749 with token rotation for security
func (s *OAuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Validate refresh token JWT
	userID, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Hash the token to lookup in database
	tokenHash := entities.HashToken(refreshToken)

	// Find token family by hash with SELECT FOR UPDATE lock
	// This prevents race condition when multiple requests try to rotate the same token
	tokenFamily, err := s.tokenFamilyRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		// Token not found - could be:
		// 1. Already rotated (used token)
		// 2. Stolen and reused
		// Check if this token was part of a revoked family
		return nil, entities.ErrInvalidToken
	}

	// Check if token is valid
	if !tokenFamily.IsValid() {
		return nil, entities.ErrSessionExpired
	}
	
	// Double-check: If token is already revoked (by concurrent request), reject it
	if tokenFamily.RevokedAt != nil {
		fmt.Printf("⚠️ [REFRESH] Token already revoked (race condition handled)\n")
		return nil, entities.ErrInvalidToken
	}

	// SECURITY: Check for token reuse (theft detection)
	// If this token has already been rotated, it means someone is reusing an old token
	// This is a strong indicator of token theft
	allTokensInFamily, err := s.tokenFamilyRepo.FindByFamilyID(ctx, tokenFamily.FamilyID)
	if err == nil && len(allTokensInFamily) > 0 {
		// Check if there's a newer token in the family (means this one was already rotated)
		for _, tf := range allTokensInFamily {
			if tf.ParentTokenHash != nil && *tf.ParentTokenHash == tokenHash {
				// This token was already rotated! Possible theft detected
				// Revoke the entire token family
				fmt.Printf("🚨 [SECURITY] Token reuse detected! Revoking family: %s\n", tokenFamily.FamilyID)
				_ = s.tokenFamilyRepo.RevokeFamily(ctx, tokenFamily.FamilyID)
				return nil, entities.ErrTokenReuse
			}
		}
	}

	// Find user
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Generate NEW access token
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate NEW refresh token (rotation)
	newRefreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create new token family entry (rotated token)
	newTokenFamily := tokenFamily.RotateToken(
		newRefreshToken,
		time.Now().Add(s.jwtManager.GetRefreshExpiry()),
	)

	if err := s.tokenFamilyRepo.Create(ctx, newTokenFamily); err != nil {
		return nil, fmt.Errorf("failed to create rotated token: %w", err)
	}

	// Revoke old token (single use only) - IMPORTANT: Update in database!
	tokenFamily.Revoke()
	if err := s.tokenFamilyRepo.Update(ctx, tokenFamily); err != nil {
		// Log error but don't fail the request - new token already created
		fmt.Printf("⚠️ [WARNING] Failed to revoke old token in DB: %v\n", err)
	}

	fmt.Printf("✅ [REFRESH] Token rotated successfully for user: %s\n", user.ID)

	// Return NEW tokens (both access and refresh per OAuth2 standard)
	return &AuthResponse{
		User:         user,
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken, // OAuth2 token rotation
		ExpiresIn:    int64(s.jwtManager.GetAccessExpiry().Seconds()),
	}, nil
}

// RefreshAccessTokenBySessionID refreshes access token using a session ID stored server-side
func (s *OAuthService) RefreshAccessTokenBySessionID(ctx context.Context, sessionID uuid.UUID) (*AuthResponse, error) {
	// Check repos
	if s.sessionRepo == nil || s.userRepo == nil {
		return nil, fmt.Errorf("database not initialized: cannot refresh token without DB")
	}

	fmt.Printf("🔍 [REFRESH] Looking for session: %s\n", sessionID.String())

	// Find session by ID
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		fmt.Printf("🔴 [REFRESH] Session not found: %v\n", err)
		return nil, entities.ErrSessionNotFound
	}

	fmt.Printf("✅ [REFRESH] Session found, checking expiry...\n")

	// Check if session is expired or revoked
	if session.IsExpired() {
		fmt.Printf("🔴 [REFRESH] Session expired\n")
		return nil, entities.ErrSessionExpired
	}
	if session.RevokedAt != nil {
		fmt.Printf("🔴 [REFRESH] Session revoked\n")
		return nil, entities.ErrInvalidToken
	}

	fmt.Printf("✅ [REFRESH] Session valid, generating new access token\n")

	// Update last used (non-fatal)
	_ = s.sessionRepo.UpdateLastUsed(ctx, session.ID)

	// Find user
	user, err := s.userRepo.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	fmt.Printf("✅ [REFRESH] New access token generated\n")

	return &AuthResponse{
		User:        user,
		AccessToken: newAccessToken,
		ExpiresIn:   int64(s.jwtManager.GetAccessExpiry().Seconds()),
	}, nil
}

// ValidateSession validates a session token
func (s *OAuthService) ValidateSession(ctx context.Context, token string) (*entities.User, error) {
	// Check if repositories are initialized
	if s.userRepo == nil {
		return nil, fmt.Errorf("database not initialized: cannot validate session without DB")
	}

	// Validate JWT access token
	claims, err := s.jwtManager.ValidateAccessToken(token)
	if err != nil {
		return nil, entities.ErrInvalidToken
	}

	// Get user from database
	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, entities.ErrUnauthorized
	}

	return user, nil
}

// Logout revokes a refresh token and its family
func (s *OAuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := entities.HashToken(refreshToken)

	// Find token family
	tokenFamily, err := s.tokenFamilyRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		// Also try legacy session-based logout (backwards compatibility)
		if s.sessionRepo != nil {
			session, err := s.sessionRepo.FindByRefreshToken(ctx, refreshToken)
			if err == nil {
				return s.sessionRepo.Revoke(ctx, session.ID)
			}
		}
		return entities.ErrSessionNotFound
	}

	// Revoke entire token family (all rotated tokens)
	return s.tokenFamilyRepo.RevokeFamily(ctx, tokenFamily.FamilyID)
}

// LogoutAll revokes all token families for a user
func (s *OAuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	// Revoke all token families
	if err := s.tokenFamilyRepo.RevokeAllByUserID(ctx, userID); err != nil {
		return err
	}

	// Also revoke legacy sessions (backwards compatibility)
	if s.sessionRepo != nil {
		_ = s.sessionRepo.RevokeAllByUserID(ctx, userID)
	}

	return nil
}

// RevokeSessionByID revokes a session by its UUID
func (s *OAuthService) RevokeSessionByID(ctx context.Context, sessionID uuid.UUID) error {
	if s.sessionRepo == nil {
		return fmt.Errorf("database not initialized: cannot revoke session without DB")
	}
	return s.sessionRepo.Revoke(ctx, sessionID)
}

// ValidateSessionByID validates a session by its session ID and returns the associated user
func (s *OAuthService) ValidateSessionByID(ctx context.Context, sessionID uuid.UUID) (*entities.User, error) {
	if s.sessionRepo == nil || s.userRepo == nil {
		return nil, fmt.Errorf("database not initialized: cannot validate session")
	}

	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if !session.IsValid() {
		return nil, entities.ErrSessionExpired
	}

	user, err := s.userRepo.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// CreateTestAccessToken creates a test JWT token for development/testing (DO NOT USE IN PRODUCTION)
func (s *OAuthService) CreateTestAccessToken(ctx context.Context, userID uuid.UUID, email string) (*AuthResponse, error) {
	// Only for development/testing
	accessToken, err := s.jwtManager.GenerateAccessToken(userID, email, "user")
	if err != nil {
		return nil, fmt.Errorf("failed to generate test access token: %w", err)
	}

	return &AuthResponse{
		User: &entities.User{
			ID:    userID,
			Email: email,
			Name:  "Test User",
		},
		AccessToken: accessToken,
		ExpiresIn:   int64(s.jwtManager.GetAccessExpiry().Seconds() * 1000),
	}, nil
}
