package errors

import (
	"fmt"
	"net/http"
	"time"
)

// AppError là custom error type cho application
type AppError struct {
	Raw       error
	HTTPCode  int
	Code      ErrorCode
	Message   string
	Details   map[string]string
	Timestamp time.Time
}

// Error implements error interface
func (e AppError) Error() string {
	if e.Raw != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code.String(), e.Message, e.Raw)
	}
	return fmt.Sprintf("[%s] %s", e.Code.String(), e.Message)
}

// WithDetail adds a detail to the error
func (e AppError) WithDetail(key, value string) AppError {
	if e.Details == nil {
		e.Details = make(map[string]string)
	}
	e.Details[key] = value
	return e
}

// General Errors
func ErrInternal(err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_INTERNAL,
		Message:  "Internal server error",
	}
}

func ErrInvalidArgument(message string) AppError {
	return AppError{
		HTTPCode: http.StatusBadRequest,
		Code:     ErrorCode_INVALID_ARGUMENT,
		Message:  message,
	}
}

func ErrNotFound(resource string) AppError {
	return AppError{
		HTTPCode: http.StatusNotFound,
		Code:     ErrorCode_NOT_FOUND,
		Message:  fmt.Sprintf("%s not found", resource),
	}
}

func ErrAlreadyExists(resource string) AppError {
	return AppError{
		HTTPCode: http.StatusConflict,
		Code:     ErrorCode_ALREADY_EXISTS,
		Message:  fmt.Sprintf("%s already exists", resource),
	}
}

func ErrPermissionDenied(action string) AppError {
	return AppError{
		HTTPCode: http.StatusForbidden,
		Code:     ErrorCode_PERMISSION_DENIED,
		Message:  fmt.Sprintf("Permission denied: %s", action),
	}
}

func ErrUnauthenticated() AppError {
	return AppError{
		HTTPCode: http.StatusUnauthorized,
		Code:     ErrorCode_UNAUTHENTICATED,
		Message:  "Authentication required",
	}
}

// Authentication Errors
func ErrInvalidToken() AppError {
	return AppError{
		HTTPCode: http.StatusUnauthorized,
		Code:     ErrorCode_AUTH_INVALID_TOKEN,
		Message:  "Invalid authentication token",
	}
}

func ErrTokenExpired() AppError {
	return AppError{
		HTTPCode: http.StatusUnauthorized,
		Code:     ErrorCode_AUTH_TOKEN_EXPIRED,
		Message:  "Authentication token has expired",
	}
}

func ErrInvalidCredentials() AppError {
	return AppError{
		HTTPCode: http.StatusUnauthorized,
		Code:     ErrorCode_AUTH_INVALID_CREDENTIALS,
		Message:  "Invalid email or password",
	}
}

func ErrUserNotFound() AppError {
	return AppError{
		HTTPCode: http.StatusNotFound,
		Code:     ErrorCode_AUTH_USER_NOT_FOUND,
		Message:  "User not found",
	}
}

func ErrUserAlreadyExists(email string) AppError {
	return AppError{
		HTTPCode: http.StatusConflict,
		Code:     ErrorCode_AUTH_USER_ALREADY_EXISTS,
		Message:  "User already exists",
	}.WithDetail("email", email)
}

func ErrInvalidRefreshToken() AppError {
	return AppError{
		HTTPCode: http.StatusUnauthorized,
		Code:     ErrorCode_AUTH_INVALID_REFRESH_TOKEN,
		Message:  "Invalid refresh token",
	}
}

func ErrOAuthFailed(provider string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusUnauthorized,
		Code:     ErrorCode_AUTH_OAUTH_FAILED,
		Message:  fmt.Sprintf("OAuth authentication failed with %s", provider),
	}
}

// Room Management Errors
func ErrRoomNotFound(roomID string) AppError {
	return AppError{
		HTTPCode: http.StatusNotFound,
		Code:     ErrorCode_ROOM_NOT_FOUND,
		Message:  "Room not found",
	}.WithDetail("room_id", roomID)
}

func ErrRoomAlreadyExists(roomID string) AppError {
	return AppError{
		HTTPCode: http.StatusConflict,
		Code:     ErrorCode_ROOM_ALREADY_EXISTS,
		Message:  "Room already exists",
	}.WithDetail("room_id", roomID)
}

func ErrRoomFull(roomID string, maxParticipants int) AppError {
	return AppError{
		HTTPCode: http.StatusTooManyRequests,
		Code:     ErrorCode_ROOM_FULL,
		Message:  "Room is full",
	}.WithDetail("room_id", roomID).
		WithDetail("max_participants", fmt.Sprintf("%d", maxParticipants))
}

func ErrRoomClosed(roomID string) AppError {
	return AppError{
		HTTPCode: http.StatusForbidden,
		Code:     ErrorCode_ROOM_CLOSED,
		Message:  "Room is closed",
	}.WithDetail("room_id", roomID)
}

func ErrRoomAccessDenied(roomID string) AppError {
	return AppError{
		HTTPCode: http.StatusForbidden,
		Code:     ErrorCode_ROOM_ACCESS_DENIED,
		Message:  "Access to room denied",
	}.WithDetail("room_id", roomID)
}

func ErrRoomCreationFailed(err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_ROOM_CREATION_FAILED,
		Message:  "Failed to create room",
	}
}

func ErrRoomInvalidState(roomID, currentState, expectedState string) AppError {
	return AppError{
		HTTPCode: http.StatusBadRequest,
		Code:     ErrorCode_ROOM_INVALID_STATE,
		Message:  "Room is in invalid state",
	}.WithDetail("room_id", roomID).
		WithDetail("current_state", currentState).
		WithDetail("expected_state", expectedState)
}

// Recording Errors
func ErrRecordingNotFound(recordingID string) AppError {
	return AppError{
		HTTPCode: http.StatusNotFound,
		Code:     ErrorCode_RECORDING_NOT_FOUND,
		Message:  "Recording not found",
	}.WithDetail("recording_id", recordingID)
}

func ErrRecordingStartFailed(roomID string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_RECORDING_START_FAILED,
		Message:  "Failed to start recording",
	}.WithDetail("room_id", roomID)
}

func ErrRecordingStopFailed(recordingID string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_RECORDING_STOP_FAILED,
		Message:  "Failed to stop recording",
	}.WithDetail("recording_id", recordingID)
}

func ErrRecordingUploadFailed(recordingID string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_RECORDING_UPLOAD_FAILED,
		Message:  "Failed to upload recording",
	}.WithDetail("recording_id", recordingID)
}

func ErrRecordingDownloadFailed(recordingID string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_RECORDING_DOWNLOAD_FAILED,
		Message:  "Failed to download recording",
	}.WithDetail("recording_id", recordingID)
}

// AI Analysis Errors
func ErrAIAnalysisFailed(err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_AI_ANALYSIS_FAILED,
		Message:  "AI analysis failed",
	}
}

func ErrAITranscriptionFailed(err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_AI_TRANSCRIPTION_FAILED,
		Message:  "Audio transcription failed",
	}
}

func ErrAISummaryFailed(err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_AI_SUMMARY_FAILED,
		Message:  "Failed to generate summary",
	}
}

func ErrAIServiceUnavailable(service string) AppError {
	return AppError{
		HTTPCode: http.StatusServiceUnavailable,
		Code:     ErrorCode_AI_SERVICE_UNAVAILABLE,
		Message:  "AI service temporarily unavailable",
	}.WithDetail("service", service)
}

func ErrAIQuotaExceeded() AppError {
	return AppError{
		HTTPCode: http.StatusTooManyRequests,
		Code:     ErrorCode_AI_QUOTA_EXCEEDED,
		Message:  "AI service quota exceeded",
	}
}

// Report Errors
func ErrReportNotFound(reportID string) AppError {
	return AppError{
		HTTPCode: http.StatusNotFound,
		Code:     ErrorCode_REPORT_NOT_FOUND,
		Message:  "Report not found",
	}.WithDetail("report_id", reportID)
}

func ErrReportGenerationFailed(err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_REPORT_GENERATION_FAILED,
		Message:  "Failed to generate report",
	}
}

func ErrReportExportFailed(format string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_REPORT_EXPORT_FAILED,
		Message:  "Failed to export report",
	}.WithDetail("format", format)
}

// Integration Errors
func ErrLiveKitFailed(operation string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_INTEGRATION_LIVEKIT_FAILED,
		Message:  fmt.Sprintf("LiveKit operation failed: %s", operation),
	}
}

func ErrStorageFailed(operation string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_INTEGRATION_STORAGE_FAILED,
		Message:  fmt.Sprintf("Storage operation failed: %s", operation),
	}
}

func ErrCacheFailed(operation string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_INTEGRATION_CACHE_FAILED,
		Message:  fmt.Sprintf("Cache operation failed: %s", operation),
	}
}

func ErrExternalAPIFailed(service string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_INTEGRATION_EXTERNAL_API_FAILED,
		Message:  fmt.Sprintf("External API call failed: %s", service),
	}
}

// Database Errors
func ErrDBConnectionFailed(err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_DB_CONNECTION_FAILED,
		Message:  "Database connection failed",
	}
}

func ErrDBQueryFailed(query string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_DB_QUERY_FAILED,
		Message:  "Database query failed",
	}.WithDetail("query", query)
}

func ErrDBTransactionFailed(err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_DB_TRANSACTION_FAILED,
		Message:  "Database transaction failed",
	}
}

func ErrDBConstraintViolation(constraint string, err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_DB_CONSTRAINT_VIOLATION,
		Message:  "Database constraint violation",
	}.WithDetail("constraint", constraint)
}

// Custom Errors
func ErrInvalidPayload() AppError {
	return AppError{
		HTTPCode: http.StatusBadRequest,
		Code:     ErrorCode_INVALID_PAYLOAD,
		Message:  "Invalid payload",
	}
}

func ErrMissingRecordingURL() AppError {
	return AppError{
		HTTPCode: http.StatusBadRequest,
		Code:     ErrorCode_MISSING_RECORDING_URL,
		Message:  "Missing recording URL",
	}
}

func ErrProcessingFailed(err error) AppError {
	return AppError{
		Raw:      err,
		HTTPCode: http.StatusInternalServerError,
		Code:     ErrorCode_PROCESSING_FAILED,
		Message:  "Processing failed",
	}
}

// Added missing error codes and constructors.

// ErrorCode_NOT_HOST indicates the user is not the host.
func ErrNotHost() AppError {
	return AppError{
		HTTPCode: http.StatusForbidden,
		Code:     ErrorCode_PERMISSION_DENIED,
		Message:  "User is not the host",
	}
}

// ErrorCode_PARTICIPANT_NOT_FOUND indicates a participant was not found.
func ErrParticipantNotFound(participantID string) AppError {
	return AppError{
		HTTPCode: http.StatusNotFound,
		Code:     ErrorCode_NOT_FOUND,
		Message:  "Participant not found",
	}.WithDetail("participant_id", participantID)
}

// ErrForbidden represents a forbidden error.
func ErrForbidden(message string) AppError {
	return AppError{
		HTTPCode: http.StatusForbidden,
		Code:     ErrorCode_FORBIDDEN,
		Message:  message,
	}
}

// HTTPStatusOK represents a successful HTTP response.
func HTTPStatusOK(message string) AppError {
	return AppError{
		HTTPCode: http.StatusOK,
		Code:     ErrorCode_HTTP_OK,
		Message:  message,
	}
}
