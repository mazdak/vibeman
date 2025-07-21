package errors

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
)

// HTTPErrorResponse represents the structure of error responses sent to clients
type HTTPErrorResponse struct {
	Error   ErrorInfo              `json:"error"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ErrorInfo contains the core error information
type ErrorInfo struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// ToHTTPError converts a VibemanError to an Echo HTTP error
func ToHTTPError(err error) error {
	if ve, ok := err.(*VibemanError); ok {
		return echo.NewHTTPError(ve.GetHTTPStatus(), HTTPErrorResponse{
			Error: ErrorInfo{
				Code:    ve.Code,
				Message: ve.Message,
				Details: ve.Details,
			},
			Context: ve.Context,
		})
	}

	// For non-VibemanError, create a generic internal error
	return echo.NewHTTPError(http.StatusInternalServerError, HTTPErrorResponse{
		Error: ErrorInfo{
			Code:    ErrInternal,
			Message: "Internal server error",
			Details: err.Error(),
		},
	})
}

// WriteJSONError writes a structured JSON error response
func WriteJSONError(w http.ResponseWriter, err error) error {
	var response HTTPErrorResponse
	var statusCode int

	if ve, ok := err.(*VibemanError); ok {
		response = HTTPErrorResponse{
			Error: ErrorInfo{
				Code:    ve.Code,
				Message: ve.Message,
				Details: ve.Details,
			},
			Context: ve.Context,
		}
		statusCode = ve.GetHTTPStatus()
	} else {
		response = HTTPErrorResponse{
			Error: ErrorInfo{
				Code:    ErrInternal,
				Message: "Internal server error",
				Details: err.Error(),
			},
		}
		statusCode = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(response)
}

// HandleError is a helper function for consistent error handling in HTTP handlers
func HandleError(c echo.Context, err error) error {
	if err == nil {
		return nil
	}

	// Log the error for debugging (in production, use proper logging)
	c.Logger().Error(err)

	return ToHTTPError(err)
}

// BadRequest creates a 400 Bad Request error
func BadRequest(message, details string) error {
	return echo.NewHTTPError(http.StatusBadRequest, HTTPErrorResponse{
		Error: ErrorInfo{
			Code:    ErrInvalidInput,
			Message: message,
			Details: details,
		},
	})
}

// NotFound creates a 404 Not Found error
func NotFound(resource, id string) error {
	return echo.NewHTTPError(http.StatusNotFound, HTTPErrorResponse{
		Error: ErrorInfo{
			Code:    ErrConfigNotFound, // Generic not found code
			Message: "Resource not found",
			Details: resource + " with ID '" + id + "' not found",
		},
	})
}

// Unauthorized creates a 401 Unauthorized error
func Unauthorized(message string) error {
	return echo.NewHTTPError(http.StatusUnauthorized, HTTPErrorResponse{
		Error: ErrorInfo{
			Code:    ErrAuthFailed,
			Message: "Authentication required",
			Details: message,
		},
	})
}

// Forbidden creates a 403 Forbidden error
func Forbidden(message string) error {
	return echo.NewHTTPError(http.StatusForbidden, HTTPErrorResponse{
		Error: ErrorInfo{
			Code:    ErrPermissionDenied,
			Message: "Permission denied",
			Details: message,
		},
	})
}

// Conflict creates a 409 Conflict error
func Conflict(message, details string) error {
	return echo.NewHTTPError(http.StatusConflict, HTTPErrorResponse{
		Error: ErrorInfo{
			Code:    ErrServiceAlreadyRunning, // Generic conflict code
			Message: message,
			Details: details,
		},
	})
}

// InternalServerError creates a 500 Internal Server Error
func InternalServerError(details string) error {
	return echo.NewHTTPError(http.StatusInternalServerError, HTTPErrorResponse{
		Error: ErrorInfo{
			Code:    ErrInternal,
			Message: "Internal server error",
			Details: details,
		},
	})
}
