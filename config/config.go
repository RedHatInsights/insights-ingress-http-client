package config

import (
	"time"
)

// Configuration defines the standard config for this operator.
type Configuration struct {
	Report               bool
	StoragePath          string
	Interval             time.Duration
	Endpoint             string
	ReportEndpoint       string
	ReportPullingDelay   time.Duration
	ReportMinRetryTime   time.Duration
	ReportPullingTimeout time.Duration
	Impersonate          string

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
