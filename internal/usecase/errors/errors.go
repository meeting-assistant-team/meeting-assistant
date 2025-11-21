package errors

import "errors"

// Common errors
var (
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden access")
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrConflict      = errors.New("resource conflict")
	ErrInternalError = errors.New("internal server error")
)

// Auth errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenInvalid       = errors.New("token invalid")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
)

// Room errors
var (
	ErrRoomNotFound           = errors.New("room not found")
	ErrRoomFull               = errors.New("room is full")
	ErrRoomEnded              = errors.New("room has ended")
	ErrRoomAlreadyStarted     = errors.New("room already started")
	ErrInvalidRoomType        = errors.New("invalid room type")
	ErrInvalidMaxParticipants = errors.New("max participants must be between 2 and 100")
)

// Participant errors
var (
	ErrNotHost                  = errors.New("user is not the host")
	ErrNotParticipant           = errors.New("user is not a participant")
	ErrAlreadyInRoom            = errors.New("user already in room")
	ErrParticipantNotFound      = errors.New("participant not found")
	ErrCannotRemoveSelf         = errors.New("cannot remove yourself")
	ErrCannotTransferToSelf     = errors.New("cannot transfer host to yourself")
	ErrNotInvited               = errors.New("user not invited to this room")
	ErrAccessDenied             = errors.New("access denied to this room")
	ErrTooEarly                 = errors.New("cannot join room before scheduled time")
	ErrAlreadyInvited           = errors.New("user already invited or in room")
	ErrInvalidParticipantStatus = errors.New("invalid participant status for this operation")
	ErrWaitingForHostApproval   = errors.New("waiting for host approval")
)

// Recording errors
var (
	ErrRecordingNotFound   = errors.New("recording not found")
	ErrRecordingInProgress = errors.New("recording already in progress")
	ErrRecordingNotStarted = errors.New("recording not started")
	ErrRecordingFailed     = errors.New("recording failed")
)

// LiveKit errors
var (
	ErrLivekitConnection = errors.New("failed to connect to LiveKit")
	ErrLivekitToken      = errors.New("failed to generate LiveKit token")
	ErrLivekitRoom       = errors.New("LiveKit room error")
)

// User errors
var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserNotActive    = errors.New("user is not active")
	ErrEmailAlreadyUsed = errors.New("email already in use")
)
