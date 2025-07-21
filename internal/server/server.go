package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vibeman/internal/config"
	"vibeman/internal/constants"
	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/git"
	"vibeman/internal/interfaces"
	"vibeman/internal/logger"
	"vibeman/internal/service"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Config holds the server configuration
type Config struct {
	// Server settings
	Host            string        `toml:"host"`
	Port            int           `toml:"port"`
	ReadTimeout     time.Duration `toml:"read_timeout"`
	WriteTimeout    time.Duration `toml:"write_timeout"`
	ShutdownTimeout time.Duration `toml:"shutdown_timeout"`

	// CORS settings
	AllowOrigins []string `toml:"allow_origins"`
	AllowHeaders []string `toml:"allow_headers"`

	// Logging
	LogLevel  string `toml:"log_level"`
	LogFormat string `toml:"log_format"`

	// Configuration file path (for compatibility with app.go)
	ConfigPath string `toml:"-"`
}

// DefaultConfig returns the default server configuration
func DefaultConfig() *Config {
	return &Config{
		Host:            "localhost", // Changed from 0.0.0.0 for security
		Port:            constants.DefaultServerPort,
		ReadTimeout:     constants.DefaultServerReadTimeout,
		WriteTimeout:    constants.DefaultServerWriteTimeout,
		ShutdownTimeout: constants.DefaultServerShutdownTimeout,
		AllowOrigins:    []string{"*"},
		AllowHeaders:    []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		LogLevel:        "info",
		LogFormat:       "json",
	}
}

// Server represents the main HTTP server
type Server struct {
	config       *Config
	configMgr    *config.Manager
	echo         *echo.Echo
	containerMgr interfaces.ContainerManager
	gitMgr       interfaces.GitManager
	serviceMgr   interfaces.ServiceManager
	db           *db.DB
	startTime    time.Time
}

// getDB safely retrieves the database instance
func (s *Server) getDB() (*db.DB, error) {
	if s.db == nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "database not initialized")
	}

	return s.db, nil
}

// getServiceManager safely retrieves the service manager with type assertion
func (s *Server) getServiceManager() (*service.Manager, error) {
	if s.serviceMgr == nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "service manager not initialized")
	}

	serviceMgr, ok := s.serviceMgr.(*service.Manager)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "invalid service manager type")
	}

	return serviceMgr, nil
}

// getContainerManager safely retrieves the container manager with type assertion
func (s *Server) getContainerManager() (*container.Manager, error) {
	if s.containerMgr == nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "container manager not initialized")
	}

	containerMgr, ok := s.containerMgr.(*container.Manager)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "invalid container manager type")
	}

	return containerMgr, nil
}

// getGitManager safely retrieves the git manager with type assertion
func (s *Server) getGitManager() (*git.Manager, error) {
	if s.gitMgr == nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "git manager not initialized")
	}

	gitMgr, ok := s.gitMgr.(*git.Manager)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "invalid git manager type")
	}

	return gitMgr, nil
}

// New creates a new server instance with minimal configuration
func New(cfg *Config) *Server {
	return NewWithConfigManager(cfg, nil)
}

// NewWithConfigManager creates a new server instance with full configuration
func NewWithConfigManager(cfg *Config, configMgr *config.Manager) *Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Set logger level if configured
	if cfg.LogLevel != "" {
		logger.SetLevel(cfg.LogLevel)
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Set custom error handler
	e.HTTPErrorHandler = ErrorHandler

	return &Server{
		config:    cfg,
		configMgr: configMgr,
		echo:      e,
		startTime: time.Now(),
	}
}

// NewWithDependencies creates a new server instance with all dependencies
func NewWithDependencies(cfg *Config, containerMgr interfaces.ContainerManager, gitMgr interfaces.GitManager, serviceMgr interfaces.ServiceManager, db *db.DB) *Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Set custom error handler
	e.HTTPErrorHandler = ErrorHandler

	return &Server{
		config:       cfg,
		echo:         e,
		containerMgr: containerMgr,
		gitMgr:       gitMgr,
		serviceMgr:   serviceMgr,
		db:           db,
		startTime:    time.Now(),
	}
}

// Echo returns the Echo instance
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// SetDependencies sets the server dependencies
func (s *Server) SetDependencies(containerMgr interfaces.ContainerManager, gitMgr interfaces.GitManager, serviceMgr interfaces.ServiceManager, db *db.DB) {
	s.containerMgr = containerMgr
	s.gitMgr = gitMgr
	s.serviceMgr = serviceMgr
	s.db = db
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	// Setup middleware and routes if not already done
	s.setupMiddleware()
	s.setupRoutes()
	return s.echo
}

// Start starts the server and blocks until shutdown
func (s *Server) Start(ctx ...context.Context) error {
	var shutdownCtx context.Context
	if len(ctx) > 0 {
		shutdownCtx = ctx[0]
	} else {
		shutdownCtx = context.Background()
	}
	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	// Start server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.echo.Logger.Infof("Starting server on %s", addr)

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.echo,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start server: %w", err)
		}
	}()

	// Wait for interrupt signal, context cancellation, or error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case <-quit:
		s.echo.Logger.Info("Shutting down server...")
	case <-shutdownCtx.Done():
		s.echo.Logger.Info("Context cancelled, shutting down server...")
	}

	// Graceful shutdown
	shutdownTimeout, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownTimeout); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.echo.Logger.Info("Server stopped gracefully")
	return nil
}

// setupMiddleware configures all middleware
func (s *Server) setupMiddleware() {
	// Use our custom request logger instead of echo's default
	s.echo.Use(logger.RequestLogger())

	// Recover middleware
	s.echo.Use(middleware.Recover())

	// CORS middleware
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: s.config.AllowOrigins,
		AllowHeaders: s.config.AllowHeaders,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
	}))

	// Custom middleware
	s.echo.Use(contextEnricher(s.configMgr))
}

// getLogFormat returns the appropriate log format based on configuration
func (s *Server) getLogFormat() string {
	if s.config.LogFormat == "json" {
		return `{"time":"${time_rfc3339}","id":"${id}","remote_ip":"${remote_ip}","method":"${method}","uri":"${uri}","status":${status},"latency_human":"${latency_human}"}` + "\n"
	}
	// Default to text format
	return "${time_rfc3339} ${id} ${remote_ip} ${method} ${uri} ${status} ${latency_human}\n"
}
