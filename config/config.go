package config

// Configuration defines the standard config for this operator.
type Configuration struct {
	Report         bool
	StoragePath    string
	Endpoint       string
	ReportEndpoint string
	Impersonate    string

	Username string
	Password string
	Token    string

	HTTPConfig HTTPConfig
}

// HTTPConfig configures http proxy and exception settings if they come from config
type HTTPConfig struct {
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}
