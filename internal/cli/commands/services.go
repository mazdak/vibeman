package commands

import (
	"context"
	"fmt"

	"vibeman/internal/config"
	"vibeman/internal/logger"
	"vibeman/internal/operations"

	"github.com/spf13/cobra"
)

// ServicesCommands creates plural service management commands
func ServicesCommands(cfg *config.Manager, cm ContainerManager, sm ServiceManager) []*cobra.Command {
	// Create operations instance
	var serviceOps *operations.ServiceOperations
	if sm != nil {
		serviceOps = operations.NewServiceOperations(cfg, sm)
	}
	commands := []*cobra.Command{}

	// vibeman services start
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start all global services",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			return startAllServices(cmd.Context(), serviceOps)
		},
	}
	commands = append(commands, startCmd)

	// vibeman services stop
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop all global services",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			force, _ := cmd.Flags().GetBool("force")
			return stopAllServices(cmd.Context(), serviceOps, force)
		},
	}
	stopCmd.Flags().BoolP("force", "f", false, "Force stop even if services are in use")
	commands = append(commands, stopCmd)

	// vibeman services status
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all global services",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			return allServicesStatusWithOps(cmd.Context(), serviceOps)
		},
	}
	commands = append(commands, statusCmd)

	return commands
}

func startAllServices(ctx context.Context, serviceOps *operations.ServiceOperations) error {
	// Get all services
	services, err := serviceOps.ListServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	if len(services) == 0 {
		logger.Info("No services configured")
		return nil
	}

	logger.Info("Starting all global services...")
	
	// Start each service that's not already running
	started := 0
	failed := 0
	for _, svc := range services {
		if svc.Status == "running" {
			logger.WithFields(logger.Fields{"service": svc.Name}).Info("Already running")
			continue
		}

		logger.WithFields(logger.Fields{"service": svc.Name}).Info("Starting service")
		if err := serviceOps.StartService(ctx, svc.Name); err != nil {
			logger.WithFields(logger.Fields{
				"service": svc.Name,
				"error":   err,
			}).Error("Failed to start service")
			failed++
		} else {
			started++
		}
	}

	if failed > 0 {
		return fmt.Errorf("started %d services, %d failed", started, failed)
	}

	logger.WithFields(logger.Fields{"count": started}).Info("✓ All services started successfully")
	return nil
}

func stopAllServices(ctx context.Context, serviceOps *operations.ServiceOperations, force bool) error {
	// Get all services
	services, err := serviceOps.ListServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	if len(services) == 0 {
		logger.Info("No services configured")
		return nil
	}

	// Check if any services are in use
	if !force {
		inUse := false
		for _, svc := range services {
			if svc.RefCount > 0 {
				logger.WithFields(logger.Fields{
					"service":  svc.Name,
					"refCount": svc.RefCount,
				}).Warn("Service is in use")
				inUse = true
			}
		}
		if inUse {
			return fmt.Errorf("services are in use. Use --force to stop anyway")
		}
	}

	logger.Info("Stopping all global services...")

	// Stop each running service
	stopped := 0
	failed := 0
	for _, svc := range services {
		if svc.Status != "running" {
			logger.WithFields(logger.Fields{"service": svc.Name}).Info("Already stopped")
			continue
		}

		logger.WithFields(logger.Fields{"service": svc.Name}).Info("Stopping service")
		if err := serviceOps.StopService(ctx, svc.Name); err != nil {
			logger.WithFields(logger.Fields{
				"service": svc.Name,
				"error":   err,
			}).Error("Failed to stop service")
			failed++
		} else {
			stopped++
		}
	}

	if failed > 0 {
		return fmt.Errorf("stopped %d services, %d failed", stopped, failed)
	}

	logger.WithFields(logger.Fields{"count": stopped}).Info("✓ All services stopped successfully")
	return nil
}