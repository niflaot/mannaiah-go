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

// orderGateway defines WooCommerce source and destination behavior used by runtime composition.
type orderGateway interface {
	port.OrderSource
	port.OrderDestination
	port.CouponSource
	port.CouponDestination
}

// newSource creates WooCommerce order source adapters from module config values.
func newSource(cfg Config) (orderGateway, error) {
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

// UpdateOrderFromMainstream returns startup validation failures.
func (f failingSource) UpdateOrderFromMainstream(ctx context.Context, command port.MainstreamOrderUpdateCommand) error {
	return f.err
}

// ListCoupons returns startup validation failures.
func (f failingSource) ListCoupons(ctx context.Context, page int, pageSize int) (coupons []port.WooCoupon, hasNext bool, err error) {
	return nil, false, f.err
}

// GetCouponByID returns startup validation failures.
func (f failingSource) GetCouponByID(ctx context.Context, id int) (port.WooCoupon, error) {
	return port.WooCoupon{}, f.err
}

// UpsertCoupon returns startup validation failures.
func (f failingSource) UpsertCoupon(ctx context.Context, command port.CouponSyncCommand) (port.CouponSyncResult, error) {
	return port.CouponSyncResult{}, f.err
}
