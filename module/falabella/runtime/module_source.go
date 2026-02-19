package runtime

import (
	"context"
	"fmt"

	corecircuitbreaker "mannaiah/module/core/circuitbreaker"
	falabellaadapter "mannaiah/module/falabella/adapter/falabella"
	"mannaiah/module/falabella/port"

	"go.uber.org/zap"
)

// source defines Falabella source behavior used by runtime composition.
type source interface {
	// Validate verifies Falabella integration availability.
	Validate(ctx context.Context) error
	// GetBrands retrieves Falabella brand payload.
	GetBrands(ctx context.Context) ([]byte, error)
	// SyncProduct upserts Falabella product values.
	SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error)
	// SyncProductImages configures Falabella product images.
	SyncProductImages(ctx context.Context, request port.SyncProductImagesRequest) ([]byte, error)
	// GetFeedStatus retrieves Falabella feed status by feed identifier.
	GetFeedStatus(ctx context.Context, feedID string) ([]byte, error)
}

// circuitBreaker defines circuit-breaker behavior used by runtime source wrappers.
type circuitBreaker interface {
	// Execute runs operations behind circuit-breaker protection.
	Execute(operation func() error) error
	// IsOpenError reports open-circuit rejections.
	IsOpenError(err error) bool
}

// protectedSource defines Falabella source wrappers with circuit-breaker protection.
type protectedSource struct {
	// source defines delegated source behavior.
	source source
	// breaker defines circuit-breaker dependencies.
	breaker circuitBreaker
}

// newSource creates Falabella source adapters from module config values.
func newSource(cfg Config, providedLogger *zap.Logger) (source, error) {
	client, err := falabellaadapter.NewClient(falabellaadapter.Config{
		URL:       cfg.URL,
		UserID:    cfg.UserID,
		APIKey:    cfg.APIKey,
		UserAgent: cfg.UserAgent,
		Version:   cfg.Version,
		Timeout:   resolveRequestTimeout(cfg.RequestTimeoutMS),
		Logger:    providedLogger,
	})
	if err != nil {
		return nil, err
	}

	breaker := newSourceCircuitBreaker(cfg, providedLogger)
	if breaker == nil {
		return client, nil
	}

	return protectedSource{source: client, breaker: breaker}, nil
}

// Validate verifies Falabella integration availability with breaker protection.
func (s protectedSource) Validate(ctx context.Context) error {
	return s.executeWithBreaker(func() error {
		return s.source.Validate(ctx)
	})
}

// GetBrands retrieves Falabella brands with breaker protection.
func (s protectedSource) GetBrands(ctx context.Context) ([]byte, error) {
	var payload []byte
	err := s.executeWithBreaker(func() error {
		var sourceErr error
		payload, sourceErr = s.source.GetBrands(ctx)
		return sourceErr
	})
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// SyncProduct upserts Falabella products with breaker protection.
func (s protectedSource) SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error) {
	var payload []byte
	err := s.executeWithBreaker(func() error {
		var sourceErr error
		payload, sourceErr = s.source.SyncProduct(ctx, request)
		return sourceErr
	})
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// SyncProductImages configures Falabella product images with breaker protection.
func (s protectedSource) SyncProductImages(ctx context.Context, request port.SyncProductImagesRequest) ([]byte, error) {
	var payload []byte
	err := s.executeWithBreaker(func() error {
		var sourceErr error
		payload, sourceErr = s.source.SyncProductImages(ctx, request)
		return sourceErr
	})
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// GetFeedStatus retrieves Falabella feed status with breaker protection.
func (s protectedSource) GetFeedStatus(ctx context.Context, feedID string) ([]byte, error) {
	var payload []byte
	err := s.executeWithBreaker(func() error {
		var sourceErr error
		payload, sourceErr = s.source.GetFeedStatus(ctx, feedID)
		return sourceErr
	})
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// executeWithBreaker runs source operations with breaker protection when configured.
func (s protectedSource) executeWithBreaker(operation func() error) error {
	if s.breaker == nil {
		return operation()
	}

	err := s.breaker.Execute(operation)
	if err != nil && s.breaker.IsOpenError(err) {
		return fmt.Errorf("falabella source circuit breaker is open: %w", err)
	}

	return err
}

// newSourceCircuitBreaker creates Falabella source circuit-breaker dependencies from module config values.
func newSourceCircuitBreaker(cfg Config, providedLogger *zap.Logger) circuitBreaker {
	if !cfg.CircuitBreakerEnabled {
		return nil
	}

	breaker, err := corecircuitbreaker.NewBreaker(
		corecircuitbreaker.Config{
			Name:             "falabella-source",
			MaxRequests:      cfg.CircuitBreakerMaxRequests,
			IntervalMS:       cfg.CircuitBreakerIntervalMS,
			TimeoutMS:        cfg.CircuitBreakerTimeoutMS,
			FailureThreshold: cfg.CircuitBreakerFailureThreshold,
		},
		providedLogger,
	)
	if err != nil {
		return nil
	}

	return breaker
}

// failingSource defines unavailable Falabella source behavior.
type failingSource struct {
	// err defines startup validation errors.
	err error
}

// Validate returns startup validation failures.
func (f failingSource) Validate(ctx context.Context) error {
	return f.err
}

// GetBrands returns startup validation failures.
func (f failingSource) GetBrands(ctx context.Context) ([]byte, error) {
	return nil, f.err
}

// SyncProduct returns startup validation failures.
func (f failingSource) SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error) {
	return nil, f.err
}

// SyncProductImages returns startup validation failures.
func (f failingSource) SyncProductImages(ctx context.Context, request port.SyncProductImagesRequest) ([]byte, error) {
	return nil, f.err
}

// GetFeedStatus returns startup validation failures.
func (f failingSource) GetFeedStatus(ctx context.Context, feedID string) ([]byte, error) {
	return nil, f.err
}

var (
	// _ ensures runtime source wrappers satisfy Falabella source contracts.
	_ source = (*protectedSource)(nil)
	_ source = (*failingSource)(nil)
)
