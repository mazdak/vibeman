// Package errors provides typed error definitions for vibeman.
// This package consolidates error handling and provides structured error types
// that can be used for better error classification and handling.
package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents a unique identifier for different error types
type ErrorCode string

const (
	// Configuration errors
	ErrConfigNotFound   ErrorCode = "CONFIG_NOT_FOUND"
	ErrConfigInvalid    ErrorCode = "CONFIG_INVALID"
	ErrConfigParse      ErrorCode = "CONFIG_PARSE"
	ErrConfigValidation ErrorCode = "CONFIG_VALIDATION"
	ErrInvalidConfig    ErrorCode = "INVALID_CONFIG"

	// Container errors
	ErrContainerNotFound     ErrorCode = "CONTAINER_NOT_FOUND"
	ErrContainerCreateFailed ErrorCode = "CONTAINER_CREATE_FAILED"
	ErrContainerStartFailed  ErrorCode = "CONTAINER_START_FAILED"
	ErrContainerStopFailed   ErrorCode = "CONTAINER_STOP_FAILED"
	ErrContainerExecFailed   ErrorCode = "CONTAINER_EXEC_FAILED"
	ErrContainerInvalidID    ErrorCode = "CONTAINER_INVALID_ID"
	ErrContainerCreate       ErrorCode = "CONTAINER_CREATE"
	ErrContainerStart        ErrorCode = "CONTAINER_START"
	ErrContainerNotRunning   ErrorCode = "CONTAINER_NOT_RUNNING"

	// Service errors
	ErrServiceNotFound        ErrorCode = "SERVICE_NOT_FOUND"
	ErrServiceAlreadyRunning  ErrorCode = "SERVICE_ALREADY_RUNNING"
	ErrServiceStartFailed     ErrorCode = "SERVICE_START_FAILED"
	ErrServiceStopFailed      ErrorCode = "SERVICE_STOP_FAILED"
	ErrServiceHealthCheckFail ErrorCode = "SERVICE_HEALTH_CHECK_FAIL"
	ErrServiceDependencyFail  ErrorCode = "SERVICE_DEPENDENCY_FAIL"

	// Git errors
	ErrGitRepoNotFound    ErrorCode = "GIT_REPO_NOT_FOUND"
	ErrGitCloneFailed     ErrorCode = "GIT_CLONE_FAILED"
	ErrGitWorktreeFailed  ErrorCode = "GIT_WORKTREE_FAILED"
	ErrGitBranchNotFound  ErrorCode = "GIT_BRANCH_NOT_FOUND"
	ErrGitUncommitted     ErrorCode = "GIT_UNCOMMITTED_CHANGES"
	ErrGitUnpushed        ErrorCode = "GIT_UNPUSHED_COMMITS"
	ErrGitBranchNotMerged ErrorCode = "GIT_BRANCH_NOT_MERGED"

	// Database errors
	ErrDatabaseConnection ErrorCode = "DATABASE_CONNECTION"
	ErrDatabaseQuery      ErrorCode = "DATABASE_QUERY"
	ErrDatabaseMigration  ErrorCode = "DATABASE_MIGRATION"

	// Network/API errors
	ErrNetworkConnection ErrorCode = "NETWORK_CONNECTION"
	ErrAPICall           ErrorCode = "API_CALL"
	ErrAuthFailed        ErrorCode = "AUTH_FAILED"
	ErrUnauthorized      ErrorCode = "UNAUTHORIZED"
	ErrForbidden         ErrorCode = "FORBIDDEN"
	ErrPermissionDenied  ErrorCode = "PERMISSION_DENIED"
	ErrRateLimited       ErrorCode = "RATE_LIMITED"

	// Validation errors
	ErrValidationFailed ErrorCode = "VALIDATION_FAILED"
	ErrValidation       ErrorCode = "VALIDATION"
	ErrInvalidInput     ErrorCode = "INVALID_INPUT"
	ErrInvalidPath      ErrorCode = "INVALID_PATH"
	ErrInvalidPort      ErrorCode = "INVALID_PORT"
	ErrInvalidState     ErrorCode = "INVALID_STATE"

	// Internal errors
	ErrInternal       ErrorCode = "INTERNAL_ERROR"
	ErrNotImplemented ErrorCode = "NOT_IMPLEMENTED"
	ErrTimeout        ErrorCode = "TIMEOUT"
	ErrCancelled      ErrorCode = "CANCELLED"
	ErrShuttingDown   ErrorCode = "SHUTTING_DOWN"

	// File/IO errors
	ErrFileCreate ErrorCode = "FILE_CREATE"
	ErrFileWrite  ErrorCode = "FILE_WRITE"
	ErrFileRead   ErrorCode = "FILE_READ"
	ErrNotFound   ErrorCode = "NOT_FOUND"
	ErrFileSystem ErrorCode = "FILE_SYSTEM"

	// JSON errors
	ErrJSONMarshal   ErrorCode = "JSON_MARSHAL"
	ErrJSONUnmarshal ErrorCode = "JSON_UNMARSHAL"

	// Cleanup errors
	ErrCleanup ErrorCode = "CLEANUP"

	// AI/Assistant specific errors
	ErrAIAssistantNotReady ErrorCode = "AI_ASSISTANT_NOT_READY"
	ErrComposeFileNotFound ErrorCode = "COMPOSE_FILE_NOT_FOUND"
)

// VibemanError represents a structured error with additional context
type VibemanError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	Context    map[string]interface{} `json:"context,omitempty"`
	HTTPStatus int                    `json:"-"`
}

// Error implements the error interface
func (e *VibemanError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *VibemanError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e *VibemanError) WithContext(key string, value interface{}) *VibemanError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithCause adds the underlying cause error
func (e *VibemanError) WithCause(cause error) *VibemanError {
	e.Cause = cause
	return e
}

// GetHTTPStatus returns the appropriate HTTP status code for this error
func (e *VibemanError) GetHTTPStatus() int {
	if e.HTTPStatus != 0 {
		return e.HTTPStatus
	}

	// Default status codes based on error type
	switch e.Code {
	case ErrConfigNotFound, ErrContainerNotFound, ErrServiceNotFound, ErrGitRepoNotFound, ErrGitBranchNotFound:
		return http.StatusNotFound
	case ErrAuthFailed:
		return http.StatusUnauthorized
	case ErrPermissionDenied:
		return http.StatusForbidden
	case ErrValidationFailed, ErrInvalidInput, ErrInvalidPath, ErrInvalidPort, ErrContainerInvalidID:
		return http.StatusBadRequest
	case ErrServiceAlreadyRunning:
		return http.StatusConflict
	case ErrNotImplemented:
		return http.StatusNotImplemented
	case ErrTimeout:
		return http.StatusRequestTimeout
	default:
		return http.StatusInternalServerError
	}
}

// New creates a new VibemanError
func New(code ErrorCode, message string) *VibemanError {
	return &VibemanError{
		Code:    code,
		Message: message,
	}
}

// NewWithDetails creates a new VibemanError with details
func NewWithDetails(code ErrorCode, message, details string) *VibemanError {
	return &VibemanError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Wrap creates a new VibemanError that wraps an existing error
func Wrap(code ErrorCode, message string, cause error) *VibemanError {
	return &VibemanError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WrapWithDetails creates a new VibemanError with details that wraps an existing error
func WrapWithDetails(code ErrorCode, message, details string, cause error) *VibemanError {
	return &VibemanError{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   cause,
	}
}

// IsVibemanError checks if an error is a VibemanError
func IsVibemanError(err error) bool {
	_, ok := err.(*VibemanError)
	return ok
}

// GetCode extracts the error code from an error, if it's a VibemanError
func GetCode(err error) ErrorCode {
	if ve, ok := err.(*VibemanError); ok {
		return ve.Code
	}
	return ""
}

// HasCode checks if an error has a specific error code
func HasCode(err error, code ErrorCode) bool {
	return GetCode(err) == code
}

// Common pre-defined errors for consistency
var (
	// Worktree errors
	ErrWorktreeNotClean         = New(ErrInvalidState, "worktree has uncommitted changes")
	ErrBranchNotMerged          = New(ErrGitBranchNotMerged, "branch has unmerged commits")
	ErrBranchHasUnpushedCommits = New(ErrGitUnpushed, "branch has unpushed commits")

	// Compose errors
	ErrComposeFileNotFoundError = New(ErrComposeFileNotFound, "docker-compose.yml not found")
	ErrComposeServiceNotFound   = New(ErrServiceNotFound, "compose service not found in docker-compose.yml")

	// AI Assistant errors
	ErrAIAssistantNotReadyError = New(ErrAIAssistantNotReady, "AI assistant failed to initialize")
	ErrAIAssistantTimeout       = New(ErrTimeout, "AI assistant initialization timeout")

	// Container errors
	ErrContainerAlreadyExists   = New(ErrContainerCreateFailed, "container with this name already exists")
	ErrContainerNotRunningError = New(ErrContainerNotRunning, "container is not running")

	// Service errors
	ErrServiceAlreadyRunningError = New(ErrServiceAlreadyRunning, "service is already running")
	ErrServiceNotFoundError       = New(ErrServiceNotFound, "service not found")

	// Validation errors
	ErrEmptyInput       = New(ErrInvalidInput, "input cannot be empty")
	ErrInvalidPathError = New(ErrInvalidPath, "path is invalid or does not exist")
	ErrInvalidPortError = New(ErrInvalidPort, "port number is invalid")
)
