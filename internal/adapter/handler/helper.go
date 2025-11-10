package handler

import (
	"encoding/json"
	"net/http"
	"strings"
)

// RespondJSON sends a JSON response with the given status code and data
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error if needed
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// RespondError sends a JSON error response with the given status code and message
func RespondError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]interface{}{
		"error":  message,
		"status": status,
	}

	if err != nil {
		response["details"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		// Fallback to plain text if JSON encoding fails
		http.Error(w, message, status)
	}
}

// ExtractToken extracts the authentication token from the request
// It checks both the Authorization header and cookies
func ExtractToken(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return strings.TrimSpace(parts[1])
		}
	}

	// Try cookie as fallback
	cookie, err := r.Cookie("access_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}

// ParseRequestBody is a generic helper to parse JSON request body
func ParseRequestBody(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return json.NewDecoder(strings.NewReader("{}")).Decode(v)
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// SetCookie sets an HTTP cookie with common security settings
func SetCookie(w http.ResponseWriter, name, value string, maxAge int) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}

// DeleteCookie deletes an HTTP cookie by setting MaxAge to -1
func DeleteCookie(w http.ResponseWriter, name string) {
	cookie := &http.Cookie{
		Name:   name,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

// GetQueryParam is a helper to get query parameter with a default value
func GetQueryParam(r *http.Request, key, defaultValue string) string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// ValidateContentType checks if the request content type matches the expected type
func ValidateContentType(r *http.Request, expectedType string) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(strings.ToLower(contentType), strings.ToLower(expectedType))
}
