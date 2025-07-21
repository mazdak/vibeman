package server

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"vibeman/internal/compose"
	"vibeman/internal/config"
	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/errors"
	"vibeman/internal/interfaces"
	"vibeman/internal/logger"
	"vibeman/internal/operations"
	"vibeman/internal/types"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// handleError converts errors to appropriate HTTP responses
func handleError(c echo.Context, err error, defaultMessage string) error {
	if ve, ok := err.(*errors.VibemanError); ok {
		return echo.NewHTTPError(ve.GetHTTPStatus(), ve.Error())
	}
	
	// Check for "not found" errors
	errMsg := err.Error()
	if strings.Contains(errMsg, "not found") {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("%s: %v", defaultMessage, err))
	}
	
	return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("%s: %v", defaultMessage, err))
}

// getContainerManagerInterface safely retrieves the container manager as interface
func (s *Server) getContainerManagerInterface() (interfaces.ContainerManager, error) {
	if s.containerMgr == nil {
		return nil, fmt.Errorf("container manager not available")
	}
	
	containerMgr, ok := s.containerMgr.(interfaces.ContainerManager)
	if !ok {
		return nil, fmt.Errorf("invalid container manager type")
	}
	
	return containerMgr, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Swagger documentation
	s.echo.GET("/swagger/*", echoSwagger.WrapHandler)

	// Health check
	s.echo.GET("/health", s.handleHealth)

	// API group
	api := s.echo.Group("/api")

	// Repositories (formerly projects)
	repos := api.Group("/repositories")
	repos.GET("", s.handleListRepositories)
	repos.POST("", s.handleAddRepository)
	repos.DELETE("/:id", s.handleRemoveRepository)

	// Worktrees
	worktrees := api.Group("/worktrees")
	worktrees.GET("", s.handleListWorktrees)
	worktrees.POST("", s.handleCreateWorktree)
	worktrees.GET("/:id", s.handleGetWorktree)
	worktrees.DELETE("/:id", s.handleDeleteWorktree)
	worktrees.POST("/:id/start", s.handleStartWorktree)
	worktrees.POST("/:id/stop", s.handleStopWorktree)
	worktrees.GET("/:id/logs", s.handleGetWorktreeLogs)

	// Services
	services := api.Group("/services")
	services.GET("", s.handleListServices)
	services.POST("/:id/start", s.handleStartService)
	services.POST("/:id/stop", s.handleStopService)
	services.GET("/:id/logs", s.handleGetServiceLogs)

	// System status endpoint
	api.GET("/status", s.handleSystemStatus)

	// Containers
	containers := api.Group("/containers")
	containers.GET("", s.handleListContainers)
	containers.POST("", s.handleCreateContainer)
	containers.GET("/:id", s.handleGetContainer)
	containers.DELETE("/:id", s.handleDeleteContainer)
	containers.POST("/:id/action", s.handleContainerAction)
	containers.GET("/:id/logs", s.handleGetContainerLogs)

	// Configuration endpoint (read-only)
	api.GET("/config", s.handleGetConfig)

	// AI container WebSocket endpoint
	ai := api.Group("/ai")
	ai.GET("/attach/:worktree", s.handleAIWebSocket)
}

// handleHealth godoc
// @Summary Health check
// @Description Check if the API is healthy
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"version": "1.0.0",
	})
}

// handleSystemStatus godoc
// @Summary System status
// @Description Get comprehensive system status including service health and resource counts
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} SystemStatusResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/status [get]
func (s *Server) handleSystemStatus(c echo.Context) error {
	// Calculate uptime
	uptime := time.Since(s.startTime)

	// Check database status
	dbStatus := "unhealthy"
	if dbInstance, err := s.getDB(); err == nil {
		if err := dbInstance.Ping(); err == nil {
			dbStatus = "healthy"
		}
	}

	// Check container engine status
	containerStatus := "unknown"
	if containerMgr, err := s.getContainerManagerInterface(); err == nil {
		// Try to list containers to verify connectivity
		if _, listErr := containerMgr.List(c.Request().Context()); listErr == nil {
			containerStatus = "healthy"
		} else {
			containerStatus = "unhealthy"
		}
	}

	// Check Git status (always healthy if we can execute git commands)
	gitStatus := "healthy"
	if gitMgr, err := s.getGitManager(); err != nil {
		gitStatus = "unhealthy"
		_ = gitMgr // suppress unused variable warning
	}

	// Get resource counts
	var repositoryCount, worktreeCount, containerCount int

	if dbInstance, err := s.getDB(); err == nil {
		// Count repositories
		repoRepo := db.NewRepositoryRepository(dbInstance)
		if repos, err := repoRepo.List(c.Request().Context()); err == nil {
			repositoryCount = len(repos)
		}

		// Count worktrees
		worktreeRepo := db.NewWorktreeRepository(dbInstance)
		if worktrees, err := worktreeRepo.List(c.Request().Context(), "", ""); err == nil {
			worktreeCount = len(worktrees)
		}
	}

	// Count containers
	if containerMgr, err := s.getContainerManagerInterface(); err == nil {
		if containers, listErr := containerMgr.List(c.Request().Context()); listErr == nil {
			containerCount = len(containers)
		}
	}

	// Determine overall status
	overallStatus := "healthy"
	if dbStatus != "healthy" || containerStatus == "unhealthy" || gitStatus != "healthy" {
		overallStatus = "degraded"
	}

	status := SystemStatusResponse{
		Status:       overallStatus,
		Version:      "1.0.0", // TODO: Get from build info
		Uptime:       uptime.String(),
		Services: ServiceHealthStatus{
			Database:        dbStatus,
			ContainerEngine: containerStatus,
			Git:             gitStatus,
		},
		Repositories: repositoryCount,
		Worktrees:    worktreeCount,
		Containers:   containerCount,
	}

	return c.JSON(http.StatusOK, status)
}

// handleListRepositories godoc
// @Summary List repositories
// @Description Get a list of tracked repositories
// @Tags repositories
// @Accept json
// @Produce json
// @Success 200 {object} RepositoriesResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/repositories [get]
func (s *Server) handleListRepositories(c echo.Context) error {
	// Check required dependencies
	dbInstance, err := s.getDB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Database not available",
		})
	}

	gitMgr, err := s.getGitManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Git manager not available",
		})
	}

	// Create operations instance
	ops := operations.NewRepositoryOperations(s.configMgr, gitMgr, dbInstance)

	// List repositories using shared operations
	repositories, err := ops.ListRepositories(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to list repositories: %v", err),
		})
	}

	return c.JSON(http.StatusOK, RepositoriesResponse{
		Repositories: repositories,
		Total:        len(repositories),
	})
}

// handleAddRepository godoc
// @Summary Add a repository
// @Description Add a repository to the tracked list
// @Tags repositories
// @Accept json
// @Produce json
// @Param request body AddRepositoryRequest true "Repository details"
// @Success 201 {object} db.Repository
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/repositories [post]
func (s *Server) handleAddRepository(c echo.Context) error {
	// Check required dependencies
	dbInstance, err := s.getDB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Database not available",
		})
	}

	gitMgr, err := s.getGitManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Git manager not available",
		})
	}

	var req AddRepositoryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request body",
		})
	}

	// Validate path
	if req.Path == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Repository path is required",
		})
	}

	// Create operations instance
	ops := operations.NewRepositoryOperations(s.configMgr, gitMgr, dbInstance)

	// Add repository using shared operations
	repository, err := ops.AddRepository(c.Request().Context(), operations.AddRepositoryRequest{
		Path: req.Path,
		Name: req.Name,
	})

	if err != nil {
		return handleError(c, err, "Failed to add repository")
	}

	return c.JSON(http.StatusCreated, repository)
}

// handleRemoveRepository godoc
// @Summary Remove a repository
// @Description Stop tracking a repository (doesn't delete files)
// @Tags repositories
// @Accept json
// @Produce json
// @Param id path string true "Repository ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/repositories/{id} [delete]
func (s *Server) handleRemoveRepository(c echo.Context) error {
	// Check required dependencies
	dbInstance, err := s.getDB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Database not available",
		})
	}

	gitMgr, err := s.getGitManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Git manager not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Missing repository ID",
		})
	}

	// Create operations instance
	ops := operations.NewRepositoryOperations(s.configMgr, gitMgr, dbInstance)

	// Remove repository using shared operations
	if err := ops.RemoveRepository(c.Request().Context(), id); err != nil {
		return handleError(c, err, "Failed to remove repository")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Repository removed successfully",
	})
}

// handleListWorktrees godoc
// @Summary List worktrees
// @Description Get a list of worktrees with optional filters
// @Tags worktrees
// @Accept json
// @Produce json
// @Security Bearer
// @Param repository_id query string false "Filter by repository ID"
// @Param status query string false "Filter by status"
// @Success 200 {object} WorktreesResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/worktrees [get]
func (s *Server) handleListWorktrees(c echo.Context) error {
	if s.db == nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Database not available",
		})
	}

	// Get filters from query params
	repositoryID := c.QueryParam("repository_id")
	status := c.QueryParam("status")

	dbInstance, err := s.getDB()
	if err != nil {
		return err
	}
	repo := db.NewWorktreeRepository(dbInstance)

	worktrees, err := repo.List(c.Request().Context(), repositoryID, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to list worktrees",
		})
	}

	return c.JSON(http.StatusOK, WorktreesResponse{
		Worktrees: worktrees,
		Total:     len(worktrees),
	})
}

// handleCreateWorktree godoc
// @Summary Create a new worktree
// @Description Create a new development worktree for a repository
// @Tags worktrees
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateWorktreeRequest true "Worktree creation request"
// @Success 201 {object} db.Worktree
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/worktrees [post]
func (s *Server) handleCreateWorktree(c echo.Context) error {
	// Check required dependencies
	dbInstance, err := s.getDB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Database not available",
		})
	}

	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Container manager not available",
		})
	}

	gitMgr, err := s.getGitManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Git manager not available",
		})
	}

	// Parse request
	var req CreateWorktreeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request body",
		})
	}

	// Create operations instance
	serviceMgr, err := s.getServiceManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Service manager not available",
		})
	}
	// Create adapter for container manager
	containerAdapter := &containerManagerAdapter{mgr: containerMgr}
	ops := operations.NewWorktreeOperations(dbInstance, gitMgr, containerAdapter, serviceMgr, s.configMgr)

	// Create worktree using shared operations
	result, err := ops.CreateWorktree(c.Request().Context(), operations.CreateWorktreeRequest{
		RepositoryID:    req.RepositoryID,
		Name:            req.Name,
		Branch:          req.Branch,
		BaseBranch:      req.BaseBranch,
		SkipSetup:       req.SkipSetup,
		ContainerImage:  req.ContainerImage,
		AutoStart:       req.AutoStart,
		ComposeFile:     req.ComposeFile,
		Services: req.ComposeServices,
		PostScripts:     req.PostScripts,
	})

	if err != nil {
		return handleError(c, err, "Failed to create worktree")
	}

	return c.JSON(http.StatusCreated, result.Worktree)
}

// handleGetWorktree godoc
// @Summary Get worktree by ID
// @Description Get a specific worktree by its ID
// @Tags worktrees
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "Worktree ID"
// @Success 200 {object} db.Worktree
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/worktrees/{id} [get]
func (s *Server) handleGetWorktree(c echo.Context) error {
	if s.db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Database not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing worktree ID",
		})
	}

	dbInstance, err := s.getDB()
	if err != nil {
		return err
	}
	repo := db.NewWorktreeRepository(dbInstance)

	worktree, err := repo.Get(c.Request().Context(), id)
	if err != nil {
		if err.Error() == "worktree not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Worktree not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get worktree",
		})
	}

	return c.JSON(http.StatusOK, worktree)
}

func (s *Server) handleDeleteWorktree(c echo.Context) error {
	// Check required dependencies
	dbInstance, err := s.getDB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Database not available",
		})
	}

	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Container manager not available",
		})
	}

	gitMgr, err := s.getGitManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Git manager not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Missing worktree ID",
		})
	}

	// Check for force flag
	force := c.QueryParam("force") == "true"

	// Create operations instance
	serviceMgr, err := s.getServiceManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Service manager not available",
		})
	}
	// Create adapter for container manager
	containerAdapter := &containerManagerAdapter{mgr: containerMgr}
	ops := operations.NewWorktreeOperations(dbInstance, gitMgr, containerAdapter, serviceMgr, s.configMgr)

	// Remove worktree using shared operations
	if err := ops.RemoveWorktree(c.Request().Context(), id, force); err != nil {
		return handleError(c, err, "Failed to delete worktree")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Worktree deleted successfully",
	})
}

// handleStartWorktree godoc
// @Summary Start a worktree
// @Description Start a stopped worktree and its associated container
// @Tags worktrees
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "Worktree ID"
// @Success 200 {object} WorktreeStatusResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/worktrees/{id}/start [post]
func (s *Server) handleStartWorktree(c echo.Context) error {
	// Check required dependencies
	dbInstance, err := s.getDB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Database not available",
		})
	}

	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Container manager not available",
		})
	}

	gitMgr, err := s.getGitManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Git manager not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Missing worktree ID",
		})
	}

	// Create operations instance
	serviceMgr, err := s.getServiceManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Service manager not available",
		})
	}
	// Create adapter for container manager
	containerAdapter := &containerManagerAdapter{mgr: containerMgr}
	ops := operations.NewWorktreeOperations(dbInstance, gitMgr, containerAdapter, serviceMgr, s.configMgr)

	// Start worktree using shared operations
	if err := ops.StartWorktree(c.Request().Context(), id); err != nil {
		return handleError(c, err, "Failed to start worktree")
	}

	return c.JSON(http.StatusOK, WorktreeStatusResponse{
		Message: "Worktree started successfully",
		ID:      id,
		Status:  string(db.StatusRunning),
	})
}

// handleStopWorktree godoc
// @Summary Stop a worktree
// @Description Stop a running worktree and its associated container
// @Tags worktrees
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "Worktree ID"
// @Success 200 {object} WorktreeStatusResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/worktrees/{id}/stop [post]
func (s *Server) handleStopWorktree(c echo.Context) error {
	// Check required dependencies
	dbInstance, err := s.getDB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Database not available",
		})
	}

	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Container manager not available",
		})
	}

	gitMgr, err := s.getGitManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Git manager not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Missing worktree ID",
		})
	}

	// Create operations instance
	serviceMgr, err := s.getServiceManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Service manager not available",
		})
	}
	// Create adapter for container manager
	containerAdapter := &containerManagerAdapter{mgr: containerMgr}
	ops := operations.NewWorktreeOperations(dbInstance, gitMgr, containerAdapter, serviceMgr, s.configMgr)

	// Stop worktree using shared operations
	if err := ops.StopWorktree(c.Request().Context(), id); err != nil {
		return handleError(c, err, "Failed to stop worktree")
	}

	return c.JSON(http.StatusOK, WorktreeStatusResponse{
		Message: "Worktree stopped successfully",
		ID:      id,
		Status:  string(db.StatusStopped),
	})
}

// Service handlers

// handleListServices godoc
// @Summary List services
// @Description Get a list of available services
// @Tags services
// @Accept json
// @Produce json
// @Success 200 {object} ServicesResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/services [get]
func (s *Server) handleListServices(c echo.Context) error {
	// Check required dependencies
	serviceMgr, err := s.getServiceManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Service manager not available",
		})
	}

	// Create operations instance
	ops := operations.NewServiceOperations(s.configMgr, serviceMgr)

	// List services using shared operations
	services, err := ops.ListServices(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to list services: %v", err),
		})
	}
	// Convert operations service info to API response
	apiServices := make([]Service, 0, len(services))
	for _, svc := range services {
		// Determine service type from description or name
		serviceType := "other"
		if svc.Config.Description != "" {
			desc := strings.ToLower(svc.Config.Description)
			if strings.Contains(desc, "database") || strings.Contains(desc, "postgres") || strings.Contains(desc, "mysql") {
				serviceType = "database"
			} else if strings.Contains(desc, "cache") || strings.Contains(desc, "redis") {
				serviceType = "cache"
			} else if strings.Contains(desc, "queue") || strings.Contains(desc, "kafka") || strings.Contains(desc, "rabbitmq") {
				serviceType = "queue"
			}
		} else {
			// Fallback to name-based detection
			if svc.Name == "postgres" || svc.Name == "mysql" {
				serviceType = "database"
			} else if svc.Name == "redis" {
				serviceType = "cache"
			}
		}

		// Parse uptime to get start time
		createdAt := time.Now()
		if svc.StartTime != nil {
			createdAt = *svc.StartTime
		}

		// Extract port from compose file
		port := 0
		if svc.Config.ComposeFile != "" && svc.Config.Service != "" {
			composeFile, err := compose.ParseComposeFile(svc.Config.ComposeFile)
			if err != nil {
				logger.WithError(err).WithFields(logger.Fields{
					"compose_file": svc.Config.ComposeFile,
					"service": svc.Config.Service,
				}).Debug("Failed to parse compose file for port extraction")
			} else {
				baseDir := filepath.Dir(svc.Config.ComposeFile)
				parsedService, err := composeFile.ParseService(svc.Config.Service, baseDir)
				if err != nil {
					logger.WithError(err).WithFields(logger.Fields{
						"compose_file": svc.Config.ComposeFile,
						"service": svc.Config.Service,
					}).Debug("Failed to parse service from compose file")
				} else if len(parsedService.Ports) > 0 {
					// Get the first host port if available
					port = parsedService.Ports[0].HostPort
					logger.WithFields(logger.Fields{
						"service": svc.Name,
						"port": port,
					}).Debug("Extracted port from compose file")
				}
			}
		}

		apiServices = append(apiServices, Service{
			ID:          svc.Name, // Use name as ID
			Name:        svc.Name,
			Type:        serviceType,
			Status:      string(svc.Status),
			Port:        port,
			ContainerID: svc.ContainerID,
			CreatedAt:   createdAt,
		})
	}

	return c.JSON(http.StatusOK, ServicesResponse{
		Services: apiServices,
		Total:    len(apiServices),
	})
}

// handleStartService godoc
// @Summary Start a service
// @Description Start a specific service by ID
// @Tags services
// @Accept json
// @Produce json
// @Param id path string true "Service ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/services/{id}/start [post]
func (s *Server) handleStartService(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Missing service ID",
		})
	}

	// Check required dependencies
	serviceMgr, err := s.getServiceManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Service manager not available",
		})
	}

	// Create operations instance
	ops := operations.NewServiceOperations(s.configMgr, serviceMgr)

	// Start service using shared operations
	if err := ops.StartService(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to start service: %v", err),
		})
	}

	return c.JSON(http.StatusOK, ServiceStatusResponse{
		Message: "Service started successfully",
		ID:      id,
		Status:  "starting",
	})
}

// handleStopService godoc
// @Summary Stop a service
// @Description Stop a specific service by ID
// @Tags services
// @Accept json
// @Produce json
// @Param id path string true "Service ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/services/{id}/stop [post]
func (s *Server) handleStopService(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Missing service ID",
		})
	}

	// Check required dependencies
	serviceMgr, err := s.getServiceManager()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Service manager not available",
		})
	}

	// Create operations instance
	ops := operations.NewServiceOperations(s.configMgr, serviceMgr)

	// Stop service using shared operations
	if err := ops.StopService(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to stop service: %v", err),
		})
	}

	return c.JSON(http.StatusOK, ServiceStatusResponse{
		Message: "Service stopped successfully",
		ID:      id,
		Status:  "stopping",
	})
}

// isValidGitURL validates if the provided URL is a valid Git repository URL
func isValidGitURL(url string) bool {
	// Basic validation for common Git URL patterns
	patterns := []string{
		`^https?://[a-zA-Z0-9.-]+(/.*)?$`,         // HTTP/HTTPS URLs
		`^git@[a-zA-Z0-9.-]+:.*\.git$`,            // SSH URLs like git@github.com:user/repo.git
		`^ssh://[a-zA-Z0-9.-]+(/.*)?$`,            // SSH URLs
		`^[a-zA-Z0-9_-]+@[a-zA-Z0-9.-]+:.*\.git$`, // Generic SSH format
		`^file://.*$`, // Local file URLs
		`^/.*$`,       // Absolute local paths
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, url); matched {
			return true
		}
	}

	return false
}

// Global configuration structure
type GlobalConfig struct {
	Storage struct {
		RepositoriesPath string `json:"repositories_path"`
		WorktreesPath    string `json:"worktrees_path"`
	} `json:"storage"`
	Git struct {
		DefaultBranchPrefix string `json:"default_branch_prefix"`
		AutoFetch           bool   `json:"auto_fetch"`
	} `json:"git"`
	Container struct {
		DefaultRuntime string `json:"default_runtime"`
		AutoStart      bool   `json:"auto_start"`
	} `json:"container"`
}

// handleGetConfig godoc
// @Summary Get global configuration
// @Description Get the global Vibeman configuration
// @Tags config
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} ConfigResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/config [get]
func (s *Server) handleGetConfig(c echo.Context) error {
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to load configuration",
		})
	}

	// Convert to API response format
	response := ConfigResponse{
		Storage: cfg.Storage,
		Git: GitConfig{
			DefaultBranchPrefix: "feature/", // Default value since not in global config
			AutoFetch:           true,       // Default value since not in global config
		},
		Container: ContainerConfig{
			DefaultRuntime: "docker", // Default value since not in global config
			AutoStart:      true,     // Default value since not in global config
		},
	}

	return c.JSON(http.StatusOK, response)
}

// Container handlers

// handleListContainers godoc
// @Summary List containers
// @Description Get a list of all containers
// @Tags containers
// @Accept json
// @Produce json
// @Param repository query string false "Filter by repository"
// @Param status query string false "Filter by status"
// @Success 200 {object} ContainersResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/containers [get]
func (s *Server) handleListContainers(c echo.Context) error {
	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Container manager not available",
		})
	}

	containers, err := containerMgr.List(c.Request().Context())
	if err != nil {
		return handleError(c, err, "Failed to list containers")
	}

	// Apply filters
	repositoryFilter := c.QueryParam("repository")
	statusFilter := c.QueryParam("status")
	
	filteredContainers := containers
	if repositoryFilter != "" || statusFilter != "" {
		filteredContainers = make([]*container.Container, 0)
		for _, cont := range containers {
			include := true
			if repositoryFilter != "" && cont.Repository != repositoryFilter {
				include = false
			}
			if statusFilter != "" && cont.Status != statusFilter {
				include = false
			}
			if include {
				filteredContainers = append(filteredContainers, cont)
			}
		}
	}

	// Convert to API response format
	response := make([]*ContainerResponse, len(filteredContainers))
	for i, container := range filteredContainers {
		// Convert ports map to slice
		var ports []string
		for host, containerPort := range container.Ports {
			ports = append(ports, fmt.Sprintf("%s:%s", host, containerPort))
		}

		response[i] = &ContainerResponse{
			ID:         container.ID,
			Name:       container.Name,
			Image:      container.Image,
			Status:     container.Status,
			State:      container.Status, // Use Status for State since State field doesn't exist
			Ports:      ports,
			Repository: container.Repository,
			Worktree:   container.Environment, // Environment field maps to worktree
			Labels:     make(map[string]string), // Empty labels for now
			CreatedAt:  container.CreatedAt, // Already a string
		}
	}

	return c.JSON(http.StatusOK, ContainersResponse{
		Containers: response,
		Total:      len(response),
	})
}

// handleCreateContainer godoc
// @Summary Create container
// @Description Create a new container
// @Tags containers
// @Accept json
// @Produce json
// @Param container body CreateContainerRequest true "Container configuration"
// @Success 201 {object} ContainerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/containers [post]
func (s *Server) handleCreateContainer(c echo.Context) error {
	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Container manager not available",
		})
	}

	var req CreateContainerRequest
	if err := c.Bind(&req); err != nil {
		return handleError(c, err, "Invalid request format")
	}

	// Validate required fields
	if req.Repository == "" {
		return handleError(c, errors.New(errors.ErrValidationFailed, "repository is required"), "Validation failed")
	}
	if req.Image == "" {
		return handleError(c, errors.New(errors.ErrValidationFailed, "image is required"), "Validation failed")
	}

	// Create container
	environmentName := req.Worktree
	if environmentName == "" {
		environmentName = "default"
	}

	container, err := containerMgr.Create(c.Request().Context(), req.Repository, environmentName, req.Image)
	if err != nil {
		return handleError(c, err, "Failed to create container")
	}

	// Auto-start if requested
	if req.AutoStart {
		if startErr := containerMgr.Start(c.Request().Context(), container.ID); startErr != nil {
			// Log the error but don't fail the creation
			logger.WithError(startErr).Warn("Failed to auto-start container")
		}
	}

	// Convert ports map to slice
	var ports []string
	for host, containerPort := range container.Ports {
		ports = append(ports, fmt.Sprintf("%s:%s", host, containerPort))
	}

	// Convert to API response format
	response := &ContainerResponse{
		ID:         container.ID,
		Name:       container.Name,
		Image:      container.Image,
		Status:     container.Status,
		State:      container.Status, // Use Status for State
		Ports:      ports,
		Repository: container.Repository,
		Worktree:   container.Environment,
		Labels:     make(map[string]string), // Empty labels for now
		CreatedAt:  container.CreatedAt, // Already a string
	}

	return c.JSON(http.StatusCreated, response)
}

// handleGetContainer godoc
// @Summary Get container by ID
// @Description Get a specific container by its ID
// @Tags containers
// @Accept json
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} ContainerResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/containers/{id} [get]
func (s *Server) handleGetContainer(c echo.Context) error {
	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Container manager not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Container ID is required",
		})
	}

	// Get container by name (ID)
	container, err := containerMgr.GetByName(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err, "Failed to get container")
	}

	// Convert ports map to slice
	var ports []string
	for host, containerPort := range container.Ports {
		ports = append(ports, fmt.Sprintf("%s:%s", host, containerPort))
	}

	// Convert to API response format
	response := &ContainerResponse{
		ID:         container.ID,
		Name:       container.Name,
		Image:      container.Image,
		Status:     container.Status,
		State:      container.Status, // Use Status for State
		Ports:      ports,
		Repository: container.Repository,
		Worktree:   container.Environment,
		Labels:     make(map[string]string), // Empty labels for now
		CreatedAt:  container.CreatedAt, // Already a string
	}

	return c.JSON(http.StatusOK, response)
}

// handleDeleteContainer godoc
// @Summary Delete container
// @Description Delete a container by ID
// @Tags containers
// @Accept json
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/containers/{id} [delete]
func (s *Server) handleDeleteContainer(c echo.Context) error {
	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Container manager not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Container ID is required",
		})
	}

	// Stop container first if running
	if err := containerMgr.Stop(c.Request().Context(), id); err != nil {
		// Log warning but continue with removal
		logger.WithError(err).Warn("Failed to stop container before removal")
	}

	// Remove container
	if err := containerMgr.Remove(c.Request().Context(), id); err != nil {
		return handleError(c, err, "Failed to delete container")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Container deleted successfully",
	})
}

// handleContainerAction godoc
// @Summary Perform action on container
// @Description Start, stop, or restart a container
// @Tags containers
// @Accept json
// @Produce json
// @Param id path string true "Container ID"
// @Param action body ContainerActionRequest true "Action to perform"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/containers/{id}/action [post]
func (s *Server) handleContainerAction(c echo.Context) error {
	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Container manager not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Container ID is required",
		})
	}

	var req ContainerActionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request format",
		})
	}

	// Perform the requested action
	switch req.Action {
	case "start":
		if err := containerMgr.Start(c.Request().Context(), id); err != nil {
			return handleError(c, err, "Failed to start container")
		}
	case "stop":
		if err := containerMgr.Stop(c.Request().Context(), id); err != nil {
			return handleError(c, err, "Failed to stop container")
		}
	case "restart":
		// Stop then start
		if err := containerMgr.Stop(c.Request().Context(), id); err != nil {
			logger.WithError(err).Warn("Failed to stop container during restart")
		}
		if err := containerMgr.Start(c.Request().Context(), id); err != nil {
			return handleError(c, err, "Failed to restart container")
		}
	default:
		return handleError(c, errors.New(errors.ErrInvalidInput, "Invalid action. Must be 'start', 'stop', or 'restart'"), "Invalid action")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Container %s %sed successfully", req.Action, req.Action),
	})
}

// handleGetContainerLogs godoc
// @Summary Get container logs
// @Description Get logs from a container
// @Tags containers
// @Accept json
// @Produce json
// @Param id path string true "Container ID"
// @Param follow query bool false "Follow log output"
// @Param tail query int false "Number of lines to show from end of logs"
// @Success 200 {object} ContainerLogsResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/containers/{id}/logs [get]
func (s *Server) handleGetContainerLogs(c echo.Context) error {
	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Container manager not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Container ID is required",
		})
	}

	// Get query parameters
	follow := c.QueryParam("follow") == "true"

	// Get container logs
	logsBytes, err := containerMgr.Logs(c.Request().Context(), id, follow)
	if err != nil {
		return handleError(c, err, "Failed to get container logs")
	}

	// Convert bytes to string and split into lines
	logsStr := string(logsBytes)
	logLines := strings.Split(strings.TrimSpace(logsStr), "\n")

	// Filter out empty lines
	var filteredLogs []string
	for _, line := range logLines {
		if strings.TrimSpace(line) != "" {
			filteredLogs = append(filteredLogs, line)
		}
	}

	response := ContainerLogsResponse{
		Logs:      filteredLogs,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return c.JSON(http.StatusOK, response)
}

// containerManagerAdapter adapts interfaces.ContainerManager to operations.ContainerManager
type containerManagerAdapter struct {
	mgr interfaces.ContainerManager
}

func (a *containerManagerAdapter) Create(ctx context.Context, repositoryName, environment, image string) (*container.Container, error) {
	typesContainer, err := a.mgr.Create(ctx, repositoryName, environment, image)
	if err != nil {
		return nil, err
	}
	return convertTypesContainerToContainer(typesContainer), nil
}

func (a *containerManagerAdapter) CreateWithConfig(ctx context.Context, config *container.CreateConfig) (*container.Container, error) {
	// This is the method that's missing from interfaces.ContainerManager
	// We'll need to use the Create method and build the config ourselves
	typesContainer, err := a.mgr.Create(ctx, config.Repository, config.Environment, config.Image)
	if err != nil {
		return nil, err
	}
	return convertTypesContainerToContainer(typesContainer), nil
}

func (a *containerManagerAdapter) Start(ctx context.Context, containerID string) error {
	return a.mgr.Start(ctx, containerID)
}

func (a *containerManagerAdapter) Stop(ctx context.Context, containerID string) error {
	return a.mgr.Stop(ctx, containerID)
}

func (a *containerManagerAdapter) Remove(ctx context.Context, containerID string) error {
	return a.mgr.Remove(ctx, containerID)
}

func (a *containerManagerAdapter) List(ctx context.Context) ([]*container.Container, error) {
	typesContainers, err := a.mgr.List(ctx)
	if err != nil {
		return nil, err
	}
	
	containers := make([]*container.Container, len(typesContainers))
	for i, tc := range typesContainers {
		containers[i] = convertTypesContainerToContainer(tc)
	}
	return containers, nil
}

func (a *containerManagerAdapter) GetByName(ctx context.Context, name string) (*container.Container, error) {
	typesContainer, err := a.mgr.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return convertTypesContainerToContainer(typesContainer), nil
}

func (a *containerManagerAdapter) Logs(ctx context.Context, containerID string, follow bool) ([]byte, error) {
	return a.mgr.Logs(ctx, containerID, follow)
}

// convertTypesContainerToContainer converts types.Container to container.Container
func convertTypesContainerToContainer(tc *types.Container) *container.Container {
	return &container.Container{
		ID:          tc.ID,
		Name:        tc.Name,
		Image:       tc.Image,
		Status:      tc.Status,
		Repository:  tc.Repository,
		Environment: tc.Environment,
		CreatedAt:   tc.CreatedAt,
		Ports:       tc.Ports,
		Command:     tc.Command,
		EnvVars:     tc.EnvVars,
		Type:        tc.Type,
	}
}
