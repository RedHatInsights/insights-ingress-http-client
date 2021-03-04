package config

// Configurator An interface for handling the upload configuration
type Configurator interface {
	IsEnabled() bool
	GetEndpoint() string
}

// SimpleConfigurator defines the standard config for this operator.
type SimpleConfigurator struct {
	Report   bool
	Endpoint string
}

// IsEnabled Returns the seting for the configuration
func (s *SimpleConfigurator) IsEnabled() bool {
	return s.Report
}

// GetEndpoint Returns the endpoint for the configuration
func (s *SimpleConfigurator) GetEndpoint() string {
	return s.Endpoint
}
