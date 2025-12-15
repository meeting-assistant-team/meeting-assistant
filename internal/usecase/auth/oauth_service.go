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
	userRepo     repositories.UserRepository
	sessionRepo  repositories.SessionRepository
	google       *oauth.GoogleProvider
	stateManager *oauth.StateManager
	jwtManager   *jwt.Manager
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(
	userRepo repositories.UserRepository,
	sessionRepo repositories.SessionRepository,
	google *oauth.GoogleProvider,
	stateManager *oauth.StateManager,
	jwtManager *jwt.Manager,
) *OAuthService {
	return &OAuthService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		google:       google,
		stateManager: stateManager,
		jwtManager:   jwtManager,
	}
}

// GoogleAuthURLResponse represents the response for auth URL request
type GoogleAuthURLResponse struct {
	URL   string `json:"url"`
	State string `json:"state"`
}

// GetGoogleAuthURL generates Google OAuth URL
func (s *OAuthService) GetGoogleAuthURL(ctx context.Context) (*GoogleAuthURLResponse, error) {
	state, err := s.stateManager.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	url := s.google.GetAuthURL(state)

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
	User        *entities.User `json:"user"`
	AccessToken string         `json:"access_token"`
	ExpiresIn   int64          `json:"expires_in"`
	SessionID   string         `json:"session_id,omitempty"`
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

	// Exchange code for token
	token, err := s.google.ExchangeCode(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
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

	// Create session
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session with raw refresh token
	session := entities.NewSession(
		user.ID,
		refreshToken,
		time.Now().Add(s.jwtManager.GetRefreshExpiry()),
	)

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &AuthResponse{
		User:        user,
		AccessToken: accessToken,
		ExpiresIn:   int64(s.jwtManager.GetAccessExpiry().Seconds()),
		SessionID:   session.ID.String(),
	}, nil
}

// RefreshAccessToken refreshes the access token using refresh token
func (s *OAuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Check if repositories are initialized
	if s.userRepo == nil || s.sessionRepo == nil {
		return nil, fmt.Errorf("database not initialized: cannot refresh token without DB")
	}

	// Validate refresh token
	userID, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if session exists and not revoked
	session, err := s.sessionRepo.FindByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if !session.IsValid() {
		return nil, entities.ErrSessionExpired
	}

	// Find user
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &AuthResponse{
		User:        user,
		AccessToken: newAccessToken,
		ExpiresIn:   int64(s.jwtManager.GetAccessExpiry().Seconds()),
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

// Logout revokes a session
func (s *OAuthService) Logout(ctx context.Context, refreshToken string) error {
	// Check if repositories are initialized
	if s.sessionRepo == nil {
		return fmt.Errorf("database not initialized: cannot logout without DB")
	}

	session, err := s.sessionRepo.FindByRefreshToken(ctx, refreshToken)
	if err != nil {
		return entities.ErrSessionNotFound
	}

	return s.sessionRepo.Revoke(ctx, session.ID)
}

// LogoutAll revokes all sessions for a user
func (s *OAuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	// Check if repositories are initialized
	if s.sessionRepo == nil {
		return fmt.Errorf("database not initialized: cannot logout without DB")
	}

	return s.sessionRepo.RevokeAllByUserID(ctx, userID)
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
