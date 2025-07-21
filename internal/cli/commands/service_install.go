package commands

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"vibeman/internal/logger"

	"github.com/spf13/cobra"
)

// Embed service files directly
//
//go:embed service_files/systemd/vibeman.service
var vibemanSystemdService string

//go:embed service_files/systemd/vibeman-web.service
var vibemanWebSystemdService string

//go:embed service_files/launchd/com.vibeman.server.plist
var vibemanLaunchdPlist string

//go:embed service_files/launchd/com.vibeman.web.plist
var vibemanWebLaunchdPlist string

type serviceConfig struct {
	Username      string
	HomeDir       string
	VibemanPath   string
	VibemanWebDir string
	BunPath       string
	LogDir        string
}

func ServiceInstallCommands() []*cobra.Command {
	return []*cobra.Command{
		installServiceCommand(),
		uninstallServiceCommand(),
	}
}

func installServiceCommand() *cobra.Command {
	var webDir string
	var user string

	cmd := &cobra.Command{
		Use:   "install-service",
		Short: "Install vibeman as a system service",
		Long: `Install vibeman and vibeman-web as system services that start automatically.
		
On Linux, this installs systemd services.
On macOS, this installs launchd agents.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
				return fmt.Errorf("service installation is only supported on Linux and macOS")
			}

			// Get current user if not specified
			if user == "" {
				user = os.Getenv("USER")
			}

			// Get paths
			vibemanPath, err := exec.LookPath("vibeman")
			if err != nil {
				return fmt.Errorf("vibeman not found in PATH: %w", err)
			}

			bunPath, err := exec.LookPath("bun")
			if err != nil {
				return fmt.Errorf("bun not found in PATH: %w", err)
			}

			homeDir := os.Getenv("HOME")
			if homeDir == "" {
				homeDir = filepath.Join("/home", user)
			}

			// Default web directory if not specified
			if webDir == "" {
				if runtime.GOOS == "darwin" {
					webDir = "/opt/vibeman-web"
				} else {
					webDir = "/opt/vibeman-web"
				}
			}

			cfg := serviceConfig{
				Username:      user,
				HomeDir:       homeDir,
				VibemanPath:   vibemanPath,
				VibemanWebDir: webDir,
				BunPath:       bunPath,
				LogDir:        "/usr/local/var/log/vibeman",
			}

			// Create log directory
			if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
				logger.WithFields(logger.Fields{"error": err}).Warn("Could not create log directory")
			}

			if runtime.GOOS == "linux" {
				return installSystemdServices(cfg)
			} else {
				return installLaunchdServices(cfg)
			}
		},
	}

	cmd.Flags().StringVar(&webDir, "web-dir", "", "Directory where vibeman-web is installed")
	cmd.Flags().StringVar(&user, "user", "", "User to run services as (default: current user)")

	return cmd
}

func uninstallServiceCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall-service",
		Short: "Uninstall vibeman system services",
		Long:  `Remove vibeman and vibeman-web system services.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
				return fmt.Errorf("service uninstallation is only supported on Linux and macOS")
			}

			if runtime.GOOS == "linux" {
				return uninstallSystemdServices()
			} else {
				return uninstallLaunchdServices()
			}
		},
	}
}

func installSystemdServices(cfg serviceConfig) error {
	logger.Info("Installing systemd services...")

	// Process templates
	vibemanContent := strings.ReplaceAll(vibemanSystemdService, "%i", cfg.Username)
	vibemanContent = strings.ReplaceAll(vibemanContent, "/usr/local/bin/vibeman", cfg.VibemanPath)

	webContent := strings.ReplaceAll(vibemanWebSystemdService, "%i", cfg.Username)
	webContent = strings.ReplaceAll(webContent, "/usr/bin/bun", cfg.BunPath)
	webContent = strings.ReplaceAll(webContent, "/opt/vibeman-web", cfg.VibemanWebDir)

	// Write service files
	vibemanServicePath := fmt.Sprintf("/etc/systemd/system/vibeman@%s.service", cfg.Username)
	webServicePath := fmt.Sprintf("/etc/systemd/system/vibeman-web@%s.service", cfg.Username)

	if err := os.WriteFile(vibemanServicePath, []byte(vibemanContent), 0644); err != nil {
		return fmt.Errorf("failed to write vibeman service file: %w", err)
	}

	if err := os.WriteFile(webServicePath, []byte(webContent), 0644); err != nil {
		return fmt.Errorf("failed to write vibeman-web service file: %w", err)
	}

	// Reload systemd and enable services
	commands := [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", fmt.Sprintf("vibeman@%s.service", cfg.Username)},
		{"systemctl", "enable", fmt.Sprintf("vibeman-web@%s.service", cfg.Username)},
		{"systemctl", "start", fmt.Sprintf("vibeman@%s.service", cfg.Username)},
		{"systemctl", "start", fmt.Sprintf("vibeman-web@%s.service", cfg.Username)},
	}

	for _, args := range commands {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %s: %w", strings.Join(args, " "), err)
		}
	}

	logger.Info("Systemd services installed and started successfully!")
	logger.Infof("Check status with: systemctl status vibeman@%s vibeman-web@%s", cfg.Username, cfg.Username)
	return nil
}

func installLaunchdServices(cfg serviceConfig) error {
	logger.Info("Installing launchd services...")

	// Process templates
	vibemanContent := strings.ReplaceAll(vibemanLaunchdPlist, "/Users/USERNAME", cfg.HomeDir)
	vibemanContent = strings.ReplaceAll(vibemanContent, "/usr/local/bin/vibeman", cfg.VibemanPath)

	webContent := strings.ReplaceAll(vibemanWebLaunchdPlist, "/Users/USERNAME", cfg.HomeDir)
	webContent = strings.ReplaceAll(webContent, "/usr/local/bin/bun", cfg.BunPath)
	webContent = strings.ReplaceAll(webContent, "/opt/vibeman-web", cfg.VibemanWebDir)

	// Write plist files
	launchAgentsDir := filepath.Join(cfg.HomeDir, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	vibemanPlistPath := filepath.Join(launchAgentsDir, "com.vibeman.server.plist")
	webPlistPath := filepath.Join(launchAgentsDir, "com.vibeman.web.plist")

	if err := os.WriteFile(vibemanPlistPath, []byte(vibemanContent), 0644); err != nil {
		return fmt.Errorf("failed to write vibeman plist: %w", err)
	}

	if err := os.WriteFile(webPlistPath, []byte(webContent), 0644); err != nil {
		return fmt.Errorf("failed to write vibeman-web plist: %w", err)
	}

	// Load services
	commands := [][]string{
		{"launchctl", "load", "-w", vibemanPlistPath},
		{"launchctl", "load", "-w", webPlistPath},
	}

	for _, args := range commands {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %s: %w", strings.Join(args, " "), err)
		}
	}

	logger.Info("Launchd services installed and started successfully!")
	logger.Info("Check status with: launchctl list | grep vibeman")
	return nil
}

func uninstallSystemdServices() error {
	logger.Info("Uninstalling systemd services...")

	user := os.Getenv("USER")

	commands := [][]string{
		{"systemctl", "stop", fmt.Sprintf("vibeman@%s.service", user)},
		{"systemctl", "stop", fmt.Sprintf("vibeman-web@%s.service", user)},
		{"systemctl", "disable", fmt.Sprintf("vibeman@%s.service", user)},
		{"systemctl", "disable", fmt.Sprintf("vibeman-web@%s.service", user)},
	}

	for _, args := range commands {
		// Ignore errors as service might not exist
		exec.Command(args[0], args[1:]...).Run()
	}

	// Remove service files
	os.Remove(fmt.Sprintf("/etc/systemd/system/vibeman@%s.service", user))
	os.Remove(fmt.Sprintf("/etc/systemd/system/vibeman-web@%s.service", user))

	// Reload systemd
	exec.Command("systemctl", "daemon-reload").Run()

	logger.Info("Systemd services uninstalled successfully!")
	return nil
}

func uninstallLaunchdServices() error {
	logger.Info("Uninstalling launchd services...")

	homeDir := os.Getenv("HOME")
	launchAgentsDir := filepath.Join(homeDir, "Library", "LaunchAgents")

	vibemanPlistPath := filepath.Join(launchAgentsDir, "com.vibeman.server.plist")
	webPlistPath := filepath.Join(launchAgentsDir, "com.vibeman.web.plist")

	// Unload services
	commands := [][]string{
		{"launchctl", "unload", "-w", vibemanPlistPath},
		{"launchctl", "unload", "-w", webPlistPath},
	}

	for _, args := range commands {
		// Ignore errors as service might not exist
		exec.Command(args[0], args[1:]...).Run()
	}

	// Remove plist files
	os.Remove(vibemanPlistPath)
	os.Remove(webPlistPath)

	logger.Info("Launchd services uninstalled successfully!")
	return nil
}
