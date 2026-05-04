package service

import (
	"strings"

	"go.uber.org/zap"
)

const (
	// defaultPageSize defines fallback order page-size values.
	defaultPageSize = 100
	// defaultWorkerCount defines fallback concurrent worker values.
	defaultWorkerCount = 8
)

// normalizeSyncConfig resolves sync config defaults.
func normalizeSyncConfig(cfg SyncConfig) SyncConfig {
	if cfg.PageSize <= 0 {
		cfg.PageSize = defaultPageSize
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = defaultWorkerCount
	}

	return cfg
}

// normalizeTrigger normalizes sync trigger labels.
func normalizeTrigger(trigger string) string {
	normalized := strings.TrimSpace(trigger)
	if normalized == "" {
		return "manual"
	}

	return normalized
}

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// resolveCircuitBreakers resolves optional breaker dependencies.
func resolveCircuitBreakers(values []CircuitBreakers) CircuitBreakers {
	if len(values) == 0 {
		return CircuitBreakers{}
	}

	return values[0]
}

// executeWithBreaker runs operations behind optional circuit breakers.
func (s *OrderSyncService) executeWithBreaker(breaker CircuitBreaker, unavailableErr error, operation func() error) error {
	if breaker == nil {
		return operation()
	}

	err := breaker.Execute(operation)
	if breaker.IsOpenError(err) {
		return unavailableErr
	}

	return err
}
