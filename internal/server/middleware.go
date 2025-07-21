package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"vibeman/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// ContextKeyRequestID is the key for request ID in context
	ContextKeyRequestID contextKey = "request_id"
	// ContextKeyTenantID is the key for tenant ID in context
	ContextKeyTenantID contextKey = "tenant_id"
	// ContextKeyUserID is the key for user ID in context
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyConfig is the key for config manager in context
	ContextKeyConfig contextKey = "config"
)

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return xid.New().String()
}

// requestLogger is a custom logging middleware
func requestLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()

		// Get request ID from context
		reqID := c.Request().Header.Get(echo.HeaderXRequestID)
		if reqID == "" {
			reqID = c.Response().Header().Get(echo.HeaderXRequestID)
		}

		// Log request start
		c.Logger().Infof("Request started: %s %s %s [%s]",
			reqID,
			c.Request().Method,
			c.Request().URL.Path,
			c.RealIP(),
		)

		// Process request
		err := next(c)

		// Log request completion
		latency := time.Since(start)
		status := c.Response().Status

		logLevel := "info"
		if status >= 500 {
			logLevel = "error"
		} else if status >= 400 {
			logLevel = "warn"
		}

		msg := fmt.Sprintf("Request completed: %s %s %s [%s] %d %v",
			reqID,
			c.Request().Method,
			c.Request().URL.Path,
			c.RealIP(),
			status,
			latency,
		)

		switch logLevel {
		case "error":
			c.Logger().Error(msg)
		case "warn":
			c.Logger().Warn(msg)
		default:
			c.Logger().Info(msg)
		}

		return err
	}
}

// contextEnricher adds common values to the request context
func contextEnricher(configMgr *config.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// Add request ID to context
			reqID := c.Request().Header.Get(echo.HeaderXRequestID)
			if reqID != "" {
				ctx = context.WithValue(ctx, ContextKeyRequestID, reqID)
			}

			// Add config manager to context
			ctx = context.WithValue(ctx, ContextKeyConfig, configMgr)

			// Update request with new context
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// AuthMiddleware validates JWT tokens and sets user context
// This is a placeholder that will be implemented when auth is added
func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Pass through - authentication not yet implemented
			// When implemented, this will validate JWT tokens
			return next(c)
		}
	}
}

// TenantMiddleware validates tenant access
// This is a placeholder that will be implemented when multi-tenancy is added
func TenantMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Pass through - multi-tenancy not yet implemented
			// When implemented, this will validate tenant access
			return next(c)
		}
	}
}

// RateLimitMiddleware implements rate limiting per IP/user
// This is a placeholder that can be implemented later
func RateLimitMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Pass through - rate limiting not yet implemented
			// When implemented, this will limit requests per IP/user
			return next(c)
		}
	}
}

// ErrorHandler is a custom error handler for the server
func ErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	message := "Internal server error"

	// Check if it's an echo HTTP error
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if msg, ok := he.Message.(string); ok {
			message = msg
		}
	}

	// Log the error
	reqID := c.Request().Header.Get(echo.HeaderXRequestID)
	c.Logger().Errorf("Request error: %s %s %s [%s] %d %v",
		reqID,
		c.Request().Method,
		c.Request().URL.Path,
		c.RealIP(),
		code,
		err,
	)

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			c.NoContent(code)
		} else {
			c.JSON(code, map[string]interface{}{
				"error":      message,
				"request_id": reqID,
			})
		}
	}
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// GetTenantID retrieves the tenant ID from context
func GetTenantID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKeyTenantID).(string); ok {
		return id
	}
	return ""
}

// GetUserID retrieves the user ID from context
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKeyUserID).(string); ok {
		return id
	}
	return ""
}

// GetConfig retrieves the config manager from context
func GetConfig(ctx context.Context) *config.Manager {
	if cfg, ok := ctx.Value(ContextKeyConfig).(*config.Manager); ok {
		return cfg
	}
	return nil
}
