package logger

import (
	"context"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

// Logger is the global logger instance
var Logger *logrus.Logger

// Fields is an alias for logrus.Fields
type Fields = logrus.Fields

// init initializes the global logger
func init() {
	Logger = logrus.New()
	Logger.SetOutput(os.Stdout)
	Logger.SetLevel(logrus.InfoLevel)

	// Use JSON formatter in production
	if os.Getenv("VIBEMAN_ENV") == "production" {
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		// Use text formatter for development
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}
}

// SetLevel sets the logging level
func SetLevel(level string) {
	switch level {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "info":
		Logger.SetLevel(logrus.InfoLevel)
	case "warn":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}
}

// WithContext returns a logger with context fields
func WithContext(ctx context.Context) *logrus.Entry {
	// Extract request ID from context if available
	if reqID, ok := ctx.Value("request_id").(string); ok {
		return Logger.WithField("request_id", reqID)
	}
	return Logger.WithContext(ctx)
}

// WithFields returns a logger with additional fields
func WithFields(fields Fields) *logrus.Entry {
	return Logger.WithFields(fields)
}

// Info logs an info message
func Info(msg string) {
	Logger.Info(msg)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	Logger.Infof(format, args...)
}

// Debug logs a debug message
func Debug(msg string) {
	Logger.Debug(msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	Logger.Debugf(format, args...)
}

// Warn logs a warning message
func Warn(msg string) {
	Logger.Warn(msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	Logger.Warnf(format, args...)
}

// Error logs an error message
func Error(msg string) {
	Logger.Error(msg)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	Logger.Errorf(format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	Logger.Fatal(msg)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	Logger.Fatalf(format, args...)
}

// RequestLogger returns a middleware for logging HTTP requests
func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Generate request ID
			reqID := xid.New().String()
			c.Set("request_id", reqID)

			// Create request logger
			reqLogger := Logger.WithFields(Fields{
				"request_id": reqID,
				"method":     c.Request().Method,
				"path":       c.Request().URL.Path,
				"ip":         c.RealIP(),
				"user_agent": c.Request().UserAgent(),
			})

			// Set logger in context
			c.Set("logger", reqLogger)

			// Process request
			err := next(c)

			// Log response
			latency := time.Since(start)
			status := c.Response().Status

			fields := Fields{
				"status":     status,
				"latency_ms": latency.Milliseconds(),
				"latency":    latency.String(),
			}

			if err != nil {
				fields["error"] = err.Error()
				c.Error(err)
			}

			// Choose log level based on status code
			entry := reqLogger.WithFields(fields)

			switch {
			case status >= 500:
				entry.Error("Request failed")
			case status >= 400:
				entry.Warn("Request error")
			case status >= 300:
				entry.Info("Request redirected")
			default:
				entry.Info("Request completed")
			}

			return err
		}
	}
}

// GetLogger extracts logger from echo context
func GetLogger(c echo.Context) *logrus.Entry {
	if logger, ok := c.Get("logger").(*logrus.Entry); ok {
		return logger
	}
	// Fallback to logger with request ID
	if reqID, ok := c.Get("request_id").(string); ok {
		return Logger.WithField("request_id", reqID)
	}
	return Logger.WithFields(Fields{})
}

// WithError adds an error field to the logger
func WithError(err error) *logrus.Entry {
	return Logger.WithError(err)
}

// WithField adds a field to the logger
func WithField(key string, value interface{}) *logrus.Entry {
	return Logger.WithField(key, value)
}
