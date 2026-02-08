package config

import "mannaiah/module/core/logger"

// Core defines the shared core runtime configuration.
type Core struct {
	// Host defines the bind or advertised host for the service.
	Host string `mapstructure:"CORE_HOST" default:"0.0.0.0"`
	// Port defines the bind port for the service.
	Port int `mapstructure:"CORE_PORT" default:"8080"`
	// Environment identifies the runtime environment.
	Environment string `mapstructure:"CORE_ENVIRONMENT" default:"development"`
	// Logging defines shared logging settings.
	Logging logger.Settings `mapstructure:",squash"`
}
