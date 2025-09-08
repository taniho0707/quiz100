package errors

import (
	"fmt"
	"net/http"
	"time"
)

// ErrorCode represents different types of quiz application errors
type ErrorCode string

const (
	// User-related errors
	ErrCodeUserNotFound      ErrorCode = "USER_NOT_FOUND"
	ErrCodeUserAlreadyJoined ErrorCode = "USER_ALREADY_JOINED"
	ErrCodeSessionExpired    ErrorCode = "SESSION_EXPIRED"
	ErrCodeInvalidNickname   ErrorCode = "INVALID_NICKNAME"

	// Question-related errors
	ErrCodeQuestionNotFound   ErrorCode = "QUESTION_NOT_FOUND"
	ErrCodeInvalidQuestionNum ErrorCode = "INVALID_QUESTION_NUMBER"
	ErrCodeAlreadyAnswered    ErrorCode = "ALREADY_ANSWERED"
	ErrCodeInvalidAnswer      ErrorCode = "INVALID_ANSWER"

	// State-related errors
	ErrCodeInvalidStateTransition ErrorCode = "INVALID_STATE_TRANSITION"
	ErrCodeStateNotAllowed        ErrorCode = "STATE_NOT_ALLOWED"
	ErrCodeEventNotStarted        ErrorCode = "EVENT_NOT_STARTED"
	ErrCodeEventAlreadyStarted    ErrorCode = "EVENT_ALREADY_STARTED"

	// Team-related errors
	ErrCodeTeamNotFound         ErrorCode = "TEAM_NOT_FOUND"
	ErrCodeTeamFull             ErrorCode = "TEAM_FULL"
	ErrCodeNotInTeamMode        ErrorCode = "NOT_IN_TEAM_MODE"
	ErrCodeTeamAssignmentFailed ErrorCode = "TEAM_ASSIGNMENT_FAILED"

	// WebSocket-related errors
	ErrCodeWebSocketClosed ErrorCode = "WEBSOCKET_CLOSED"
	ErrCodeWebSocketError  ErrorCode = "WEBSOCKET_ERROR"
	ErrCodeConnectionLost  ErrorCode = "CONNECTION_LOST"
	ErrCodeBroadcastFailed ErrorCode = "BROADCAST_FAILED"

	// Database-related errors
	ErrCodeDatabaseError ErrorCode = "DATABASE_ERROR"
	ErrCodeDataNotFound  ErrorCode = "DATA_NOT_FOUND"
	ErrCodeDataCorrupted ErrorCode = "DATA_CORRUPTED"

	// Validation errors
	ErrCodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	ErrCodeInvalidRequest   ErrorCode = "INVALID_REQUEST"
	ErrCodeMissingParameter ErrorCode = "MISSING_PARAMETER"
	ErrCodeInvalidParameter ErrorCode = "INVALID_PARAMETER"

	// System errors
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeTimeout            ErrorCode = "TIMEOUT"
	ErrCodeResourceExhausted  ErrorCode = "RESOURCE_EXHAUSTED"
)

// QuizError represents a standardized error in the quiz application
type QuizError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	HTTPStatus int       `json:"-"`
	Cause      error     `json:"-"`
}

// Error implements the error interface
func (e *QuizError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *QuizError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for errors.Is()
func (e *QuizError) Is(target error) bool {
	if t, ok := target.(*QuizError); ok {
		return e.Code == t.Code
	}
	return false
}

// WithDetails adds details to the error
func (e *QuizError) WithDetails(details string) *QuizError {
	newErr := *e
	newErr.Details = details
	return &newErr
}

// WithCause adds a cause error
func (e *QuizError) WithCause(cause error) *QuizError {
	newErr := *e
	newErr.Cause = cause
	return &newErr
}

// NewQuizError creates a new QuizError
func NewQuizError(code ErrorCode, message string, httpStatus int) *QuizError {
	return &QuizError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Predefined common errors
var (
	// User errors
	ErrUserNotFound = NewQuizError(
		ErrCodeUserNotFound,
		"ユーザーが見つかりません",
		http.StatusNotFound,
	)

	ErrUserAlreadyJoined = NewQuizError(
		ErrCodeUserAlreadyJoined,
		"既に参加済みです",
		http.StatusConflict,
	)

	ErrSessionExpired = NewQuizError(
		ErrCodeSessionExpired,
		"セッションが期限切れです",
		http.StatusUnauthorized,
	)

	ErrInvalidNickname = NewQuizError(
		ErrCodeInvalidNickname,
		"ニックネームが無効です",
		http.StatusBadRequest,
	)

	// Question errors
	ErrQuestionNotFound = NewQuizError(
		ErrCodeQuestionNotFound,
		"問題が見つかりません",
		http.StatusNotFound,
	)

	ErrInvalidQuestionNumber = NewQuizError(
		ErrCodeInvalidQuestionNum,
		"問題番号が無効です",
		http.StatusBadRequest,
	)

	ErrAlreadyAnswered = NewQuizError(
		ErrCodeAlreadyAnswered,
		"既に回答済みです",
		http.StatusConflict,
	)

	ErrInvalidAnswer = NewQuizError(
		ErrCodeInvalidAnswer,
		"回答が無効です",
		http.StatusBadRequest,
	)

	// State errors
	ErrInvalidStateTransition = NewQuizError(
		ErrCodeInvalidStateTransition,
		"無効な状態遷移です",
		http.StatusBadRequest,
	)

	ErrStateNotAllowed = NewQuizError(
		ErrCodeStateNotAllowed,
		"この状態では操作できません",
		http.StatusBadRequest,
	)

	ErrEventNotStarted = NewQuizError(
		ErrCodeEventNotStarted,
		"イベントが開始されていません",
		http.StatusBadRequest,
	)

	ErrEventAlreadyStarted = NewQuizError(
		ErrCodeEventAlreadyStarted,
		"イベントは既に開始されています",
		http.StatusConflict,
	)

	// Team errors
	ErrTeamNotFound = NewQuizError(
		ErrCodeTeamNotFound,
		"チームが見つかりません",
		http.StatusNotFound,
	)

	ErrTeamFull = NewQuizError(
		ErrCodeTeamFull,
		"チームが満員です",
		http.StatusConflict,
	)

	ErrNotInTeamMode = NewQuizError(
		ErrCodeNotInTeamMode,
		"チーム戦モードではありません",
		http.StatusBadRequest,
	)

	// WebSocket errors
	ErrWebSocketClosed = NewQuizError(
		ErrCodeWebSocketClosed,
		"WebSocket接続が切断されました",
		http.StatusServiceUnavailable,
	)

	ErrConnectionLost = NewQuizError(
		ErrCodeConnectionLost,
		"接続が失われました",
		http.StatusServiceUnavailable,
	)

	ErrBroadcastFailed = NewQuizError(
		ErrCodeBroadcastFailed,
		"メッセージの配信に失敗しました",
		http.StatusInternalServerError,
	)

	// Database errors
	ErrDatabaseError = NewQuizError(
		ErrCodeDatabaseError,
		"データベースエラーが発生しました",
		http.StatusInternalServerError,
	)

	ErrDataNotFound = NewQuizError(
		ErrCodeDataNotFound,
		"データが見つかりません",
		http.StatusNotFound,
	)

	// Validation errors
	ErrValidationFailed = NewQuizError(
		ErrCodeValidationFailed,
		"入力検証に失敗しました",
		http.StatusBadRequest,
	)

	ErrInvalidRequest = NewQuizError(
		ErrCodeInvalidRequest,
		"リクエストが無効です",
		http.StatusBadRequest,
	)

	ErrMissingParameter = NewQuizError(
		ErrCodeMissingParameter,
		"必須パラメータが不足しています",
		http.StatusBadRequest,
	)

	// System errors
	ErrInternalError = NewQuizError(
		ErrCodeInternalError,
		"内部エラーが発生しました",
		http.StatusInternalServerError,
	)

	ErrServiceUnavailable = NewQuizError(
		ErrCodeServiceUnavailable,
		"サービスが利用できません",
		http.StatusServiceUnavailable,
	)

	ErrTimeout = NewQuizError(
		ErrCodeTimeout,
		"処理がタイムアウトしました",
		http.StatusRequestTimeout,
	)
)

// ErrorResponse represents the standardized error response format
type ErrorResponse struct {
	Success   bool      `json:"success"`
	Error     QuizError `json:"error"`
	RequestID string    `json:"request_id,omitempty"`
	Timestamp int64     `json:"timestamp"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(err *QuizError, requestID string) *ErrorResponse {
	return &ErrorResponse{
		Success:   false,
		Error:     *err,
		RequestID: requestID,
		Timestamp: getCurrentTimestamp(),
	}
}

// SuccessResponse represents the standardized success response format
type SuccessResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(data interface{}, message, requestID string) *SuccessResponse {
	return &SuccessResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		RequestID: requestID,
		Timestamp: getCurrentTimestamp(),
	}
}

// Helper functions

// WrapError wraps a standard error into a QuizError
func WrapError(err error, code ErrorCode, message string, httpStatus int) *QuizError {
	return NewQuizError(code, message, httpStatus).WithCause(err)
}

// FromStandardError converts a standard error to a QuizError
func FromStandardError(err error) *QuizError {
	if qErr, ok := err.(*QuizError); ok {
		return qErr
	}

	return NewQuizError(
		ErrCodeInternalError,
		err.Error(),
		http.StatusInternalServerError,
	).WithCause(err)
}

// IsQuizError checks if an error is a QuizError
func IsQuizError(err error) bool {
	_, ok := err.(*QuizError)
	return ok
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	if qErr, ok := err.(*QuizError); ok {
		return qErr.Code
	}
	return ErrCodeInternalError
}

// GetHTTPStatus extracts the HTTP status code from an error
func GetHTTPStatus(err error) int {
	if qErr, ok := err.(*QuizError); ok {
		return qErr.HTTPStatus
	}
	return http.StatusInternalServerError
}

// getCurrentTimestamp returns current Unix timestamp in milliseconds
func getCurrentTimestamp() int64 {
	return getCurrentTime().UnixMilli()
}

// getCurrentTime returns current time (can be mocked for testing)
var getCurrentTime = func() time.Time {
	return time.Now()
}

// Error classification helpers

// IsUserError checks if the error is user-related
func IsUserError(err error) bool {
	if qErr, ok := err.(*QuizError); ok {
		switch qErr.Code {
		case ErrCodeUserNotFound, ErrCodeUserAlreadyJoined,
			ErrCodeSessionExpired, ErrCodeInvalidNickname:
			return true
		}
	}
	return false
}

// IsStateError checks if the error is state-related
func IsStateError(err error) bool {
	if qErr, ok := err.(*QuizError); ok {
		switch qErr.Code {
		case ErrCodeInvalidStateTransition, ErrCodeStateNotAllowed,
			ErrCodeEventNotStarted, ErrCodeEventAlreadyStarted:
			return true
		}
	}
	return false
}

// IsValidationError checks if the error is validation-related
func IsValidationError(err error) bool {
	if qErr, ok := err.(*QuizError); ok {
		switch qErr.Code {
		case ErrCodeValidationFailed, ErrCodeInvalidRequest,
			ErrCodeMissingParameter, ErrCodeInvalidParameter:
			return true
		}
	}
	return false
}

// IsSystemError checks if the error is system-related
func IsSystemError(err error) bool {
	if qErr, ok := err.(*QuizError); ok {
		switch qErr.Code {
		case ErrCodeInternalError, ErrCodeServiceUnavailable,
			ErrCodeTimeout, ErrCodeResourceExhausted, ErrCodeDatabaseError:
			return true
		}
	}
	return false
}

// IsTemporaryError checks if the error is temporary and retryable
func IsTemporaryError(err error) bool {
	if qErr, ok := err.(*QuizError); ok {
		switch qErr.Code {
		case ErrCodeTimeout, ErrCodeServiceUnavailable,
			ErrCodeConnectionLost, ErrCodeWebSocketError:
			return true
		}
	}
	return false
}

// Error factory methods for common scenarios

// NewUserError creates a user-related error
func NewUserError(code ErrorCode, message, details string) *QuizError {
	return NewQuizError(code, message, http.StatusBadRequest).WithDetails(details)
}

// NewStateError creates a state-related error
func NewStateError(code ErrorCode, message, details string) *QuizError {
	return NewQuizError(code, message, http.StatusBadRequest).WithDetails(details)
}

// NewValidationError creates a validation error
func NewValidationError(message, details string) *QuizError {
	return NewQuizError(ErrCodeValidationFailed, message, http.StatusBadRequest).WithDetails(details)
}

// NewDatabaseError creates a database error
func NewDatabaseError(cause error, operation string) *QuizError {
	return NewQuizError(
		ErrCodeDatabaseError,
		fmt.Sprintf("Database operation failed: %s", operation),
		http.StatusInternalServerError,
	).WithCause(cause)
}

// NewWebSocketError creates a WebSocket error
func NewWebSocketError(cause error, message string) *QuizError {
	return NewQuizError(
		ErrCodeWebSocketError,
		message,
		http.StatusInternalServerError,
	).WithCause(cause)
}
