package store

import (
	"strings"

	corecircuitbreaker "mannaiah/module/core/circuitbreaker"

	"go.uber.org/zap"
)

// Breaker defines circuit breaker behavior required by Redis store operations.
type Breaker interface {
	// Execute runs operations through a circuit breaker.
	Execute(operation func() error) error
	// IsOpenError reports whether the provided error corresponds to open-circuit behavior.
	IsOpenError(err error) bool
}

// resolveLogger returns either the provided logger or a no-op logger fallback.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// normalizeRequiredKey validates and normalizes key input.
func normalizeRequiredKey(key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", ErrEmptyKey
	}

	return trimmed, nil
}

// normalizePattern normalizes empty key matchers to a wildcard.
func normalizePattern(pattern string) string {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return "*"
	}

	return trimmed
}

// normalizeScanCount ensures SCAN count hints are always valid.
func normalizeScanCount(scanCount int64) int64 {
	if scanCount <= 0 {
		return 200
	}

	return scanCount
}

// normalizeBatchSize ensures batched reads always use a valid size.
func normalizeBatchSize(batchSize int) int {
	if batchSize <= 0 {
		return 200
	}

	return batchSize
}

// executeWithBreaker executes Redis operations with optional circuit-breaker protection.
func (s *Store) executeWithBreaker(operation func() error) error {
	if s.breaker == nil {
		return operation()
	}

	err := s.breaker.Execute(operation)
	if s.breaker.IsOpenError(err) {
		s.logger.Warn("redis circuit breaker is open; operation rejected")
		return ErrUnavailable
	}

	return err
}

// newCircuitBreaker creates a circuit breaker for Redis operations.
func newCircuitBreaker(cfg Config, providedLogger *zap.Logger) (Breaker, error) {
	return corecircuitbreaker.NewBreaker(
		corecircuitbreaker.Config{
			Name:             "redis-store",
			MaxRequests:      cfg.CircuitBreakerMaxRequests,
			IntervalMS:       cfg.CircuitBreakerIntervalMS,
			TimeoutMS:        cfg.CircuitBreakerTimeoutMS,
			FailureThreshold: cfg.CircuitBreakerFailureThreshold,
		},
		providedLogger,
	)
}
