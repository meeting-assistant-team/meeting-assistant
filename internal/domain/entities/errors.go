package entities

import "errors"

// Domain errors
var (
	// User errors
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidEmail      = errors.New("invalid email")
	ErrInvalidName       = errors.New("invalid name")
	ErrInvalidRole       = errors.New("invalid role")
	ErrInvalidPassword   = errors.New("invalid password")

	// OAuth errors
	ErrOAuthProviderNotSupported = errors.New("oauth provider not supported")
	ErrOAuthStateMismatch        = errors.New("oauth state mismatch")
	ErrOAuthCodeInvalid          = errors.New("oauth code invalid")

	// Session errors
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrInvalidToken    = errors.New("invalid token")

	// Generic errors
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidRequest = errors.New("invalid request")
)
