// Package constants defines application-wide constants to avoid magic numbers
package constants

import "time"

// Network and Port Constants
const (
	// DefaultServerPort is the default port for the Vibeman API server
	DefaultServerPort = 8080
	
	// DefaultWebUIPort is the default port for the web UI development server
	DefaultWebUIPort = 8081
	
	// DefaultDevPort is the default port used in development environments
	DefaultDevPort = 3000
)

// File System Permissions
const (
	// DirPermissions is the standard directory permissions for vibeman directories
	DirPermissions = 0755
	
	// FilePermissions is the standard file permissions for vibeman config files
	FilePermissions = 0644
	
	// SecureDirPermissions is used for directories containing sensitive data
	SecureDirPermissions = 0700
	
	// SecureFilePermissions is used for files containing sensitive data
	SecureFilePermissions = 0600
)

// Database Configuration
const (
	// DefaultMaxOpenConnections is the default maximum number of database connections
	DefaultMaxOpenConnections = 25
	
	// DefaultMaxIdleConnections is the default maximum number of idle database connections
	DefaultMaxIdleConnections = 5
	
	// DefaultConnectionTimeout is the default database connection timeout
	DefaultConnectionTimeout = 5 * time.Minute
	
	// DefaultIdleTimeout is the default database idle connection timeout
	DefaultIdleTimeout = 1 * time.Minute
)

// HTTP Configuration
const (
	// DefaultHTTPClientTimeout is the default timeout for HTTP client requests
	DefaultHTTPClientTimeout = 30 * time.Second
	
	// DefaultServerReadTimeout is the default server read timeout
	DefaultServerReadTimeout = 10 * time.Second
	
	// DefaultServerWriteTimeout is the default server write timeout
	DefaultServerWriteTimeout = 10 * time.Second
	
	// DefaultServerShutdownTimeout is the default server graceful shutdown timeout
	DefaultServerShutdownTimeout = 30 * time.Second
)

// Pagination Constants
const (
	// DefaultPageSize is the default number of items per page in paginated responses
	DefaultPageSize = 20
	
	// MaxPageSize is the maximum allowed page size to prevent resource exhaustion
	MaxPageSize = 100
)

// Container and Service Management
const (
	// DefaultExecutorPoolSize is the default size of the container executor pool
	DefaultExecutorPoolSize = 5
	
	// DefaultPoolIdleTimeout is the default timeout for idle executors in the pool
	DefaultPoolIdleTimeout = 5 * time.Minute
	
	// DefaultServiceOperationTimeout is the default timeout for service operations
	DefaultServiceOperationTimeout = 30 * time.Second
)

// Logging and Output Limits
const (
	// DefaultLogTailLines is the default number of log lines to display
	DefaultLogTailLines = 100
	
	// DefaultServiceLogTailLines is the default number of service log lines to display
	DefaultServiceLogTailLines = 50
	
	// MaxErrorMessageLength is the maximum length for error messages before truncation
	MaxErrorMessageLength = 500
	
	// MaxOutputLength is the maximum length for command output before truncation
	MaxOutputLength = 200
)

// Network Port Validation
const (
	// MinPortNumber is the minimum valid TCP port number
	MinPortNumber = 1
	
	// MaxPortNumber is the maximum valid TCP port number
	MaxPortNumber = 65535
)

// Timing and Delays
const (
	// DefaultRetryDelay is the default delay between retry attempts
	DefaultRetryDelay = 100 * time.Millisecond
	
	// DefaultServiceWaitDelay is the default delay when waiting for service operations
	DefaultServiceWaitDelay = 1 * time.Second
	
	// DefaultServiceStartDelay is the default delay for service startup operations
	DefaultServiceStartDelay = 2 * time.Second
)