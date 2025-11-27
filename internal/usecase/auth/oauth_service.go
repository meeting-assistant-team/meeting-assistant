package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/cache"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/external/oauth"
	"github.com/johnquangdev/meeting-assistant/pkg/jwt"
)

// OAuthService handles OAuth authentication
type OAuthService struct {
	userRepo     repositories.UserRepository
	sessionRepo  repositories.SessionRepository
	cache        *cache.RedisClient
	google       *oauth.GoogleProvider
	stateManager *oauth.StateManager
	jwtManager   *jwt.Manager
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(
	userRepo repositories.UserRepository,
	sessionRepo repositories.SessionRepository,
	cache *cache.RedisClient,
	google *oauth.GoogleProvider,
	stateManager *oauth.StateManager,
	jwtManager *jwt.Manager,
) *OAuthService {
	return &OAuthService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		cache:        cache,
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

// GoogleCallbackRequest represents the callback request
type GoogleCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	User         *entities.User `json:"user"`
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token,omitempty"`
	ExpiresIn    int64          `json:"expires_in"`
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

	// Hash the refresh token before storing (we keep the raw token to return to client)
	tokenHash := s.jwtManager.HashToken(refreshToken)

	// Store hashed refresh token in session for revocation capability
	session := entities.NewSession(
		user.ID,
		tokenHash, // store hash, not raw token
		time.Now().Add(s.jwtManager.GetRefreshExpiry()),
	)

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	// Save access token to redis
	if err := s.cache.SaveAccessToken(ctx, user.ID, accessToken, s.jwtManager.GetAccessExpiry()); err != nil {
		return nil, fmt.Errorf("failed to save access token to redis: %w", err)
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.GetAccessExpiry().Seconds()),
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
	session, err := s.sessionRepo.FindByTokenHash(ctx, refreshToken)
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
	// add blacklist
	if err := s.cache.DeleteAccessToken(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to delete old access token from redis: %w", err)
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	// Save new access token to redis
	if err := s.cache.SaveAccessToken(ctx, user.ID, newAccessToken, s.jwtManager.GetAccessExpiry()); err != nil {
		return nil, fmt.Errorf("failed to save new access token to redis: %w", err)
	}
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

	session, err := s.sessionRepo.FindByTokenHash(ctx, refreshToken)
	if err != nil {
		return entities.ErrSessionNotFound
	}

	// Delete access token from redis
	if err := s.cache.DeleteAccessToken(ctx, session.UserID); err != nil {
		return fmt.Errorf("failed to delete access token from redis: %w", err)
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
