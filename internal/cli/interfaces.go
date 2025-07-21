package cli

// Command interfaces - these extend the base interfaces with any CLI-specific needs

// ContainerManagerInterface extends ContainerManager for command usage
type ContainerManagerInterface interface {
	ContainerManager
}

// GitManagerInterface extends GitManager for command usage
type GitManagerInterface interface {
	GitManager
}

// ServiceManagerInterface extends ServiceManager for command usage
type ServiceManagerInterface interface {
	ServiceManager
}
