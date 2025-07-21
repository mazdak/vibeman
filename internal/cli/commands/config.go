package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"vibeman/internal/config"
	"vibeman/internal/logger"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

// ConfigCommands creates configuration management commands
func ConfigCommands(cfg *config.Manager) []*cobra.Command {
	commands := []*cobra.Command{}

	// vibeman config init
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			repository, _ := cmd.Flags().GetString("repository")
			template, _ := cmd.Flags().GetString("template")
			return initConfig(cmd.Context(), global, repository, template, cfg)
		},
	}
	initCmd.Flags().BoolP("global", "g", false, "Initialize global configuration")
	initCmd.Flags().StringP("repository", "r", "", "Repository name for repository configuration")
	initCmd.Flags().StringP("template", "t", "", "Use a configuration template")
	commands = append(commands, initCmd)

	// vibeman config validate
	validateCmd := &cobra.Command{
		Use:   "validate [config-file]",
		Short: "Validate configuration file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := ""
			if len(args) > 0 {
				configFile = args[0]
			}
			return validateConfig(cmd.Context(), configFile, cfg)
		},
	}
	commands = append(commands, validateCmd)

	// vibeman config show
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			repository, _ := cmd.Flags().GetBool("repository")
			services, _ := cmd.Flags().GetBool("services")
			return showConfig(cmd.Context(), global, repository, services, cfg)
		},
	}
	showCmd.Flags().BoolP("global", "g", false, "Show global configuration")
	showCmd.Flags().BoolP("repository", "r", false, "Show repository configuration")
	showCmd.Flags().BoolP("services", "s", false, "Show services configuration")
	commands = append(commands, showCmd)

	// vibeman config get <key>
	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getConfigValue(cmd.Context(), args[0], cfg)
		},
	}
	commands = append(commands, getCmd)

	// vibeman config set <key> <value>
	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			return setConfigValue(cmd.Context(), args[0], args[1], global, cfg)
		},
	}
	setCmd.Flags().BoolP("global", "g", false, "Set in global configuration")
	commands = append(commands, setCmd)

	// vibeman config edit
	editCmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit configuration file in editor",
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			services, _ := cmd.Flags().GetBool("services")
			return editConfig(cmd.Context(), global, services, cfg)
		},
	}
	editCmd.Flags().BoolP("global", "g", false, "Edit global configuration")
	editCmd.Flags().BoolP("services", "s", false, "Edit services configuration")
	commands = append(commands, editCmd)

	return commands
}

func initConfig(ctx context.Context, global bool, repositoryName string, template string, cfg *config.Manager) error {
	if global {
		return initGlobalConfig(ctx, cfg)
	}

	if repositoryName != "" {
		return initRepositoryConfig(ctx, repositoryName, template, cfg)
	}

	// Interactive mode
	logger.Info("Configuration Initialization")
	logger.Info("===========================")
	logger.Info("")
	logger.Info("What would you like to initialize?")
	logger.Info("1. Global configuration")
	logger.Info("2. Repository configuration")
	logger.Info("3. Services configuration")
	logger.Info("\nSelect option (1-3): ")

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		return initGlobalConfig(ctx, cfg)
	case "2":
		logger.Info("Enter repository name: ")
		fmt.Scanln(&repositoryName)
		return initRepositoryConfig(ctx, repositoryName, template, cfg)
	case "3":
		return initServicesConfig(ctx, cfg)
	default:
		return fmt.Errorf("invalid option")
	}
}

func initGlobalConfig(ctx context.Context, cfg *config.Manager) error {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "vibeman", "config.toml")

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("global configuration already exists at %s", configPath)
	}

	// Create directory if needed
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Global config is no longer supported
	return fmt.Errorf("global configuration is no longer supported")
}

func initRepositoryConfig(ctx context.Context, repositoryName string, template string, cfg *config.Manager) error {
	configPath := "vibeman.toml"

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("repository configuration already exists at %s", configPath)
	}

	// Create repository config based on template
	repositoryConfig := &config.RepositoryConfig{}
	repositoryConfig.Repository.Name = repositoryName
	repositoryConfig.Repository.Description = fmt.Sprintf("%s development repository", repositoryName)

	switch template {
	case "web", "webapp":
		repositoryConfig.Repository.Container.ComposeFile = "docker-compose.yml"
		repositoryConfig.Repository.Container.Services = []string{"web"}
		repositoryConfig.Repository.Container.Setup = []string{
			"npm install",
			"npm run dev",
		}
		repositoryConfig.Repository.Services = map[string]config.ServiceRequirement{
			"postgres": {Required: true},
			"redis":    {Required: true},
		}

	case "api":
		repositoryConfig.Repository.Container.ComposeFile = "docker-compose.yml"
		repositoryConfig.Repository.Container.Services = []string{"api"}
		repositoryConfig.Repository.Container.Setup = []string{
			"go mod download",
			"go run .",
		}
		repositoryConfig.Repository.Services = map[string]config.ServiceRequirement{
			"postgres": {Required: true},
			"redis":    {Required: true},
		}

	case "ml", "datascience":
		repositoryConfig.Repository.Container.ComposeFile = "docker-compose.yml"
		repositoryConfig.Repository.Container.Services = []string{"jupyter"}
		repositoryConfig.Repository.Container.Setup = []string{
			"pip install -r requirements.txt",
			"jupyter lab --ip=0.0.0.0 --no-browser",
		}

	default:
		// Basic template
		repositoryConfig.Repository.Container.ComposeFile = "docker-compose.yml"
		repositoryConfig.Repository.Container.Services = []string{"app"}
	}

	// Add git configuration if in a git repo
	if _, err := os.Stat(".git"); err == nil {
		// Git configuration has been removed from the simplified config
	}

	// Write config
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	encoder.SetIndentTables(true)
	if err := encoder.Encode(repositoryConfig); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	logger.WithFields(logger.Fields{"repository": repositoryName}).Info("✓ Repository configuration initialized")
	if template != "" {
		logger.WithFields(logger.Fields{"template": template}).Info("  Using template")
	}
	logger.Info("\nNext steps:")
	logger.Info("  1. Edit vibeman.toml to customize settings")
	logger.Infof("  2. Create the repository: vibeman create %s", repositoryName)
	logger.Infof("  3. Start working: vibeman start %s", repositoryName)

	return nil
}

func initServicesConfig(ctx context.Context, cfg *config.Manager) error {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "vibeman", "services.toml")

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("services configuration already exists at %s", configPath)
	}

	// Create directory if needed
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default services config
	servicesConfig := &config.ServicesConfig{
		Services: map[string]config.ServiceConfig{
			"postgres": {
				ComposeFile: "./docker-compose.services.yml",
				Service:     "postgres",
				Description: "PostgreSQL database service",
			},
			"redis": {
				ComposeFile: "./docker-compose.services.yml",
				Service:     "redis",
				Description: "Redis cache service",
			},
			"mysql": {
				ComposeFile: "./docker-compose.services.yml",
				Service:     "mysql",
				Description: "MySQL database service",
			},
		},
	}

	// Write config
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	encoder.SetIndentTables(true)
	if err := encoder.Encode(servicesConfig); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	logger.WithFields(logger.Fields{"path": configPath}).Info("✓ Services configuration initialized")
	logger.Info("\nPre-configured services:")
	logger.Info("  • postgres - PostgreSQL database")
	logger.Info("  • redis - Redis cache")
	logger.Info("  • mysql - MySQL database")
	logger.Info("\nYou can add more services by editing the configuration file.")

	return nil
}

func validateConfig(ctx context.Context, configFile string, cfg *config.Manager) error {
	// If no file specified, validate all loaded configs
	if configFile == "" {
		logger.Info("Validating loaded configurations...")

		// Global config no longer used

		// Validate repository config
		if cfg.Repository != nil {
			logger.Info("Repository configuration: ")
			if err := validateRepositoryConfig(cfg.Repository); err != nil {
				logger.WithFields(logger.Fields{"error": err}).Error("✗ Invalid repository configuration")
			} else {
				logger.Info("✓ Valid")
			}
		}

		// Validate services config
		if cfg.Services != nil {
			logger.Info("Services configuration: ")
			if err := validateServicesConfig(cfg.Services); err != nil {
				logger.WithFields(logger.Fields{"error": err}).Error("✗ Invalid services configuration")
			} else {
				logger.Info("✓ Valid")
			}
		}

		return nil
	}

	// Validate specific file
	logger.WithFields(logger.Fields{"file": configFile}).Info("Validating configuration")

	// Read file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Try to parse as different config types
	var parseErr error

	// Global config no longer supported

	// Try repository config
	var repositoryConfig config.RepositoryConfig
	if err := toml.Unmarshal(data, &repositoryConfig); err == nil {
		if err := validateRepositoryConfig(&repositoryConfig); err != nil {
			return fmt.Errorf("repository config validation failed: %w", err)
		}
		logger.Info("✓ Valid repository configuration")
		return nil
	}

	// Try services config
	var servicesConfig config.ServicesConfig
	if err := toml.Unmarshal(data, &servicesConfig); err == nil {
		if err := validateServicesConfig(&servicesConfig); err != nil {
			return fmt.Errorf("services config validation failed: %w", err)
		}
		logger.Info("✓ Valid services configuration")
		return nil
	}

	return fmt.Errorf("failed to parse config file: %w", parseErr)
}

func validateRepositoryConfig(cfg *config.RepositoryConfig) error {
	if cfg.Repository.Name == "" {
		return fmt.Errorf("repository.name is required")
	}
	if cfg.Repository.Container.ComposeFile == "" {
		return fmt.Errorf("container.compose_file is required")
	}
	// Services can be empty (which means all services)
	return nil
}

func validateServicesConfig(cfg *config.ServicesConfig) error {
	if len(cfg.Services) == 0 {
		return fmt.Errorf("at least one service must be defined")
	}
	for name, svc := range cfg.Services {
		if svc.ComposeFile == "" {
			return fmt.Errorf("service %s: compose_file is required", name)
		}
		if svc.Service == "" {
			return fmt.Errorf("service %s: service name is required", name)
		}
	}
	return nil
}

func showConfig(ctx context.Context, global, repository, services bool, cfg *config.Manager) error {
	// If no flags specified, show all
	if !global && !repository && !services {
		global = true
		repository = true
		services = true
	}

	if global {
		logger.Info("=== Global Configuration ===")
		logger.Info("Global configuration is no longer supported")
	}

	if repository && cfg.Repository != nil {
		logger.Info("=== Repository Configuration ===")
		data, _ := toml.Marshal(cfg.Repository)
		logger.Info(string(data))
	}

	if services && cfg.Services != nil {
		logger.Info("=== Services Configuration ===")
		data, _ := toml.Marshal(cfg.Services)
		logger.Info(string(data))
	}

	return nil
}

func getConfigValue(ctx context.Context, key string, cfg *config.Manager) error {
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid key format. Use format: section.key or section.subsection.key")
	}

	// Determine which config to search
	switch parts[0] {
	case "global":
		return fmt.Errorf("global configuration is no longer supported")

	case "repository":
		if cfg.Repository == nil {
			return fmt.Errorf("repository configuration not loaded")
		}
		if len(parts) == 2 && parts[1] == "name" {
			logger.Info(cfg.Repository.Repository.Name)
			return nil
		}

	case "services":
		if cfg.Services == nil {
			return fmt.Errorf("services configuration not loaded")
		}
		if len(parts) >= 3 {
			serviceName := parts[1]
			if svc, exists := cfg.Services.Services[serviceName]; exists {
				switch parts[2] {
				case "compose_file":
					logger.Info(svc.ComposeFile)
					return nil
				case "service":
					logger.Info(svc.Service)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("key not found: %s", key)
}

func setConfigValue(ctx context.Context, key, value string, global bool, cfg *config.Manager) error {
	// Determine which config file to modify
	var configPath string
	if global {
		configPath = filepath.Join(os.Getenv("HOME"), ".config", "vibeman", "config.toml")
	} else {
		configPath = "vibeman.toml"
	}

	// Check if file exists
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("configuration file not found: %s", configPath)
	}

	// Read the raw TOML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse into a generic map for manipulation
	var configMap map[string]interface{}
	if err := toml.Unmarshal(data, &configMap); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Parse the key path (e.g., "repository.name" or "services.postgres.image")
	parts := strings.Split(key, ".")
	if len(parts) == 0 {
		return fmt.Errorf("invalid key: %s", key)
	}

	// Navigate to the target field
	current := configMap
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			// Create the path if it doesn't exist
			next := make(map[string]interface{})
			current[part] = next
			current = next
		}
	}

	// Set the value
	lastKey := parts[len(parts)-1]

	// Try to parse the value as appropriate type
	switch strings.ToLower(value) {
	case "true":
		current[lastKey] = true
	case "false":
		current[lastKey] = false
	default:
		// Try to parse as int
		if intVal, err := strconv.Atoi(value); err == nil {
			current[lastKey] = intVal
		} else {
			// Default to string
			current[lastKey] = value
		}
	}

	// Marshal back to TOML
	output, err := toml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.WithFields(logger.Fields{"key": key, "value": value, "path": configPath}).Info("Configuration value set")
	return nil
}

func editConfig(ctx context.Context, global, services bool, cfg *config.Manager) error {
	var configPath string

	if global {
		configPath = filepath.Join(os.Getenv("HOME"), ".config", "vibeman", "config.toml")
	} else if services {
		configPath = filepath.Join(os.Getenv("HOME"), ".config", "vibeman", "services.toml")
	} else {
		configPath = "vibeman.toml"
	}

	// Check if file exists
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("configuration file not found: %s", configPath)
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Open editor
	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	logger.WithFields(logger.Fields{"path": configPath}).Info("✓ Configuration saved")
	return nil
}
