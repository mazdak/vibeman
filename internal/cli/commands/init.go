package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vibeman/internal/config"
	"vibeman/internal/logger"

	"github.com/spf13/cobra"
)

// InitCommands creates the init command for project setup
func InitCommands(cfg *config.Manager, cm ContainerManager, gm GitManager, sm ServiceManager) []*cobra.Command {
	commands := []*cobra.Command{}

	// vibeman init
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize vibeman configuration",
		Long: `Initialize vibeman configuration.

Without arguments: Creates global configuration in ~/.config/vibeman/config.toml
With --project flag: Creates vibeman.toml in current directory for project configuration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, _ := cmd.Flags().GetBool("project")
			if project {
				// Repository initialization
				interactive, _ := cmd.Flags().GetBool("interactive")
				force, _ := cmd.Flags().GetBool("force")
				name, _ := cmd.Flags().GetString("name")
				image, _ := cmd.Flags().GetString("image")
				return initProject(cmd, interactive, force, name, image, cfg, cm, gm, sm)
			} else {
				// Global config initialization
				force, _ := cmd.Flags().GetBool("force")
				return initGlobalConfigInteractive(force)
			}
		},
	}
	initCmd.Flags().Bool("project", false, "Initialize project configuration instead of global")
	initCmd.Flags().BoolP("interactive", "i", true, "Run in interactive mode")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing configuration")
	initCmd.Flags().StringP("name", "n", "", "Repository name (defaults to directory name)")
	initCmd.Flags().StringP("image", "I", "ubuntu:22.04", "Default container image")
	commands = append(commands, initCmd)

	return commands
}

// promptYesNo asks a yes/no question and returns true for yes
func promptYesNo(question string, defaultNo bool) bool {
	defaultStr := "y/N"
	if !defaultNo {
		defaultStr = "Y/n"
	}
	logger.Infof("%s (%s) ", question, defaultStr)

	answer := readLine()
	if answer == "" {
		return !defaultNo
	}

	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

// promptString asks for a string value with an optional default
func promptString(prompt string, defaultValue string) string {
	if defaultValue != "" {
		logger.Infof("%s [%s]: ", prompt, defaultValue)
	} else {
		logger.Infof("%s: ", prompt)
	}

	value := readLine()
	if value == "" && defaultValue != "" {
		return defaultValue
	}
	return value
}

// promptMultiSelect presents multiple options and allows selecting multiple items
func promptMultiSelect(prompt string, options []string, selected map[string]bool) map[string]bool {
	logger.Infof("%s:", prompt)
	logger.Info("(Use space-separated list, e.g., 'postgres redis', or 'none' for no services)")

	// Display options
	for _, opt := range options {
		if opt != "none" {
			logger.Infof("  • %s", opt)
		}
	}

	fmt.Print("Your selection: ")
	input := readLine()

	result := make(map[string]bool)
	input = strings.ToLower(strings.TrimSpace(input))

	// Handle "none" or empty input
	if input == "none" || input == "" {
		return result
	}

	// Parse selections
	selections := strings.Fields(input)
	for _, sel := range selections {
		sel = strings.ToLower(strings.TrimSpace(sel))
		for _, opt := range options {
			if strings.ToLower(opt) == sel {
				result[opt] = true
				break
			}
		}
	}

	return result
}

// initGlobalConfigInteractive creates the global configuration file interactively
func initGlobalConfigInteractive(force bool) error {
	logger.Info("Initializing global configuration")

	// Get config directory
	configDir, err := config.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.toml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf("global config already exists at %s. Use --force to overwrite", configPath)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Interactive prompts
	reader := bufio.NewReader(os.Stdin)

	// Server port
	fmt.Print("Server port [8080]: ")
	serverPortStr, _ := reader.ReadString('\n')
	serverPortStr = strings.TrimSpace(serverPortStr)
	serverPort := 8080
	if serverPortStr != "" {
		fmt.Sscanf(serverPortStr, "%d", &serverPort)
	}

	// Web UI port
	fmt.Print("Web UI port [8081]: ")
	webUIPortStr, _ := reader.ReadString('\n')
	webUIPortStr = strings.TrimSpace(webUIPortStr)
	webUIPort := 8081
	if webUIPortStr != "" {
		fmt.Sscanf(webUIPortStr, "%d", &webUIPort)
	}

	// Repositories path
	defaultReposPath := filepath.Join(os.Getenv("HOME"), "vibeman", "repos")
	fmt.Printf("Repositories path [%s]: ", defaultReposPath)
	reposPath, _ := reader.ReadString('\n')
	reposPath = strings.TrimSpace(reposPath)
	if reposPath == "" {
		reposPath = defaultReposPath
	}

	// Worktrees path
	defaultWorktreesPath := filepath.Join(os.Getenv("HOME"), "vibeman", "worktrees")
	fmt.Printf("Worktrees path [%s]: ", defaultWorktreesPath)
	worktreesPath, _ := reader.ReadString('\n')
	worktreesPath = strings.TrimSpace(worktreesPath)
	if worktreesPath == "" {
		worktreesPath = defaultWorktreesPath
	}

	// Services config path
	defaultServicesPath := filepath.Join(configDir, "services.toml")
	fmt.Printf("Services config path [%s]: ", defaultServicesPath)
	servicesPath, _ := reader.ReadString('\n')
	servicesPath = strings.TrimSpace(servicesPath)
	if servicesPath == "" {
		servicesPath = defaultServicesPath
	}

	// Create config
	globalConfig := &config.GlobalConfig{
		Server: config.ServerConfig{
			Port:      serverPort,
			WebUIPort: webUIPort,
		},
		Storage: config.StorageConfig{
			RepositoriesPath: reposPath,
			WorktreesPath:    worktreesPath,
		},
		Services: config.GlobalServicesConfig{
			ConfigPath: servicesPath,
		},
	}

	// Save config
	if err := config.SaveGlobalConfig(globalConfig); err != nil {
		return fmt.Errorf("failed to save global config: %w", err)
	}

	logger.WithFields(logger.Fields{"path": configPath}).Info("✓ Global configuration created")

	// Create services.toml if it doesn't exist
	if _, err := os.Stat(servicesPath); os.IsNotExist(err) {
		servicesContent := `# Vibeman Services Configuration
# Services are defined by referencing docker-compose files

[services]

# Example: PostgreSQL service from a docker-compose file
# [services.postgres]
# compose_file = "/path/to/docker-compose.yaml"
# service = "postgres"
# description = "PostgreSQL database for development"
`
		if err := os.WriteFile(servicesPath, []byte(servicesContent), 0644); err != nil {
			logger.WithFields(logger.Fields{"error": err}).Warn("Failed to create services.toml")
		} else {
			logger.WithFields(logger.Fields{"path": servicesPath}).Info("✓ Services configuration created")
		}
	}

	return nil
}

func initProject(cmd *cobra.Command, interactive, force bool, name, image string, cfg *config.Manager, cm ContainerManager, gm GitManager, sm ServiceManager) error {
	// Check if we're in a directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if vibeman.toml already exists
	configPath := filepath.Join(currentDir, "vibeman.toml")
	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf("vibeman.toml already exists in current directory. Use --force to overwrite")
	}

	// Determine repository name
	if name == "" {
		name = filepath.Base(currentDir)
	}

	// Validate repository name
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("invalid repository name. Please specify a valid name with --name")
	}

	// Sanitize repository name for container usage
	name = sanitizeRepositoryName(name)

	logger.WithFields(logger.Fields{"project": name}).Info("Initializing repository")
	logger.WithFields(logger.Fields{"directory": currentDir}).Info("Repository directory")

	// Detect git repository
	var gitRepoURL, gitDefaultBranch string
	if gm.IsRepository(currentDir) {
		logger.Info("✓ Git repository detected")

		// Try to get remote URL
		// This is a simplified approach - in a real implementation, you'd want to
		// call git commands to get the remote URL
		gitRepoURL = "" // Would be populated by git remote get-url origin
		gitDefaultBranch = "main"

		// Try to detect default branch
		if defaultBranch, err := gm.GetDefaultBranch(cmd.Context(), currentDir); err == nil {
			gitDefaultBranch = defaultBranch
		}
	} else {
		logger.Info("• No git repository detected")
	}

	// Interactive configuration
	var composeFile string
	var composeServices []string
	var selectedServices map[string]string
	var customSetupCommands []string

	// Determine if we should run the full wizard
	// Skip wizard if --name is explicitly provided via command line
	nameProvidedViaFlag := cmd.Flag("name").Changed
	runWizard := interactive && !nameProvidedViaFlag

	if runWizard {
		logger.Info("\n=== Repository Setup Wizard ===")

		// Repository name
		name = promptString("Repository name", name)
		name = sanitizeRepositoryName(name)

		// Docker Compose configuration
		composeFile = promptString("Docker Compose file path", "docker-compose.yml")
		// Prompt for services (comma-separated)
		servicesStr := promptString("Services to start from compose file (comma-separated, leave empty for all)", "")
		if servicesStr != "" {
			composeServices = strings.Split(servicesStr, ",")
			for i := range composeServices {
				composeServices[i] = strings.TrimSpace(composeServices[i])
			}
		}

		// Validate compose file exists
		if composeFile != "" {
			// Handle both absolute and relative paths
			absPath := composeFile
			if !filepath.IsAbs(composeFile) {
				absPath = filepath.Join(currentDir, composeFile)
			}
			if _, err := os.Stat(absPath); err != nil {
				logger.WithFields(logger.Fields{"file": composeFile}).Warn("Docker Compose file not found")
				logger.Info("You can create it later or it will be generated with default settings.")
			}
		}

		// Services
		serviceOptions := []string{"postgres", "redis", "mysql", "mongodb", "none"}
		selectedServicesMap := promptMultiSelect("Select services you need", serviceOptions, nil)

		// Convert selected services to map with default images
		selectedServices = make(map[string]string)
		if selectedServicesMap["postgres"] {
			selectedServices["postgres"] = "postgres:15"
		}
		if selectedServicesMap["redis"] {
			selectedServices["redis"] = "redis:7-alpine"
		}
		if selectedServicesMap["mysql"] {
			selectedServices["mysql"] = "mysql:8"
		}
		if selectedServicesMap["mongodb"] {
			selectedServices["mongodb"] = "mongo:6"
		}

		// Custom setup commands
		if promptYesNo("Do you want to add custom setup commands?", true) {
			logger.Info("Enter setup commands (one per line, empty line to finish):")
			reader := bufio.NewReader(os.Stdin)
			for {
				cmd, _ := reader.ReadString('\n')
				cmd = strings.TrimSpace(cmd)
				if cmd == "" {
					break
				}
				customSetupCommands = append(customSetupCommands, cmd)
			}
		}

		// Git configuration
		if gitRepoURL == "" && gm.IsRepository(currentDir) {
			gitRepoURL = promptString("Git repository URL (leave empty to skip)", "")
		}

		if gitRepoURL != "" {
			gitDefaultBranch = promptString("Default branch", gitDefaultBranch)
		}
	} else if interactive {
		// Non-wizard interactive mode (when --name is explicitly provided via CLI)
		logger.Info("\n=== Quick Repository Setup ===")
		logger.WithFields(logger.Fields{"project": name}).Info("Creating project with default settings")
	}

	// Set default compose values if not set
	if composeFile == "" {
		composeFile = "docker-compose.yml"
	}
	// composeServices can be empty (which means all services)

	// Create project configuration
	projectConfig := &config.RepositoryConfig{}
	projectConfig.Repository.Name = name
	projectConfig.Repository.Description = fmt.Sprintf("Development project for %s", name)
	projectConfig.Repository.Container.ComposeFile = composeFile
	projectConfig.Repository.Container.Services = composeServices
	projectConfig.Repository.Container.Setup = customSetupCommands
	projectConfig.Repository.Git.RepoURL = gitRepoURL
	projectConfig.Repository.Git.DefaultBranch = gitDefaultBranch
	projectConfig.Repository.Worktrees.Directory = "../worktrees"              // Default relative path
	projectConfig.Repository.Services = map[string]config.ServiceRequirement{} // Empty by default

	// Write configuration to file
	if err := writeConfigFile(configPath, projectConfig, selectedServices); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	logger.Info("\n✓ Repository initialized successfully!")
	logger.WithFields(logger.Fields{
		"config_file":     configPath,
		"project_name":    name,
		"compose_file":    composeFile,
		"services": strings.Join(composeServices, ", "),
	}).Info("Repository configuration")

	if gitRepoURL != "" {
		logger.WithFields(logger.Fields{
			"git_repository": gitRepoURL,
			"default_branch": gitDefaultBranch,
		}).Info("Git configuration")
	}

	logger.Info("\nNext steps:")
	logger.Info("  1. Review and customize vibeman.toml as needed")
	logger.Info("  2. Run 'vibeman env create main' to create your first environment")
	logger.Info("  3. Run 'vibeman start' to start your development environment")

	return nil
}

// sanitizeRepositoryName ensures the repository name is safe for use as container names
func sanitizeRepositoryName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace invalid characters with hyphens
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, name)

	// Remove consecutive hyphens
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Trim hyphens from start and end
	name = strings.Trim(name, "-")

	return name
}

// readLine reads a line from stdin
func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// writeConfigFile writes the configuration to a TOML file
func writeConfigFile(path string, config *config.RepositoryConfig, services map[string]string) error {
	var content strings.Builder

	// Repository section
	content.WriteString("[repository]\n")
	content.WriteString(fmt.Sprintf("name = \"%s\"\n", config.Repository.Name))
	content.WriteString(fmt.Sprintf("description = \"%s\"\n", config.Repository.Description))

	// Container section
	content.WriteString("\n[repository.container]\n")
	if config.Repository.Container.ComposeFile != "" {
		content.WriteString(fmt.Sprintf("compose_file = \"%s\"\n", config.Repository.Container.ComposeFile))
	}
	if len(config.Repository.Container.Services) > 0 {
		content.WriteString("services = [")
		for i, service := range config.Repository.Container.Services {
			if i > 0 {
				content.WriteString(", ")
			}
			content.WriteString(fmt.Sprintf("\"%s\"", service))
		}
		content.WriteString("]\n")
	}

	if len(config.Repository.Container.Setup) > 0 {
		content.WriteString("setup = [\n")
		for _, cmd := range config.Repository.Container.Setup {
			content.WriteString(fmt.Sprintf("  \"%s\",\n", cmd))
		}
		content.WriteString("]\n")
	}

	// Git section
	if config.Repository.Git.RepoURL != "" || config.Repository.Git.DefaultBranch != "" {
		content.WriteString("\n[repository.git]\n")
		if config.Repository.Git.RepoURL != "" {
			content.WriteString(fmt.Sprintf("repo_url = \"%s\"\n", config.Repository.Git.RepoURL))
		}
		if config.Repository.Git.DefaultBranch != "" {
			content.WriteString(fmt.Sprintf("default_branch = \"%s\"\n", config.Repository.Git.DefaultBranch))
		}
	}

	// Worktrees section
	content.WriteString("\n[repository.worktrees]\n")
	content.WriteString(fmt.Sprintf("directory = \"%s\"\n", config.Repository.Worktrees.Directory))

	// Services section
	if len(services) > 0 {
		content.WriteString("\n[services]\n")
		for name, image := range services {
			content.WriteString(fmt.Sprintf("%s = \"%s\"\n", name, image))
		}
	}

	// Setup section
	content.WriteString("\n[project.setup]\n")
	content.WriteString("post_start = []\n")
	content.WriteString("pre_stop = []\n")

	return os.WriteFile(path, []byte(content.String()), 0644)
}
