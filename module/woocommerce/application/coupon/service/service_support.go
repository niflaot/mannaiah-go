package service

import (
	"go.uber.org/zap"
)

// resolveLogger resolves non-nil logger dependencies.
func resolveLogger(provided *zap.Logger) *zap.Logger {
	if provided != nil {
		return provided
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return zap.NewNop()
	}

	return logger
}

// normalizeSyncConfig resolves minimum viable sync config defaults.
func normalizeSyncConfig(cfg SyncConfig) SyncConfig {
	if cfg.PageSize <= 0 {
		cfg.PageSize = 100
	}

	return cfg
}

// resolveCircuitBreakers resolves optional circuit-breaker wiring.
func resolveCircuitBreakers(provided []CircuitBreakers) CircuitBreakers {
	if len(provided) > 0 {
		return provided[0]
	}

	return CircuitBreakers{}
}
