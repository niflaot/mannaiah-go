package contact

import (
	"strings"

	"go.uber.org/zap"
)

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// normalizeSyncConfig normalizes sync config defaults.
func normalizeSyncConfig(cfg SyncConfig) SyncConfig {
	resolved := cfg
	if resolved.PageSize <= 0 {
		resolved.PageSize = 100
	}
	if resolved.WorkerCount <= 0 {
		resolved.WorkerCount = 8
	}

	return resolved
}

// resolveCircuitBreakers resolves optional breaker configuration values.
func resolveCircuitBreakers(values []CircuitBreakers) CircuitBreakers {
	if len(values) == 0 {
		return CircuitBreakers{}
	}

	return values[0]
}

// normalizeTrigger resolves sync trigger fallback values.
func normalizeTrigger(trigger string) string {
	resolved := strings.TrimSpace(trigger)
	if resolved == "" {
		return "manual"
	}

	return resolved
}

// executeWithBreaker executes operations with optional circuit-breaker protection.
func (s *ContactSyncService) executeWithBreaker(breaker CircuitBreaker, openError error, operation func() error) error {
	if breaker == nil {
		return operation()
	}

	err := breaker.Execute(operation)
	if breaker.IsOpenError(err) {
		return openError
	}

	return err
}
