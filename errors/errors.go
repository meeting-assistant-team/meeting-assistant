package errors

import (
	"fmt"
	"net/http"
	"time"
)

// AppError là custom error type cho application
type AppError struct {
	Code        ErrorCode
	Message     string
	UserMessage string
	Details     map[string]string
	Timestamp   time.Time
	RequestID   string
	Err         error // underlying error
}

// Error implements error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code.String(), e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code.String(), e.Message)
}

// Unwrap implements errors.Unwrap interface
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key, value string) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]string)
	}
	e.Details[key] = value
	return e
}

// WithRequestID adds request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// HTTPStatusCode returns the appropriate HTTP status code for the error
func (e *AppError) HTTPStatusCode() int {
	switch e.Code {
	case ERROR_CODE_INVALID_ARGUMENT:
		return http.StatusBadRequest
	case ERROR_CODE_NOT_FOUND,
		ERROR_CODE_AUTH_USER_NOT_FOUND,
		ERROR_CODE_ROOM_NOT_FOUND,
		ERROR_CODE_RECORDING_NOT_FOUND,
		ERROR_CODE_REPORT_NOT_FOUND:
		return http.StatusNotFound
	case ERROR_CODE_ALREADY_EXISTS,
		ERROR_CODE_AUTH_USER_ALREADY_EXISTS,
		ERROR_CODE_ROOM_ALREADY_EXISTS,
		ERROR_CODE_RECORDING_ALREADY_EXISTS:
		return http.StatusConflict
	case ERROR_CODE_PERMISSION_DENIED,
		ERROR_CODE_ROOM_ACCESS_DENIED:
		return http.StatusForbidden
	case ERROR_CODE_UNAUTHENTICATED,
		ERROR_CODE_AUTH_INVALID_TOKEN,
		ERROR_CODE_AUTH_TOKEN_EXPIRED,
		ERROR_CODE_AUTH_INVALID_CREDENTIALS,
		ERROR_CODE_AUTH_INVALID_REFRESH_TOKEN:
		return http.StatusUnauthorized
	case ERROR_CODE_RESOURCE_EXHAUSTED,
		ERROR_CODE_ROOM_FULL,
		ERROR_CODE_AI_QUOTA_EXCEEDED:
		return http.StatusTooManyRequests
	case ERROR_CODE_UNIMPLEMENTED:
		return http.StatusNotImplemented
	case ERROR_CODE_UNAVAILABLE,
		ERROR_CODE_AI_SERVICE_UNAVAILABLE:
		return http.StatusServiceUnavailable
	case ERROR_CODE_DEADLINE_EXCEEDED:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// Wrap wraps an existing error with AppError
func Wrap(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:      code,
		Message:   message,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// ============================================================================
// General Errors
// ============================================================================

func ErrInternal(err error) *AppError {
	return Wrap(ERROR_CODE_INTERNAL, "Internal server error", err)
}

func ErrInvalidArgument(message string) *AppError {
	return New(ERROR_CODE_INVALID_ARGUMENT, message)
}

func ErrNotFound(resource string) *AppError {
	return New(ERROR_CODE_NOT_FOUND, fmt.Sprintf("%s not found", resource))
}

func ErrAlreadyExists(resource string) *AppError {
	return New(ERROR_CODE_ALREADY_EXISTS, fmt.Sprintf("%s already exists", resource))
}

func ErrPermissionDenied(action string) *AppError {
	return New(ERROR_CODE_PERMISSION_DENIED, fmt.Sprintf("Permission denied: %s", action))
}

func ErrUnauthenticated() *AppError {
	return New(ERROR_CODE_UNAUTHENTICATED, "Authentication required")
}

// ============================================================================
// Authentication Errors
// ============================================================================

func ErrInvalidToken() *AppError {
	return New(ERROR_CODE_AUTH_INVALID_TOKEN, "Invalid authentication token")
}

func ErrTokenExpired() *AppError {
	return New(ERROR_CODE_AUTH_TOKEN_EXPIRED, "Authentication token has expired")
}

func ErrInvalidCredentials() *AppError {
	return New(ERROR_CODE_AUTH_INVALID_CREDENTIALS, "Invalid email or password")
}

func ErrUserNotFound() *AppError {
	return New(ERROR_CODE_AUTH_USER_NOT_FOUND, "User not found")
}

func ErrUserAlreadyExists(email string) *AppError {
	return New(ERROR_CODE_AUTH_USER_ALREADY_EXISTS, "User already exists").
		WithDetail("email", email)
}

func ErrInvalidRefreshToken() *AppError {
	return New(ERROR_CODE_AUTH_INVALID_REFRESH_TOKEN, "Invalid refresh token")
}

func ErrOAuthFailed(provider string, err error) *AppError {
	return Wrap(ERROR_CODE_AUTH_OAUTH_FAILED, fmt.Sprintf("OAuth authentication failed with %s", provider), err)
}

// ============================================================================
// Room Management Errors
// ============================================================================

func ErrRoomNotFound(roomID string) *AppError {
	return New(ERROR_CODE_ROOM_NOT_FOUND, "Room not found").
		WithDetail("room_id", roomID)
}

func ErrRoomAlreadyExists(roomID string) *AppError {
	return New(ERROR_CODE_ROOM_ALREADY_EXISTS, "Room already exists").
		WithDetail("room_id", roomID)
}

func ErrRoomFull(roomID string, maxParticipants int) *AppError {
	return New(ERROR_CODE_ROOM_FULL, "Room is full").
		WithDetail("room_id", roomID).
		WithDetail("max_participants", fmt.Sprintf("%d", maxParticipants))
}

func ErrRoomClosed(roomID string) *AppError {
	return New(ERROR_CODE_ROOM_CLOSED, "Room is closed").
		WithDetail("room_id", roomID)
}

func ErrRoomAccessDenied(roomID string) *AppError {
	return New(ERROR_CODE_ROOM_ACCESS_DENIED, "Access to room denied").
		WithDetail("room_id", roomID)
}

func ErrRoomCreationFailed(err error) *AppError {
	return Wrap(ERROR_CODE_ROOM_CREATION_FAILED, "Failed to create room", err)
}

func ErrRoomInvalidState(roomID, currentState, expectedState string) *AppError {
	return New(ERROR_CODE_ROOM_INVALID_STATE, "Room is in invalid state").
		WithDetail("room_id", roomID).
		WithDetail("current_state", currentState).
		WithDetail("expected_state", expectedState)
}

// ============================================================================
// Recording Errors
// ============================================================================

func ErrRecordingNotFound(recordingID string) *AppError {
	return New(ERROR_CODE_RECORDING_NOT_FOUND, "Recording not found").
		WithDetail("recording_id", recordingID)
}

func ErrRecordingStartFailed(roomID string, err error) *AppError {
	return Wrap(ERROR_CODE_RECORDING_START_FAILED, "Failed to start recording", err).
		WithDetail("room_id", roomID)
}

func ErrRecordingStopFailed(recordingID string, err error) *AppError {
	return Wrap(ERROR_CODE_RECORDING_STOP_FAILED, "Failed to stop recording", err).
		WithDetail("recording_id", recordingID)
}

func ErrRecordingUploadFailed(recordingID string, err error) *AppError {
	return Wrap(ERROR_CODE_RECORDING_UPLOAD_FAILED, "Failed to upload recording", err).
		WithDetail("recording_id", recordingID)
}

func ErrRecordingDownloadFailed(recordingID string, err error) *AppError {
	return Wrap(ERROR_CODE_RECORDING_DOWNLOAD_FAILED, "Failed to download recording", err).
		WithDetail("recording_id", recordingID)
}

// ============================================================================
// AI Analysis Errors
// ============================================================================

func ErrAIAnalysisFailed(err error) *AppError {
	return Wrap(ERROR_CODE_AI_ANALYSIS_FAILED, "AI analysis failed", err)
}

func ErrAITranscriptionFailed(err error) *AppError {
	return Wrap(ERROR_CODE_AI_TRANSCRIPTION_FAILED, "Audio transcription failed", err)
}

func ErrAISummaryFailed(err error) *AppError {
	return Wrap(ERROR_CODE_AI_SUMMARY_FAILED, "Failed to generate summary", err)
}

func ErrAIServiceUnavailable(service string) *AppError {
	return New(ERROR_CODE_AI_SERVICE_UNAVAILABLE, "AI service temporarily unavailable").
		WithDetail("service", service)
}

func ErrAIQuotaExceeded() *AppError {
	return New(ERROR_CODE_AI_QUOTA_EXCEEDED, "AI service quota exceeded")
}

// ============================================================================
// Report Errors
// ============================================================================

func ErrReportNotFound(reportID string) *AppError {
	return New(ERROR_CODE_REPORT_NOT_FOUND, "Report not found").
		WithDetail("report_id", reportID)
}

func ErrReportGenerationFailed(err error) *AppError {
	return Wrap(ERROR_CODE_REPORT_GENERATION_FAILED, "Failed to generate report", err)
}

func ErrReportExportFailed(format string, err error) *AppError {
	return Wrap(ERROR_CODE_REPORT_EXPORT_FAILED, "Failed to export report", err).
		WithDetail("format", format)
}

// ============================================================================
// Integration Errors
// ============================================================================

func ErrLiveKitFailed(operation string, err error) *AppError {
	return Wrap(ERROR_CODE_INTEGRATION_LIVEKIT_FAILED, fmt.Sprintf("LiveKit operation failed: %s", operation), err)
}

func ErrStorageFailed(operation string, err error) *AppError {
	return Wrap(ERROR_CODE_INTEGRATION_STORAGE_FAILED, fmt.Sprintf("Storage operation failed: %s", operation), err)
}

func ErrCacheFailed(operation string, err error) *AppError {
	return Wrap(ERROR_CODE_INTEGRATION_CACHE_FAILED, fmt.Sprintf("Cache operation failed: %s", operation), err)
}

func ErrExternalAPIFailed(service string, err error) *AppError {
	return Wrap(ERROR_CODE_INTEGRATION_EXTERNAL_API_FAILED, fmt.Sprintf("External API call failed: %s", service), err)
}

// ============================================================================
// Database Errors
// ============================================================================

func ErrDBConnectionFailed(err error) *AppError {
	return Wrap(ERROR_CODE_DB_CONNECTION_FAILED, "Database connection failed", err)
}

func ErrDBQueryFailed(query string, err error) *AppError {
	return Wrap(ERROR_CODE_DB_QUERY_FAILED, "Database query failed", err).
		WithDetail("query", query)
}

func ErrDBTransactionFailed(err error) *AppError {
	return Wrap(ERROR_CODE_DB_TRANSACTION_FAILED, "Database transaction failed", err)
}

func ErrDBConstraintViolation(constraint string, err error) *AppError {
	return Wrap(ERROR_CODE_DB_CONSTRAINT_VIOLATION, "Database constraint violation", err).
		WithDetail("constraint", constraint)
}
