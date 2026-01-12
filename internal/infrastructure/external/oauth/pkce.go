package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// PKCEManager manages PKCE (Proof Key for Code Exchange) parameters for OAuth2
// Implements RFC 7636 for enhanced security in OAuth2 flows
type PKCEManager struct {
	store      Store
	expiration time.Duration
}

// PKCEParams holds the code verifier and challenge for PKCE flow
type PKCEParams struct {
	CodeVerifier  string
	CodeChallenge string
	Method        string // Always "S256" for SHA256
}

// NewPKCEManager creates a new PKCE manager
func NewPKCEManager(store Store) *PKCEManager {
	return &PKCEManager{
		store:      store,
		expiration: 15 * time.Minute, // Same as state expiration
	}
}

// GeneratePKCEParams generates PKCE parameters (code_verifier and code_challenge)
// Per RFC 7636: code_verifier must be 43-128 characters, we use 64 bytes (86 chars base64url)
func (pm *PKCEManager) GeneratePKCEParams() (*PKCEParams, error) {
	// Generate random code_verifier (64 bytes = 86 chars when base64url encoded)
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	codeVerifier := base64.RawURLEncoding.EncodeToString(b)

	// Generate code_challenge = BASE64URL(SHA256(code_verifier))
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCEParams{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		Method:        "S256", // SHA256
	}, nil
}

// StorePKCEVerifier stores the code_verifier associated with a state token
// The verifier is needed later to exchange the authorization code for tokens
func (pm *PKCEManager) StorePKCEVerifier(state, codeVerifier string) {
	key := fmt.Sprintf("oauth:pkce:%s", state)
	pm.store.Set(key, codeVerifier, pm.expiration)
}

// GetPKCEVerifier retrieves and deletes the code_verifier for a given state (one-time use)
func (pm *PKCEManager) GetPKCEVerifier(state string) (string, bool) {
	key := fmt.Sprintf("oauth:pkce:%s", state)

	// Get the verifier
	verifier, exists := pm.store.Get(key)
	if !exists {
		return "", false
	}

	// Delete immediately (one-time use)
	pm.store.Delete(key)

	return verifier, true
}

// ValidateCodeChallenge validates that a code_verifier matches the expected code_challenge
// This is primarily for testing; actual validation happens at Google's token endpoint
func (pm *PKCEManager) ValidateCodeChallenge(codeVerifier, expectedChallenge string) bool {
	hash := sha256.Sum256([]byte(codeVerifier))
	actualChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	return actualChallenge == expectedChallenge
}
