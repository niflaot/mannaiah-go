package runtime

import (
	"context"
	"time"

	corecircuitbreaker "mannaiah/module/core/circuitbreaker"
	wooadapter "mannaiah/module/woocommerce/adapter/woocommerce"
	woocontactservice "mannaiah/module/woocommerce/application/contact/service"
	"mannaiah/module/woocommerce/port"

	"go.uber.org/zap"
)

// newSource creates WooCommerce order source adapters from module config values.
func newSource(cfg Config) (port.OrderSource, error) {
	return wooadapter.NewClient(wooadapter.Config{
		URL:            cfg.URL,
		ConsumerKey:    cfg.ConsumerKey,
		ConsumerSecret: cfg.ConsumerSecret,
		Timeout:        time.Duration(resolveRequestTimeout(cfg.RequestTimeoutMS)) * time.Millisecond,
		VerifySSL:      cfg.VerifySSL,
	})
}

// newSourceCircuitBreaker creates WooCommerce source circuit-breaker dependencies from module config values.
func newSourceCircuitBreaker(cfg Config, providedLogger *zap.Logger) woocontactservice.CircuitBreaker {
	if !cfg.CircuitBreakerEnabled {
		return nil
	}

	breaker, err := corecircuitbreaker.NewBreaker(
		corecircuitbreaker.Config{
			Name:             "woocommerce-source",
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

// failingSource defines unavailable WooCommerce source behavior.
type failingSource struct {
	// err defines startup validation errors.
	err error
}

// Validate returns startup validation failures.
func (f failingSource) Validate(ctx context.Context) error {
	return f.err
}

// ListOrders returns startup validation failures.
func (f failingSource) ListOrders(ctx context.Context, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error) {
	return nil, false, f.err
}
