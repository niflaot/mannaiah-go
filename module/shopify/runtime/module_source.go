package runtime

import (
	"context"
	"time"

	corecircuitbreaker "mannaiah/module/core/circuitbreaker"
	shopifyhttp "mannaiah/module/shopify/adapter/http"
	shopifyadapter "mannaiah/module/shopify/adapter/shopify"
	shopifycontactservice "mannaiah/module/shopify/application/contact/service"
	shopifyorderservice "mannaiah/module/shopify/application/order/service"
	shopifyport "mannaiah/module/shopify/port"

	"go.uber.org/zap"
)

type sourceGateway interface {
	shopifyport.CustomerSource
	shopifyport.CustomerDestination
	shopifyport.OrderSource
	shopifyport.OrderDestination
	shopifyhttp.OAuthClient
}

// newSource creates Shopify source and destination adapters from module config values.
func newSource(cfg Config, resolver shopifyport.InstallationResolver) (sourceGateway, error) {
	return shopifyadapter.NewClient(shopifyadapter.Config{
		ClientID:      cfg.ClientID,
		ClientSecret:  cfg.ClientSecret,
		TokenResolver: resolver,
		Timeout:       time.Duration(resolveRequestTimeout(cfg.RequestTimeoutMS)),
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

// ListCustomers returns startup validation failures.
func (f failingSource) ListCustomers(ctx context.Context, sinceID string, limit int) ([]shopifyport.ShopifyCustomer, bool, error) {
	_ = ctx
	_ = sinceID
	_ = limit
	return nil, false, f.err
}

// UpdateCustomerTags returns startup validation failures.
func (f failingSource) UpdateCustomerTags(ctx context.Context, id string, tags []string) error {
	_ = ctx
	_ = id
	_ = tags
	return f.err
}

// AppendCustomerNote returns startup validation failures.
func (f failingSource) AppendCustomerNote(ctx context.Context, id string, note string) error {
	_ = ctx
	_ = id
	_ = note
	return f.err
}

// GetOrder returns startup validation failures.
func (f failingSource) GetOrder(ctx context.Context, id string) (shopifyport.ShopifyOrder, error) {
	_ = ctx
	_ = id
	return shopifyport.ShopifyOrder{}, f.err
}

// ListOrders returns startup validation failures.
func (f failingSource) ListOrders(ctx context.Context, sinceID string, limit int) ([]shopifyport.ShopifyOrder, bool, error) {
	_ = ctx
	_ = sinceID
	_ = limit
	return nil, false, f.err
}

// UpdateOrderFromMainstream returns startup validation failures.
func (f failingSource) UpdateOrderFromMainstream(ctx context.Context, shopifyID string, command shopifyport.MainstreamOrderUpdateCommand) error {
	_ = ctx
	_ = shopifyID
	_ = command
	return f.err
}

// ExchangeAuthorizationCode returns startup validation failures.
func (f failingSource) ExchangeAuthorizationCode(ctx context.Context, shopDomain string, code string) (string, string, error) {
	_ = ctx
	_ = shopDomain
	_ = code
	return "", "", f.err
}

// RegisterWebhooks returns startup validation failures.
func (f failingSource) RegisterWebhooks(ctx context.Context, shopDomain string, accessToken string, address string) error {
	_ = ctx
	_ = shopDomain
	_ = accessToken
	_ = address
	return f.err
}
