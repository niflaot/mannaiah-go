package runtime

import (
	"context"
	"time"

	corecircuitbreaker "mannaiah/module/core/circuitbreaker"
	ordersport "mannaiah/module/orders/port"
	shopifyhttp "mannaiah/module/shopify/adapter/http"
	shopifyadapter "mannaiah/module/shopify/adapter/shopify"
	shopifycontactservice "mannaiah/module/shopify/application/contact/service"
	shopifyport "mannaiah/module/shopify/port"

	"go.uber.org/zap"
)

type sourceGateway interface {
	shopifyport.CustomerSource
	shopifyport.OrderSource
	shopifyport.ShopifyOrderDestination
	shopifyport.ShopifyFulfillmentDestination
	shopifyhttp.OAuthClient
}

// newSource creates Shopify source adapters from module config values.
func newSource(cfg Config, resolver shopifyport.InstallationResolver) (sourceGateway, error) {
	return shopifyadapter.NewClient(shopifyadapter.Config{
		ClientID:                  cfg.ClientID,
		ClientSecret:              cfg.ClientSecret,
		TokenResolver:             resolver,
		Timeout:                   time.Duration(resolveRequestTimeout(cfg.RequestTimeoutMS)),
		AdminRateLimitInterval:    time.Duration(resolveDurationMS(cfg.AdminRateLimitIntervalMS)),
		TooManyRequestsRetryDelay: time.Duration(resolveDurationMS(cfg.TooManyRequestsRetryDelayMS)),
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

// FindCustomerByEmail returns startup validation failures.
func (f failingSource) FindCustomerByEmail(ctx context.Context, email string) (shopifyport.ShopifyCustomer, error) {
	_ = ctx
	_ = email
	return shopifyport.ShopifyCustomer{}, f.err
}

// ListCustomers returns startup validation failures.
func (f failingSource) ListCustomers(ctx context.Context, sinceID string, limit int) ([]shopifyport.ShopifyCustomer, bool, error) {
	_ = ctx
	_ = sinceID
	_ = limit
	return nil, false, f.err
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

// ApplyOrderUpdate returns startup validation failures.
func (f failingSource) ApplyOrderUpdate(ctx context.Context, shopifyOrderID string, payload ordersport.OrderEventPayload, variantResolver shopifyport.ShopifyVariantResolver) error {
	_ = ctx
	_ = shopifyOrderID
	_ = payload
	_ = variantResolver
	return f.err
}

// CancelOrder returns startup validation failures.
func (f failingSource) CancelOrder(ctx context.Context, shopifyOrderID string, reason string) error {
	_ = ctx
	_ = shopifyOrderID
	_ = reason
	return f.err
}

// FulfillOrder returns startup validation failures.
func (f failingSource) FulfillOrder(ctx context.Context, input shopifyport.ShopifyFulfillOrderInput) (string, error) {
	_ = ctx
	_ = input
	return "", f.err
}

// CancelFulfillment returns startup validation failures.
func (f failingSource) CancelFulfillment(ctx context.Context, fulfillmentID string) error {
	_ = ctx
	_ = fulfillmentID
	return f.err
}
