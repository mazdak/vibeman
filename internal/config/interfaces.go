package config

// ConfigDiscoverer defines the interface for finding configuration files
type ConfigDiscoverer interface {
	FindRepositoryConfig() (string, error)
}

// ConfigLoader defines the interface for loading configuration
type ConfigLoader interface {
	Load() error
	IsUnified() bool
}

// Validatable defines the interface for configuration validation
type Validatable interface {
	Validate() error
}
