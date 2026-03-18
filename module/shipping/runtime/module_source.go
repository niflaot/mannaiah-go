package runtime

import (
	"context"
	"errors"
	"fmt"

	corecircuitbreaker "mannaiah/module/core/circuitbreaker"
	"mannaiah/module/shipping/adapter/tcc"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"

	"go.uber.org/zap"
)

var (
	// ErrShippingDisabled is returned when shipping quote behavior is disabled.
	ErrShippingDisabled = errors.New("shipping quote behavior is disabled")
)

// circuitBreaker defines circuit-breaker behavior used by shipping source wrappers.
type circuitBreaker interface {
	// Execute runs operations behind circuit-breaker protection.
	Execute(operation func() error) error
	// IsOpenError reports open-circuit rejections.
	IsOpenError(err error) bool
}

// protectedGateway defines quote gateway wrappers with circuit-breaker protection.
type protectedGateway struct {
	// gateway defines delegated quote gateway behavior.
	gateway port.RateQuoteGateway
	// breaker defines circuit-breaker dependencies.
	breaker circuitBreaker
}

// Quote retrieves one shipping quote through breaker protection.
func (g protectedGateway) Quote(ctx context.Context, request domain.QuoteRequest) (*domain.QuoteResult, error) {
	if g.breaker == nil {
		return g.gateway.Quote(ctx, request)
	}

	var (
		result *domain.QuoteResult
		err    error
	)
	execErr := g.breaker.Execute(func() error {
		result, err = g.gateway.Quote(ctx, request)
		return err
	})
	if execErr != nil {
		if g.breaker.IsOpenError(execErr) {
			return nil, fmt.Errorf("%w: shipping tcc breaker is open", domain.ErrIntegrationUnavailable)
		}
		return nil, execErr
	}

	return result, nil
}

// failingGateway defines unavailable quote gateway behavior.
type failingGateway struct {
	// err defines startup validation errors.
	err error
}

// Quote returns startup validation failures.
func (g failingGateway) Quote(ctx context.Context, request domain.QuoteRequest) (*domain.QuoteResult, error) {
	if errors.Is(g.err, domain.ErrIntegrationUnavailable) {
		return nil, g.err
	}

	return nil, fmt.Errorf("%w: %v", domain.ErrIntegrationUnavailable, g.err)
}

// newTCCGateway creates TCC quote gateways from module config values.
func newTCCGateway(cfg Config, providedLogger *zap.Logger) (port.RateQuoteGateway, error) {
	if !cfg.Enabled {
		return nil, ErrShippingDisabled
	}

	client, err := tcc.NewClient(tcc.Config{
		BaseURL:     cfg.TCCBaseURL,
		AccessToken: cfg.TCCAccessToken,
		Account:     cfg.TCCAccount,
		Identifier:  cfg.TCCIdentifier,
		LegalName:   cfg.TCCLegalName,
		Timeout:     resolveRequestTimeout(cfg.TCCRequestTimeoutMS),
	})
	if err != nil {
		return nil, err
	}

	breaker := newTCCBreaker(cfg, providedLogger)
	if breaker == nil {
		return client, nil
	}

	return protectedGateway{
		gateway: client,
		breaker: breaker,
	}, nil
}

// newTCCBreaker creates TCC quote circuit-breaker dependencies from config values.
func newTCCBreaker(cfg Config, providedLogger *zap.Logger) circuitBreaker {
	if !cfg.TCCCircuitBreakerEnabled {
		return nil
	}

	breaker, err := corecircuitbreaker.NewBreaker(
		corecircuitbreaker.Config{
			Name:             "shipping-tcc",
			MaxRequests:      cfg.TCCCircuitBreakerMaxRequests,
			IntervalMS:       cfg.TCCCircuitBreakerIntervalMS,
			TimeoutMS:        cfg.TCCCircuitBreakerTimeoutMS,
			FailureThreshold: cfg.TCCCircuitBreakerFailureThreshold,
		},
		providedLogger,
	)
	if err != nil {
		return nil
	}

	return breaker
}

var (
	// _ ensures source wrappers satisfy quote gateway contracts.
	_ port.RateQuoteGateway = (*protectedGateway)(nil)
	_ port.RateQuoteGateway = (*failingGateway)(nil)
)
