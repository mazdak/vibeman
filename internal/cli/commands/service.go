package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"vibeman/internal/api"
	"vibeman/internal/config"
	"vibeman/internal/logger"
	"vibeman/internal/operations"
	"vibeman/internal/types"

	"github.com/spf13/cobra"
)

// ServiceCommands creates service management commands
func ServiceCommands(cfg *config.Manager, cm ContainerManager, sm ServiceManager) []*cobra.Command {
	// Create operations instance
	var serviceOps *operations.ServiceOperations
	if sm != nil {
		serviceOps = operations.NewServiceOperations(cfg, sm)
	}
	commands := []*cobra.Command{}

	// vibeman service list
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List all services",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			return listServicesWithOps(cmd.Context(), serviceOps)
		},
	}
	commands = append(commands, listCmd)

	// vibeman service start <service-name>
	startCmd := &cobra.Command{
		Use:   "start <service-name>",
		Short: "Start a service",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				availableServices := make([]string, 0, len(cfg.Services.Services))
				for name := range cfg.Services.Services {
					availableServices = append(availableServices, name)
				}
				if len(availableServices) == 0 {
					return fmt.Errorf("no services configured. Use 'vibeman service list' to see available services")
				}
				return fmt.Errorf("requires exactly 1 service name. Available services: %s", strings.Join(availableServices, ", "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			return serviceOps.StartService(cmd.Context(), args[0])
		},
	}
	commands = append(commands, startCmd)

	// vibeman service stop <service-name>
	stopCmd := &cobra.Command{
		Use:   "stop <service-name>",
		Short: "Stop a service",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				availableServices := make([]string, 0, len(cfg.Services.Services))
				for name := range cfg.Services.Services {
					availableServices = append(availableServices, name)
				}
				if len(availableServices) == 0 {
					return fmt.Errorf("no services configured. Use 'vibeman service list' to see available services")
				}
				return fmt.Errorf("requires exactly 1 service name. Available services: %s", strings.Join(availableServices, ", "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			return serviceOps.StopService(cmd.Context(), args[0])
		},
	}
	stopCmd.Flags().BoolP("force", "f", false, "Force stop even if repositories are using it")
	commands = append(commands, stopCmd)

	// vibeman service restart <service-name>
	restartCmd := &cobra.Command{
		Use:   "restart <service-name>",
		Short: "Restart a service",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				availableServices := make([]string, 0, len(cfg.Services.Services))
				for name := range cfg.Services.Services {
					availableServices = append(availableServices, name)
				}
				if len(availableServices) == 0 {
					return fmt.Errorf("no services configured. Use 'vibeman service list' to see available services")
				}
				return fmt.Errorf("requires exactly 1 service name. Available services: %s", strings.Join(availableServices, ", "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			return serviceOps.RestartService(cmd.Context(), args[0])
		},
	}
	commands = append(commands, restartCmd)

	// vibeman service status [service-name]
	statusCmd := &cobra.Command{
		Use:   "status [service-name]",
		Short: "Show service status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			if len(args) > 0 {
				return serviceStatusWithOps(cmd.Context(), args[0], serviceOps)
			}
			return allServicesStatusWithOps(cmd.Context(), serviceOps)
		},
	}
	commands = append(commands, statusCmd)

	// vibeman service logs <service-name>
	logsCmd := &cobra.Command{
		Use:   "logs <service-name>",
		Short: "Show service logs",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				availableServices := make([]string, 0, len(cfg.Services.Services))
				for name := range cfg.Services.Services {
					availableServices = append(availableServices, name)
				}
				if len(availableServices) == 0 {
					return fmt.Errorf("no services configured. Use 'vibeman service list' to see available services")
				}
				return fmt.Errorf("requires exactly 1 service name. Available services: %s", strings.Join(availableServices, ", "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			follow, _ := cmd.Flags().GetBool("follow")
			tail, _ := cmd.Flags().GetInt("tail")
			// Logs command still needs direct access for streaming
			return serviceLogs(cmd.Context(), args[0], follow, tail, cfg, sm, cm)
		},
	}
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().IntP("tail", "n", 50, "Number of lines to show from the end of the logs")
	commands = append(commands, logsCmd)

	// vibeman service exec <service-name> <command>
	execCmd := &cobra.Command{
		Use:   "exec <service-name> <command> [args...]",
		Short: "Execute command in service container",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Exec command still needs direct container access
			return serviceExec(cmd.Context(), args[0], args[1:], sm, cm)
		},
	}
	commands = append(commands, execCmd)

	return commands
}

func listServicesWithOps(ctx context.Context, serviceOps *operations.ServiceOperations) error {
	services, err := serviceOps.ListServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	if len(services) == 0 {
		logger.Info("No services configured")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tREF COUNT\tUPTIME\tDESCRIPTION")

	for _, svc := range services {
		uptime := "-"
		if svc.Status == "running" && svc.Uptime != "" {
			uptime = svc.Uptime
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			svc.Name,
			svc.Status,
			svc.RefCount,
			uptime,
			svc.Config.Description,
		)
	}

	w.Flush()
	return nil
}

func serviceStatusWithOps(ctx context.Context, serviceName string, serviceOps *operations.ServiceOperations) error {
	svc, err := serviceOps.GetService(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	logger.WithFields(logger.Fields{
		"service": svc.Name,
		"status": svc.Status,
		"container_id": svc.ContainerID,
		"ref_count": svc.RefCount,
		"repositories": strings.Join(svc.Repositories, ", "),
	}).Info("Service status")

	if svc.Status == "running" && svc.Uptime != "" {
		logger.WithFields(logger.Fields{"uptime": svc.Uptime}).Info("Service uptime")
	}

	if svc.HealthError != "" {
		logger.WithFields(logger.Fields{"health_error": svc.HealthError}).Warn("Service health check failed")
	}

	return nil
}

func allServicesStatusWithOps(ctx context.Context, serviceOps *operations.ServiceOperations) error {
	services, err := serviceOps.ListServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	if len(services) == 0 {
		logger.Info("No services configured")
		return nil
	}

	for _, svc := range services {
		logger.WithFields(logger.Fields{
			"service": svc.Name,
			"status": svc.Status,
			"ref_count": svc.RefCount,
		}).Info("Service")
	}

	return nil
}

func serviceLogs(ctx context.Context, serviceName string, follow bool, tail int, cfg *config.Manager, sm ServiceManager, cm ContainerManager) error {
	// Get service configuration
	serviceConfig, exists := cfg.Services.Services[serviceName]
	if !exists {
		return fmt.Errorf("service '%s' not configured", serviceName)
	}

	// For compose services, use docker compose logs directly
	if serviceConfig.ComposeFile != "" {
		args := []string{
			"compose",
			"-f", serviceConfig.ComposeFile,
			"logs",
		}

		if follow {
			args = append(args, "-f")
		}

		if tail > 0 {
			args = append(args, "--tail", fmt.Sprintf("%d", tail))
		}

		args = append(args, serviceConfig.Service)

		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}

	// For traditional services, use container manager
	svcInterface, err := sm.GetService(serviceName)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}

	// Type assert to access fields
	svc, ok := svcInterface.(*types.ServiceInstance)
	if !ok {
		return fmt.Errorf("unexpected service type")
	}

	if svc.Status != types.ServiceStatusRunning {
		return fmt.Errorf("service is not running")
	}

	// Get logs from container
	logs, err := cm.Logs(ctx, svc.ContainerID, follow)
	if err != nil {
		return err
	}
	fmt.Print(string(logs))
	return nil
}

func serviceExec(ctx context.Context, serviceName string, command []string, sm ServiceManager, cm ContainerManager) error {
	// Get service instance
	svcInterface, err := sm.GetService(serviceName)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}

	// Type assert to access fields
	svc, ok := svcInterface.(*types.ServiceInstance)
	if !ok {
		return fmt.Errorf("unexpected service type")
	}

	if svc.Status != types.ServiceStatusRunning {
		return fmt.Errorf("service is not running")
	}

	// Execute command in container
	output, err := cm.Exec(ctx, svc.ContainerID, command)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	fmt.Print(string(output))
	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, hours)
}

// API-based command functions

func listServicesAPI(ctx context.Context, apiClient *api.APIClient) error {
	// Get services from API
	services, err := apiClient.GetServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}

	if len(services) == 0 {
		logger.Info("No services configured")
		return nil
	}

	// Display in table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tSTATUS\tIMAGE\tPORTS\tPROJECTS\tUPTIME")

	for _, svc := range services {
		// Format status
		status := string(svc.Status)

		// Format uptime
		uptime := "-"
		if svc.Status == "running" && svc.Uptime != "" {
			uptime = svc.Uptime
		}

		// Format repositories count
		repositories := "-"
		if svc.RefCount > 0 {
			repositories = fmt.Sprintf("%d", svc.RefCount)
		}

		// Show compose info
		imageInfo := "-"
		if svc.Config.ComposeFile != "" {
			// Extract filename from compose path for display
			composePath := svc.Config.ComposeFile
			if idx := strings.LastIndex(composePath, "/"); idx != -1 {
				composePath = composePath[idx+1:]
			}
			imageInfo = fmt.Sprintf("%s:%s", composePath, svc.Config.Service)
		}

		// Ports would come from docker-compose inspection
		ports := "-"

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			svc.Name, status, imageInfo, ports, repositories, uptime)
	}

	w.Flush()
	return nil
}

func startServiceAPI(ctx context.Context, serviceName string, apiClient *api.APIClient) error {
	logger.WithField("service", serviceName).Info("Starting service")

	// Start service via API
	if err := apiClient.StartService(ctx, serviceName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Poll for service to be ready
	logger.Info("Waiting for service to be ready...")
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-checkCtx.Done():
			logger.Warn("Service started but health check timed out")
			return nil
		case <-ticker.C:
			// Check service status via API
			service, err := apiClient.GetService(ctx, serviceName)
			if err == nil && service.Status == "running" && service.HealthError == "" {
				logger.WithField("service", serviceName).Info("✓ Service is ready")
				return nil
			}
		}
	}
}

func stopServiceAPI(ctx context.Context, serviceName string, apiClient *api.APIClient) error {
	logger.WithField("service", serviceName).Info("Stopping service")

	// Stop service via API
	if err := apiClient.StopService(ctx, serviceName); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	logger.WithField("service", serviceName).Info("✓ Service stopped")
	return nil
}

func restartServiceAPI(ctx context.Context, serviceName string, apiClient *api.APIClient) error {
	logger.WithField("service", serviceName).Info("Restarting service")

	// Restart service via API
	if err := apiClient.RestartService(ctx, serviceName); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}

	// Poll for service to be ready
	logger.Info("Waiting for service to be ready...")
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-checkCtx.Done():
			logger.Warn("Service restarted but health check timed out")
			return nil
		case <-ticker.C:
			// Check service status via API
			service, err := apiClient.GetService(ctx, serviceName)
			if err == nil && service.Status == "running" && service.HealthError == "" {
				logger.WithField("service", serviceName).Info("✓ Service is ready")
				return nil
			}
		}
	}
}

func serviceStatusAPI(ctx context.Context, serviceName string, apiClient *api.APIClient) error {
	// Get service from API
	service, err := apiClient.GetService(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	logger.WithFields(logger.Fields{
		"service": serviceName,
		"status":  service.Status,
		"compose": fmt.Sprintf("%s:%s", service.Config.ComposeFile, service.Config.Service),
	}).Info("Service status")

	if service.Status == "running" {
		if service.ContainerID != "" {
			logger.WithFields(logger.Fields{
				"container_id": service.ContainerID[:12],
			}).Info("Container details")
		}

		if service.Uptime != "" {
			logger.WithField("uptime", service.Uptime).Info("Uptime")
		}

		if service.HealthError != "" {
			logger.WithFields(logger.Fields{
				"health_status": "unhealthy",
				"error":         service.HealthError,
			}).Warn("Service health")
		} else {
			logger.WithField("health_status", "healthy").Info("Service health")
		}
	}

	// Show repositories using this service
	if len(service.Repositories) > 0 {
		logger.Info("\nUsed by repositories:")
		for _, repository := range service.Repositories {
			logger.WithField("repository", repository).Info("  - Repository")
		}
	}

	return nil
}

func allServicesStatusAPI(ctx context.Context, apiClient *api.APIClient) error {
	// Get all services from API
	services, err := apiClient.GetServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}

	if len(services) == 0 {
		logger.Info("No services configured")
		return nil
	}

	logger.Info("Services Status:")

	for _, svc := range services {
		status := string(svc.Status)
		healthStatus := ""

		if svc.Status == "running" {
			if svc.HealthError != "" {
				healthStatus = " (unhealthy)"
			} else {
				healthStatus = " (healthy)"
			}
		}

		logger.Infof("• %s: %s%s", svc.Name, status, healthStatus)
	}

	return nil
}
