package http

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"go.uber.org/zap"
	coreconfig "mannaiah/module/core/config"
)

var (
	// ErrInvalidPort is returned when the configured port is out of valid range.
	ErrInvalidPort = errors.New("http port must be between 1 and 65535")
)

// resolveLogger returns the provided logger or a no-op logger fallback.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// AddressFrom resolves host and port values from HTTP config and optional core config.
func AddressFrom(cfg Config, coreCfg *coreconfig.Core) (string, error) {
	resolved := mergeConfig(cfg, coreCfg)
	address, err := buildAddress(resolved.Host, resolved.Port)
	if err != nil {
		return "", fmt.Errorf("resolve http address: %w", err)
	}

	return address, nil
}

// mergeConfig resolves HTTP config values using optional core config fallbacks.
func mergeConfig(cfg Config, coreCfg *coreconfig.Core) Config {
	resolved := cfg

	if coreCfg != nil && strings.TrimSpace(coreCfg.Host) != "" {
		resolved.Host = strings.TrimSpace(coreCfg.Host)
	} else if strings.TrimSpace(resolved.Host) == "" {
		resolved.Host = "0.0.0.0"
	}

	if coreCfg != nil && coreCfg.Port > 0 {
		resolved.Port = coreCfg.Port
	} else if resolved.Port <= 0 {
		resolved.Port = 8080
	}

	if strings.TrimSpace(resolved.AppName) == "" {
		resolved.AppName = "mannaiah-http"
	}
	if strings.TrimSpace(resolved.ServerHeader) == "" {
		resolved.ServerHeader = "mannaiah"
	}
	if resolved.ReadTimeoutMS <= 0 {
		resolved.ReadTimeoutMS = 30000
	}
	if resolved.WriteTimeoutMS <= 0 {
		resolved.WriteTimeoutMS = 30000
	}
	if resolved.IdleTimeoutMS <= 0 {
		resolved.IdleTimeoutMS = 120000
	}

	return resolved
}

// buildAddress validates host and port and returns host:port format.
func buildAddress(host string, port int) (string, error) {
	if port <= 0 || port > 65535 {
		return "", ErrInvalidPort
	}

	return net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(port)), nil
}
