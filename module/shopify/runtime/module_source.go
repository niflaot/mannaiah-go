package runtime

import (
	"context"
	"time"

	corecircuitbreaker "mannaiah/module/core/circuitbreaker"
	shopifyadapter "mannaiah/module/shopify/adapter/shopify"
	shopifycontactservice "mannaiah/module/shopify/application/contact/service"
	shopifyorderservice "mannaiah/module/shopify/application/order/service"
	shopifyport "mannaiah/module/shopify/port"

	"go.uber.org/zap"
)

type sourceGateway interface {
	shopifyport.CustomerSource
	shopifyport.OrderSource
	shopifyport.OrderDestination
}

// newSource creates Shopify source and destination adapters from module config values.
func newSource(cfg Config) (sourceGateway, error) {
	return shopifyadapter.NewClient(shopifyadapter.Config{
		Domain:      cfg.ShopDomain,
		AccessToken: cfg.AccessToken,
		Timeout:     time.Duration(resolveRequestTimeout(cfg.RequestTimeoutMS)),
	})
}

// newSourceCircuitBreaker creates Shopify source circuit-breaker dependencies from module config values.
func newSourceCircuitBreaker(cfg Config, providedLogger *zap.Logger) shopifycontactservice.CircuitBreaker {
	if !cfg.CircuitBreakerEnabled {
		return nil
	}

	breaker, err := corecircuitbreaker.NewBreaker(corecircuitbreaker.Config{
		Name:             "shopify-source",
		MaxRequests:      cfg.CircuitBreakerMaxRequests,
		IntervalMS:       cfg.CircuitBreakerIntervalMS,
		TimeoutMS:        cfg.CircuitBreakerTimeoutMS,
		FailureThreshold: cfg.CircuitBreakerFailureThreshold,
	}, providedLogger)
	if err != nil {
		return nil
	}

	return breaker
}

// newDestinationCircuitBreaker creates Shopify destination circuit-breaker dependencies from module config values.
func newDestinationCircuitBreaker(cfg Config, providedLogger *zap.Logger) shopifyorderservice.CircuitBreaker {
	if !cfg.CircuitBreakerEnabled {
		return nil
	}

	breaker, err := corecircuitbreaker.NewBreaker(corecircuitbreaker.Config{
		Name:             "shopify-destination",
		MaxRequests:      cfg.CircuitBreakerMaxRequests,
		IntervalMS:       cfg.CircuitBreakerIntervalMS,
		TimeoutMS:        cfg.CircuitBreakerTimeoutMS,
		FailureThreshold: cfg.CircuitBreakerFailureThreshold,
	}, providedLogger)
	if err != nil {
		return nil
	}

	return breaker
}

// failingSource defines unavailable Shopify source behavior.
type failingSource struct {
	// err defines startup validation errors.
	err error
}

// Validate returns startup validation failures.
func (f failingSource) Validate(ctx context.Context) error {
	_ = ctx
	return f.err
}

// GetCustomer returns startup validation failures.
func (f failingSource) GetCustomer(ctx context.Context, id string) (shopifyport.ShopifyCustomer, error) {
	_ = ctx
	_ = id
	return shopifyport.ShopifyCustomer{}, f.err
}

// GetOrder returns startup validation failures.
func (f failingSource) GetOrder(ctx context.Context, id string) (shopifyport.ShopifyOrder, error) {
	_ = ctx
	_ = id
	return shopifyport.ShopifyOrder{}, f.err
}

// UpdateOrderFromMainstream returns startup validation failures.
func (f failingSource) UpdateOrderFromMainstream(ctx context.Context, shopifyID string, command shopifyport.MainstreamOrderUpdateCommand) error {
	_ = ctx
	_ = shopifyID
	_ = command
	return f.err
}
