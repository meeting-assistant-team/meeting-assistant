package handler

import (
	"encoding/json"
	stdErrors "errors"
	"net/http"
	"strings"

	"github.com/johnquangdev/meeting-assistant/errors"
	"github.com/johnquangdev/meeting-assistant/internal/adapter/dto/room"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/domain/repositories"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
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
	appErr := errors.AppError{
		HTTPCode: status,
		Message:  message,
		Raw:      err,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if encErr := json.NewEncoder(w).Encode(appErr); encErr != nil {
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

// buildFilters converts ListRoomsRequest to repository filters
func buildFilters(req *room.ListRoomsRequest) repositories.RoomFilters {
	filters := repositories.RoomFilters{
		Search:    req.Search,
		Tags:      req.Tags,
		Limit:     req.PageSize,
		Offset:    (req.Page - 1) * req.PageSize,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	if req.Type != nil {
		roomType := entities.RoomType(*req.Type)
		filters.Type = &roomType
	}

	if req.Status != nil {
		roomStatus := entities.RoomStatus(*req.Status)
		filters.Status = &roomStatus
	}

	return filters
}

// Response shapes
type success struct {
	Code    interface{} `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type errs struct {
	Code    interface{} `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Info    string      `json:"info,omitempty"`
}

// getRequestID tries to read X-Request-ID from the request
func getRequestID(c echo.Context) string {
	if c == nil || c.Request() == nil {
		return ""
	}
	return c.Request().Header.Get("X-Request-ID")
}

// handleSuccess writes a standardized success response
func (h *Room) handleSuccess(c echo.Context, data interface{}) error {
	resp := success{
		Code:    int(errors.ErrorCode_HTTP_OK),
		Message: "success",
		Data:    data,
	}

	if h != nil && h.logger != nil {
		h.logger.Info("http.response.success",
			zap.String("request_id", getRequestID(c)),
			zap.String("path", c.Path()),
		)
	}

	return c.JSON(http.StatusOK, resp)
}

// handleError centralizes error handling and logging
func (h *Room) handleError(c echo.Context, err error) error {
	reqID := getRequestID(c)

	// Try to detect AppError from project errors package
	var appErr errors.AppError
	if stdErrors.As(err, &appErr) {
		// Structured logging
		if h != nil && h.logger != nil {
			h.logger.Error("http.response.error",
				zap.String("request_id", reqID),
				zap.String("path", c.Path()),
				zap.Any("app_code", appErr.Code),
				zap.Error(err),
			)
		}

		info := ""
		if appErr.Raw != nil {
			info = appErr.Raw.Error()
		}

		body := errs{
			Code:    appErr.Code,
			Message: appErr.Message,
			Info:    info,
		}

		return c.JSON(appErr.HTTPCode, body)
	}

	// Non-AppError => internal server error
	if h != nil && h.logger != nil {
		h.logger.Error("http.response.error",
			zap.String("request_id", reqID),
			zap.String("path", c.Path()),
			zap.Error(err),
		)
	}

	body := errs{
		Code:    errors.ErrorCode_INTERNAL,
		Message: "Internal server error",
		Info:    err.Error(),
	}

	return c.JSON(http.StatusInternalServerError, body)
}

// HandleSuccess writes a standardized success response using provided logger
func HandleSuccess(logger *zap.Logger, c echo.Context, data interface{}) error {
	resp := success{
		Code:    int(errors.ErrorCode_HTTP_OK),
		Message: "success",
		Data:    data,
	}

	if logger != nil {
		logger.Info("http.response.success",
			zap.String("request_id", getRequestID(c)),
			zap.String("path", c.Path()),
		)
	}

	return c.JSON(http.StatusOK, resp)
}

// HandleError centralizes error handling and logging using provided logger
func HandleError(logger *zap.Logger, c echo.Context, err error) error {
	reqID := getRequestID(c)

	var appErr errors.AppError
	if stdErrors.As(err, &appErr) {
		if logger != nil {
			logger.Error("http.response.error",
				zap.String("request_id", reqID),
				zap.String("path", c.Path()),
				zap.Any("app_code", appErr.Code),
				zap.Error(err),
			)
		}

		info := ""
		if appErr.Raw != nil {
			info = appErr.Raw.Error()
		}

		body := errs{
			Code:    appErr.Code,
			Message: appErr.Message,
			Info:    info,
		}

		return c.JSON(appErr.HTTPCode, body)
	}

	if logger != nil {
		logger.Error("http.response.error",
			zap.String("request_id", reqID),
			zap.String("path", c.Path()),
			zap.Error(err),
		)
	}

	body := errs{
		Code:    errors.ErrorCode_INTERNAL,
		Message: "Internal server error",
		Info:    err.Error(),
	}

	return c.JSON(http.StatusInternalServerError, body)
}
