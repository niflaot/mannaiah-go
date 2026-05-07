package shopify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	ordersport "mannaiah/module/orders/port"
	shopifyport "mannaiah/module/shopify/port"
)

const (
	apiVersion                    = "2026-04"
	defaultTimeout                = 5 * time.Second
	defaultAdminRateLimitInterval = 1200 * time.Millisecond
	defaultTooManyRequestsDelay   = 5 * time.Second
	maxRetries                    = 2
)

var (
	// ErrDomainRequired is returned when Shopify shop domains are empty.
	ErrDomainRequired = errors.New("shopify domain is required")
	// ErrAccessTokenRequired is returned when Shopify access tokens are empty.
	ErrAccessTokenRequired = errors.New("shopify access token is required")
	// ErrClientIDRequired is returned when Shopify OAuth client identifiers are empty.
	ErrClientIDRequired = errors.New("shopify client id is required")
	// ErrClientSecretRequired is returned when Shopify client secrets are empty.
	ErrClientSecretRequired = errors.New("shopify client secret is required")
	// ErrTokenResolverRequired is returned when installation token resolvers are nil.
	ErrTokenResolverRequired = errors.New("shopify token resolver is required")
	// ErrCodeRequired is returned when Shopify OAuth codes are empty.
	ErrCodeRequired = errors.New("shopify authorization code is required")
	// ErrWebhookAddressRequired is returned when webhook callback addresses are empty.
	ErrWebhookAddressRequired = errors.New("shopify webhook address is required")
)

// Config defines Shopify Admin API client configuration values.
type Config struct {
	// ClientID defines Shopify OAuth client identifier values.
	ClientID string
	// ClientSecret defines Shopify client secret values.
	ClientSecret string
	// TokenResolver defines active-installation lookup behavior.
	TokenResolver shopifyport.InstallationResolver
	// Timeout defines request timeout values.
	Timeout time.Duration
	// AdminRateLimitInterval defines minimum spacing between Shopify Admin API calls.
	AdminRateLimitInterval time.Duration
	// TooManyRequestsRetryDelay defines fallback wait time after 429 responses without Retry-After.
	TooManyRequestsRetryDelay time.Duration
	// BaseURL overrides the computed Shopify Admin base URL for testing.
	BaseURL string
}

// Client defines the Shopify Admin REST adapter.
type Client struct {
	// clientID defines Shopify OAuth client identifier values.
	clientID string
	// clientSecret defines Shopify OAuth client secret values.
	clientSecret string
	// tokenResolver defines active-installation lookup behavior.
	tokenResolver shopifyport.InstallationResolver
	// baseURL defines optional base URL overrides for tests.
	baseURL string
	// httpClient defines HTTP transport dependencies.
	httpClient *http.Client
	// rateLimitInterval defines minimum spacing between Shopify Admin API calls.
	rateLimitInterval time.Duration
	// tooManyRequestsRetryDelay defines fallback wait time after 429 responses without Retry-After.
	tooManyRequestsRetryDelay time.Duration
	// rateMu serializes per-shop Admin API pacing.
	rateMu sync.Mutex
	// lastAdminRequestAt stores the latest Admin API request timestamp per shop.
	lastAdminRequestAt map[string]time.Time
}

// NewClient creates Shopify Admin REST clients.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, ErrClientIDRequired
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, ErrClientSecretRequired
	}
	if cfg.TokenResolver == nil {
		return nil, ErrTokenResolverRequired
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	rateLimitInterval := cfg.AdminRateLimitInterval
	if rateLimitInterval == 0 && strings.TrimSpace(cfg.BaseURL) == "" {
		rateLimitInterval = defaultAdminRateLimitInterval
	}
	if rateLimitInterval < 0 {
		rateLimitInterval = 0
	}
	tooManyRequestsRetryDelay := cfg.TooManyRequestsRetryDelay
	if tooManyRequestsRetryDelay <= 0 {
		tooManyRequestsRetryDelay = defaultTooManyRequestsDelay
	}

	return &Client{
		clientID:                  strings.TrimSpace(cfg.ClientID),
		clientSecret:              strings.TrimSpace(cfg.ClientSecret),
		tokenResolver:             cfg.TokenResolver,
		baseURL:                   strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		httpClient:                &http.Client{Timeout: timeout},
		rateLimitInterval:         rateLimitInterval,
		tooManyRequestsRetryDelay: tooManyRequestsRetryDelay,
		lastAdminRequestAt:        make(map[string]time.Time),
	}, nil
}

// Validate verifies connectivity and credentials against the Shopify Admin API.
func (c *Client) Validate(ctx context.Context) error {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return err
	}

	var response map[string]any
	return c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodGet, "/shop.json", nil, &response)
}

// GetCustomer resolves one Shopify customer by identifier.
func (c *Client) GetCustomer(ctx context.Context, id string) (shopifyport.ShopifyCustomer, error) {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return shopifyport.ShopifyCustomer{}, err
	}

	var response customerResponse
	path := fmt.Sprintf("/customers/%s.json", url.PathEscape(strings.TrimSpace(id)))
	if err := c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodGet, path, nil, &response); err != nil {
		if statusErr := (*statusError)(nil); errors.As(err, &statusErr) && statusErr.Code == http.StatusNotFound {
			return shopifyport.ShopifyCustomer{}, shopifyport.ErrCustomerNotFound
		}
		return shopifyport.ShopifyCustomer{}, err
	}

	customer := normalizeCustomer(response.Customer)
	customer.ShopDomain = installation.ShopDomain
	return customer, nil
}

// FindCustomerByEmail resolves one Shopify customer by email address.
func (c *Client) FindCustomerByEmail(ctx context.Context, email string) (shopifyport.ShopifyCustomer, error) {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return shopifyport.ShopifyCustomer{}, err
	}

	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" {
		return shopifyport.ShopifyCustomer{}, shopifyport.ErrCustomerNotFound
	}

	path := "/customers/search.json?query=" + url.QueryEscape("email:"+trimmedEmail)
	var response customersListResponse
	if err := c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodGet, path, nil, &response); err != nil {
		return shopifyport.ShopifyCustomer{}, err
	}
	if len(response.Customers) == 0 {
		return shopifyport.ShopifyCustomer{}, shopifyport.ErrCustomerNotFound
	}

	customer := normalizeCustomer(response.Customers[0])
	customer.ShopDomain = installation.ShopDomain
	return customer, nil
}

// ListCustomers returns up to limit customers with numeric IDs greater than sinceID.
func (c *Client) ListCustomers(ctx context.Context, sinceID string, limit int) ([]shopifyport.ShopifyCustomer, bool, error) {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return nil, false, err
	}

	path := fmt.Sprintf("/customers.json?limit=%d", limit)
	if strings.TrimSpace(sinceID) != "" {
		path += "&since_id=" + url.QueryEscape(strings.TrimSpace(sinceID))
	}

	var response customersListResponse
	if err := c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodGet, path, nil, &response); err != nil {
		return nil, false, err
	}

	customers := make([]shopifyport.ShopifyCustomer, len(response.Customers))
	for i, payload := range response.Customers {
		cust := normalizeCustomer(payload)
		cust.ShopDomain = installation.ShopDomain
		customers[i] = cust
	}

	return customers, len(response.Customers) == limit, nil
}

// GetOrder resolves one Shopify order by identifier.
func (c *Client) GetOrder(ctx context.Context, id string) (shopifyport.ShopifyOrder, error) {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return shopifyport.ShopifyOrder{}, err
	}

	var response orderResponse
	path := fmt.Sprintf("/orders/%s.json?status=any", url.PathEscape(strings.TrimSpace(id)))
	if err := c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodGet, path, nil, &response); err != nil {
		if statusErr := (*statusError)(nil); errors.As(err, &statusErr) && statusErr.Code == http.StatusNotFound {
			return shopifyport.ShopifyOrder{}, shopifyport.ErrOrderNotFound
		}
		return shopifyport.ShopifyOrder{}, err
	}

	order := normalizeOrder(response.Order)
	order.ShopDomain = installation.ShopDomain
	if order.Customer != nil {
		order.Customer.ShopDomain = installation.ShopDomain
	}
	return order, nil
}

// ListOrders returns up to limit orders with numeric IDs greater than sinceID.
func (c *Client) ListOrders(ctx context.Context, sinceID string, limit int) ([]shopifyport.ShopifyOrder, bool, error) {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return nil, false, err
	}

	path := fmt.Sprintf("/orders.json?status=any&limit=%d", limit)
	if strings.TrimSpace(sinceID) != "" {
		path += "&since_id=" + url.QueryEscape(strings.TrimSpace(sinceID))
	}

	var response ordersListResponse
	if err := c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodGet, path, nil, &response); err != nil {
		return nil, false, err
	}

	orders := make([]shopifyport.ShopifyOrder, len(response.Orders))
	for i, payload := range response.Orders {
		o := normalizeOrder(payload)
		o.ShopDomain = installation.ShopDomain
		if o.Customer != nil {
			o.Customer.ShopDomain = installation.ShopDomain
		}
		orders[i] = o
	}

	return orders, len(response.Orders) == limit, nil
}

// ExchangeAuthorizationCode exchanges one OAuth code for a permanent offline token.
func (c *Client) ExchangeAuthorizationCode(ctx context.Context, shopDomain string, code string) (string, string, error) {
	resolvedShop := shopifyport.NormalizeShopDomain(shopDomain)
	if resolvedShop == "" {
		return "", "", ErrDomainRequired
	}
	if strings.TrimSpace(code) == "" {
		return "", "", ErrCodeRequired
	}

	requestBody := map[string]string{
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
		"code":          strings.TrimSpace(code),
	}
	var response struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}
	if err := c.doJSONAbsolute(ctx, c.buildOAuthAccessTokenURL(resolvedShop), "", http.MethodPost, requestBody, &response); err != nil {
		return "", "", err
	}

	return strings.TrimSpace(response.AccessToken), strings.TrimSpace(response.Scope), nil
}

// RegisterWebhooks registers required webhook topics for one Shopify installation.
func (c *Client) RegisterWebhooks(ctx context.Context, shopDomain string, accessToken string, address string) error {
	resolvedShop := shopifyport.NormalizeShopDomain(shopDomain)
	if resolvedShop == "" {
		return ErrDomainRequired
	}
	trimmedToken := strings.TrimSpace(accessToken)
	if trimmedToken == "" {
		return ErrAccessTokenRequired
	}
	trimmedAddress := strings.TrimSpace(address)
	if trimmedAddress == "" {
		return ErrWebhookAddressRequired
	}

	for _, topic := range []string{"orders/create", "orders/updated", "customers/create", "customers/update", "app/uninstalled"} {
		requestBody := webhookRegistrationRequest{
			Webhook: webhookRegistrationPayload{
				Topic:   topic,
				Address: trimmedAddress,
				Format:  "json",
			},
		}
		err := c.doJSONWithToken(ctx, resolvedShop, trimmedToken, http.MethodPost, "/webhooks.json", requestBody, nil)
		if statusErr := (*statusError)(nil); errors.As(err, &statusErr) {
			if statusErr.Code == http.StatusConflict || statusErr.Code == http.StatusUnprocessableEntity {
				continue
			}
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// ApplyOrderUpdate applies safe order edits to one Shopify order.
func (c *Client) ApplyOrderUpdate(ctx context.Context, shopifyOrderID string, payload ordersport.OrderEventPayload, variantResolver shopifyport.ShopifyVariantResolver) error {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return err
	}
	orderGID := orderGID(shopifyOrderID)
	begin, err := c.beginOrderEdit(ctx, installation, orderGID)
	if err != nil {
		return err
	}
	calculatedOrderID := strings.TrimSpace(begin.CalculatedOrder.ID)
	if calculatedOrderID == "" {
		return errors.New("shopify order edit calculated order id is empty")
	}
	existing := mapCalculatedLineItems(begin.CalculatedOrder.LineItems.Nodes)
	for _, item := range payload.Items {
		sku := strings.TrimSpace(item.SKU)
		if line, ok := existing[strings.ToLower(sku)]; ok {
			if line.Quantity != item.Quantity {
				if err := c.orderEditSetQuantity(ctx, installation, calculatedOrderID, line.ID, item.Quantity); err != nil {
					return err
				}
			}
			continue
		}
		if variantResolver == nil {
			continue
		}
		variantID, resolveErr := variantResolver.ResolveVariantID(ctx, item.ProductID)
		if resolveErr != nil {
			return resolveErr
		}
		if strings.TrimSpace(variantID) == "" {
			continue
		}
		if err := c.orderEditAddVariant(ctx, installation, calculatedOrderID, variantID, item.Quantity); err != nil {
			return err
		}
	}
	return c.orderEditCommit(ctx, installation, calculatedOrderID, false, "Updated from Mannaiah")
}

// CancelOrder cancels one Shopify order without notifying customers.
func (c *Client) CancelOrder(ctx context.Context, shopifyOrderID string, reason string) error {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return err
	}
	var response graphqlResponse[struct {
		OrderCancel struct {
			OrderCancelUserErrors []graphqlUserError `json:"orderCancelUserErrors"`
			UserErrors            []graphqlUserError `json:"userErrors"`
		} `json:"orderCancel"`
	}]
	err = c.doGraphQL(ctx, installation, `mutation orderCancel($orderId: ID!, $notifyCustomer: Boolean, $refundMethod: OrderCancelRefundMethodInput!, $restock: Boolean!, $reason: OrderCancelReason!, $staffNote: String) {
  orderCancel(orderId: $orderId, notifyCustomer: $notifyCustomer, refundMethod: $refundMethod, restock: $restock, reason: $reason, staffNote: $staffNote) {
    orderCancelUserErrors { field message }
    userErrors { field message }
  }
}`, map[string]any{
		"orderId":        orderGID(shopifyOrderID),
		"notifyCustomer": false,
		"refundMethod":   map[string]any{"originalPaymentMethodsRefund": false},
		"restock":        true,
		"reason":         "OTHER",
		"staffNote":      strings.TrimSpace(reason),
	}, &response)
	if err != nil {
		return err
	}
	if err := graphQLUserError(response.Data.OrderCancel.OrderCancelUserErrors); err != nil {
		return err
	}
	return graphQLUserError(response.Data.OrderCancel.UserErrors)
}

// FulfillOrder creates a Shopify fulfillment for one order.
func (c *Client) FulfillOrder(ctx context.Context, input shopifyport.ShopifyFulfillOrderInput) (string, error) {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return "", err
	}
	fulfillmentOrders, err := c.listFulfillmentOrders(ctx, installation, orderGID(input.ShopifyOrderID))
	if err != nil {
		return "", err
	}
	lineItemsByFulfillmentOrder := make([]map[string]any, 0, len(fulfillmentOrders))
	for _, fulfillmentOrder := range fulfillmentOrders {
		if strings.EqualFold(fulfillmentOrder.Status, "closed") || strings.EqualFold(fulfillmentOrder.Status, "cancelled") {
			continue
		}
		lineItemsByFulfillmentOrder = append(lineItemsByFulfillmentOrder, map[string]any{"fulfillmentOrderId": fulfillmentOrder.ID})
	}
	if len(lineItemsByFulfillmentOrder) == 0 {
		return "", errors.New("shopify fulfillment order is not fulfillable")
	}
	fulfillment := map[string]any{
		"lineItemsByFulfillmentOrder": lineItemsByFulfillmentOrder,
		"notifyCustomer":              input.NotifyCustomer,
	}
	trackingInfo := map[string]any{}
	if strings.TrimSpace(input.TrackingNumber) != "" {
		trackingInfo["number"] = strings.TrimSpace(input.TrackingNumber)
	}
	if strings.TrimSpace(input.TrackingCompany) != "" {
		trackingInfo["company"] = strings.TrimSpace(input.TrackingCompany)
	}
	if strings.TrimSpace(input.TrackingURL) != "" {
		trackingInfo["url"] = strings.TrimSpace(input.TrackingURL)
	}
	if len(trackingInfo) > 0 {
		fulfillment["trackingInfo"] = trackingInfo
	}
	var response graphqlResponse[struct {
		FulfillmentCreate struct {
			Fulfillment *struct {
				ID string `json:"id"`
			} `json:"fulfillment"`
			UserErrors []graphqlUserError `json:"userErrors"`
		} `json:"fulfillmentCreate"`
	}]
	err = c.doGraphQL(ctx, installation, `mutation fulfillmentCreate($fulfillment: FulfillmentInput!) {
  fulfillmentCreate(fulfillment: $fulfillment) {
    fulfillment { id }
    userErrors { field message }
  }
}`, map[string]any{"fulfillment": fulfillment}, &response)
	if err != nil {
		return "", err
	}
	if err := graphQLUserError(response.Data.FulfillmentCreate.UserErrors); err != nil {
		return "", err
	}
	if response.Data.FulfillmentCreate.Fulfillment == nil || strings.TrimSpace(response.Data.FulfillmentCreate.Fulfillment.ID) == "" {
		return "", errors.New("shopify fulfillment id is empty")
	}
	return strings.TrimSpace(response.Data.FulfillmentCreate.Fulfillment.ID), nil
}

// CancelFulfillment cancels one Shopify fulfillment.
func (c *Client) CancelFulfillment(ctx context.Context, fulfillmentID string) error {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return err
	}
	var response graphqlResponse[struct {
		FulfillmentCancel struct {
			UserErrors []graphqlUserError `json:"userErrors"`
		} `json:"fulfillmentCancel"`
	}]
	err = c.doGraphQL(ctx, installation, `mutation fulfillmentCancel($id: ID!) {
  fulfillmentCancel(id: $id) {
    userErrors { field message }
  }
}`, map[string]any{"id": strings.TrimSpace(fulfillmentID)}, &response)
	if err != nil {
		return err
	}
	return graphQLUserError(response.Data.FulfillmentCancel.UserErrors)
}

type statusError struct {
	Code int
	Body string
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlResponse[T any] struct {
	Data   T              `json:"data"`
	Errors []graphqlError `json:"errors,omitempty"`
}

type graphqlError struct {
	Message string `json:"message"`
}

type graphqlUserError struct {
	Field   []string `json:"field"`
	Message string   `json:"message"`
}

type orderEditBeginResult struct {
	CalculatedOrder struct {
		ID        string `json:"id"`
		LineItems struct {
			Nodes []calculatedLineItem `json:"nodes"`
		} `json:"lineItems"`
	} `json:"calculatedOrder"`
	UserErrors []graphqlUserError `json:"userErrors"`
}

type calculatedLineItem struct {
	ID       string `json:"id"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
	Variant  *struct {
		ID string `json:"id"`
	} `json:"variant"`
}

type fulfillmentOrderNode struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (e *statusError) Error() string {
	return fmt.Sprintf("shopify api returned status %d: %s", e.Code, strings.TrimSpace(e.Body))
}

func (c *Client) doGraphQL(ctx context.Context, installation *shopifyport.Installation, query string, variables map[string]any, response any) error {
	if installation == nil {
		return shopifyport.ErrInstallationNotFound
	}
	if err := c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodPost, "/graphql.json", graphqlRequest{
		Query:     query,
		Variables: variables,
	}, response); err != nil {
		return err
	}
	if withErrors, ok := response.(interface{ graphQLErrors() []graphqlError }); ok {
		if err := graphQLErrors(withErrors.graphQLErrors()); err != nil {
			return err
		}
	}
	return nil
}

func (r graphqlResponse[T]) graphQLErrors() []graphqlError {
	return r.Errors
}

func (c *Client) beginOrderEdit(ctx context.Context, installation *shopifyport.Installation, orderID string) (orderEditBeginResult, error) {
	var response graphqlResponse[struct {
		OrderEditBegin orderEditBeginResult `json:"orderEditBegin"`
	}]
	err := c.doGraphQL(ctx, installation, `mutation orderEditBegin($id: ID!) {
  orderEditBegin(id: $id) {
    calculatedOrder {
      id
      lineItems(first: 100) {
        nodes {
          id
          sku
          quantity
          variant { id }
        }
      }
    }
    userErrors { field message }
  }
}`, map[string]any{"id": orderID}, &response)
	if err != nil {
		return orderEditBeginResult{}, err
	}
	if err := graphQLUserError(response.Data.OrderEditBegin.UserErrors); err != nil {
		return orderEditBeginResult{}, err
	}
	return response.Data.OrderEditBegin, nil
}

func (c *Client) orderEditSetQuantity(ctx context.Context, installation *shopifyport.Installation, calculatedOrderID string, lineItemID string, quantity int) error {
	var response graphqlResponse[struct {
		OrderEditSetQuantity struct {
			UserErrors []graphqlUserError `json:"userErrors"`
		} `json:"orderEditSetQuantity"`
	}]
	err := c.doGraphQL(ctx, installation, `mutation orderEditSetQuantity($id: ID!, $lineItemId: ID!, $quantity: Int!) {
  orderEditSetQuantity(id: $id, lineItemId: $lineItemId, quantity: $quantity) {
    userErrors { field message }
  }
}`, map[string]any{"id": calculatedOrderID, "lineItemId": lineItemID, "quantity": quantity}, &response)
	if err != nil {
		return err
	}
	return graphQLUserError(response.Data.OrderEditSetQuantity.UserErrors)
}

func (c *Client) orderEditAddVariant(ctx context.Context, installation *shopifyport.Installation, calculatedOrderID string, variantID string, quantity int) error {
	if quantity <= 0 {
		return nil
	}
	var response graphqlResponse[struct {
		OrderEditAddVariant struct {
			UserErrors []graphqlUserError `json:"userErrors"`
		} `json:"orderEditAddVariant"`
	}]
	err := c.doGraphQL(ctx, installation, `mutation orderEditAddVariant($id: ID!, $variantId: ID!, $quantity: Int!) {
  orderEditAddVariant(id: $id, variantId: $variantId, quantity: $quantity) {
    userErrors { field message }
  }
}`, map[string]any{"id": calculatedOrderID, "variantId": strings.TrimSpace(variantID), "quantity": quantity}, &response)
	if err != nil {
		return err
	}
	return graphQLUserError(response.Data.OrderEditAddVariant.UserErrors)
}

func (c *Client) orderEditCommit(ctx context.Context, installation *shopifyport.Installation, calculatedOrderID string, notifyCustomer bool, staffNote string) error {
	var response graphqlResponse[struct {
		OrderEditCommit struct {
			UserErrors []graphqlUserError `json:"userErrors"`
		} `json:"orderEditCommit"`
	}]
	err := c.doGraphQL(ctx, installation, `mutation orderEditCommit($id: ID!, $notifyCustomer: Boolean!, $staffNote: String) {
  orderEditCommit(id: $id, notifyCustomer: $notifyCustomer, staffNote: $staffNote) {
    userErrors { field message }
  }
}`, map[string]any{"id": calculatedOrderID, "notifyCustomer": notifyCustomer, "staffNote": strings.TrimSpace(staffNote)}, &response)
	if err != nil {
		return err
	}
	return graphQLUserError(response.Data.OrderEditCommit.UserErrors)
}

func (c *Client) listFulfillmentOrders(ctx context.Context, installation *shopifyport.Installation, orderID string) ([]fulfillmentOrderNode, error) {
	var response graphqlResponse[struct {
		Order *struct {
			FulfillmentOrders struct {
				Nodes []fulfillmentOrderNode `json:"nodes"`
			} `json:"fulfillmentOrders"`
		} `json:"order"`
	}]
	err := c.doGraphQL(ctx, installation, `query orderFulfillmentOrders($id: ID!) {
  order(id: $id) {
    fulfillmentOrders(first: 25) {
      nodes { id status }
    }
  }
}`, map[string]any{"id": orderID}, &response)
	if err != nil {
		return nil, err
	}
	if response.Data.Order == nil {
		return nil, shopifyport.ErrOrderNotFound
	}
	return response.Data.Order.FulfillmentOrders.Nodes, nil
}

func graphQLErrors(values []graphqlError) error {
	messages := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Message) != "" {
			messages = append(messages, strings.TrimSpace(value.Message))
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return errors.New("shopify graphql error: " + strings.Join(messages, "; "))
}

func graphQLUserError(values []graphqlUserError) error {
	messages := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Message) != "" {
			messages = append(messages, strings.TrimSpace(value.Message))
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return errors.New("shopify graphql user error: " + strings.Join(messages, "; "))
}

func mapCalculatedLineItems(values []calculatedLineItem) map[string]calculatedLineItem {
	result := make(map[string]calculatedLineItem, len(values))
	for _, value := range values {
		sku := strings.ToLower(strings.TrimSpace(value.SKU))
		if sku != "" {
			result[sku] = value
		}
	}
	return result
}

func orderGID(id string) string {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" || strings.HasPrefix(trimmed, "gid://") {
		return trimmed
	}
	return "gid://shopify/Order/" + trimmed
}

type customerResponse struct {
	Customer customerPayload `json:"customer"`
}

type customersListResponse struct {
	Customers []customerPayload `json:"customers"`
}

type orderResponse struct {
	Order orderPayload `json:"order"`
}

type ordersListResponse struct {
	Orders []orderPayload `json:"orders"`
}

type customerPayload struct {
	ID                    any                      `json:"id"`
	Email                 string                   `json:"email"`
	FirstName             string                   `json:"first_name"`
	LastName              string                   `json:"last_name"`
	Phone                 string                   `json:"phone"`
	Tags                  string                   `json:"tags"`
	Note                  string                   `json:"note"`
	EmailMarketingConsent *marketingConsentPayload `json:"email_marketing_consent"`
	SMSMarketingConsent   *marketingConsentPayload `json:"sms_marketing_consent"`
	DefaultAddress        *addressPayload          `json:"default_address"`
	NoteAttributes        []noteAttributePayload   `json:"note_attributes"`
	CreatedAt             time.Time                `json:"created_at"`
}

type orderPayload struct {
	ID                  any                    `json:"id"`
	Name                string                 `json:"name"`
	Email               string                 `json:"email"`
	FinancialStatus     string                 `json:"financial_status"`
	FulfillmentStatus   string                 `json:"fulfillment_status"`
	Currency            string                 `json:"currency"`
	TotalPrice          string                 `json:"total_price"`
	TotalTax            string                 `json:"total_tax"`
	TotalDiscounts      string                 `json:"total_discounts"`
	Note                string                 `json:"note"`
	Tags                string                 `json:"tags"`
	CancelReason        string                 `json:"cancel_reason"`
	CancelledAt         *time.Time             `json:"cancelled_at"`
	PaymentGatewayNames []string               `json:"payment_gateway_names"`
	Customer            *customerPayload       `json:"customer"`
	BillingAddress      *addressPayload        `json:"billing_address"`
	ShippingAddress     *addressPayload        `json:"shipping_address"`
	NoteAttributes      []noteAttributePayload `json:"note_attributes"`
	LineItems           []lineItemPayload      `json:"line_items"`
	ShippingLines       []shippingLinePayload  `json:"shipping_lines"`
	DiscountCodes       []discountCodePayload  `json:"discount_codes"`
	CreatedAt           time.Time              `json:"created_at"`
}

type addressPayload struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Company   string `json:"company"`
	Address1  string `json:"address1"`
	Address2  string `json:"address2"`
	City      string `json:"city"`
	Province  string `json:"province"`
	Country   string `json:"country"`
	Zip       string `json:"zip"`
	Phone     string `json:"phone"`
}

type noteAttributePayload struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type marketingConsentPayload struct {
	State            string     `json:"state"`
	MarketingState   string     `json:"marketing_state"`
	ConsentUpdatedAt *time.Time `json:"consent_updated_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

type lineItemPayload struct {
	ID           any                    `json:"id"`
	ProductID    any                    `json:"product_id"`
	VariantID    any                    `json:"variant_id"`
	SKU          string                 `json:"sku"`
	Title        string                 `json:"title"`
	VariantTitle string                 `json:"variant_title"`
	Properties   []noteAttributePayload `json:"properties"`
	Quantity     int                    `json:"quantity"`
	Price        string                 `json:"price"`
}

type shippingLinePayload struct {
	Code  string `json:"code"`
	Title string `json:"title"`
	Price string `json:"price"`
}

type discountCodePayload struct {
	Code   string `json:"code"`
	Amount string `json:"amount"`
	Type   string `json:"type"`
}

type webhookRegistrationRequest struct {
	Webhook webhookRegistrationPayload `json:"webhook"`
}

type webhookRegistrationPayload struct {
	Topic   string `json:"topic"`
	Address string `json:"address"`
	Format  string `json:"format"`
}

func (c *Client) resolveInstallation(ctx context.Context) (*shopifyport.Installation, error) {
	if c == nil || c.tokenResolver == nil {
		return nil, ErrTokenResolverRequired
	}
	shopDomain := shopifyport.ShopDomainFromContext(ctx)
	installation, err := c.tokenResolver.ResolveInstallation(ctx, shopDomain)
	if err != nil {
		return nil, err
	}
	if installation == nil {
		return nil, shopifyport.ErrInstallationNotFound
	}
	if strings.TrimSpace(installation.AccessToken) == "" {
		return nil, ErrAccessTokenRequired
	}

	return installation, nil
}

func (c *Client) doJSONWithToken(ctx context.Context, shopDomain string, accessToken string, method string, path string, requestBody any, response any) error {
	requestURL := c.buildAdminBaseURL(shopDomain) + path
	return c.doJSON(ctx, requestURL, accessToken, method, requestBody, response, shopDomain)
}

func (c *Client) doJSONAbsolute(ctx context.Context, requestURL string, accessToken string, method string, requestBody any, response any) error {
	return c.doJSON(ctx, requestURL, accessToken, method, requestBody, response, "")
}

func (c *Client) doJSON(ctx context.Context, requestURL string, accessToken string, method string, requestBody any, response any, rateLimitKey string) error {
	body, err := marshalRequest(requestBody)
	if err != nil {
		return err
	}

	for attempt := 0; ; attempt++ {
		if err := c.waitForAdminRateLimit(ctx, rateLimitKey); err != nil {
			return err
		}
		request, requestErr := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
		if requestErr != nil {
			return requestErr
		}
		request.Header.Set("Accept", "application/json")
		if strings.TrimSpace(accessToken) != "" {
			request.Header.Set("X-Shopify-Access-Token", strings.TrimSpace(accessToken))
		}
		if len(body) > 0 {
			request.Header.Set("Content-Type", "application/json")
		}

		responseValue, doErr := c.httpClient.Do(request)
		if doErr != nil {
			if attempt >= maxRetries || !isRetryableTransportError(doErr) {
				return doErr
			}
			if waitErr := waitWithContext(ctx, retryDelay(attempt)); waitErr != nil {
				return waitErr
			}
			continue
		}

		statusErr, retryAfter, readErr := handleResponse(responseValue, response)
		if readErr != nil {
			return readErr
		}
		if statusErr == nil {
			return nil
		}
		if attempt >= maxRetries || !isRetryableStatus(statusErr.Code) {
			return statusErr
		}

		waitFor := retryAfter
		if waitFor <= 0 {
			waitFor = retryDelay(attempt)
			if statusErr.Code == http.StatusTooManyRequests && waitFor < c.tooManyRequestsRetryDelay {
				waitFor = c.tooManyRequestsRetryDelay
			}
		}
		if waitErr := waitWithContext(ctx, waitFor); waitErr != nil {
			return waitErr
		}
	}
}

func (c *Client) waitForAdminRateLimit(ctx context.Context, shopDomain string) error {
	key := shopifyport.NormalizeShopDomain(shopDomain)
	if c == nil || c.rateLimitInterval <= 0 || strings.TrimSpace(key) == "" {
		return nil
	}

	c.rateMu.Lock()
	defer c.rateMu.Unlock()

	last := c.lastAdminRequestAt[key]
	waitFor := c.rateLimitInterval - time.Since(last)
	if waitFor > 0 {
		if err := waitWithContext(ctx, waitFor); err != nil {
			return err
		}
	}
	c.lastAdminRequestAt[key] = time.Now()
	return nil
}

func (c *Client) buildAdminBaseURL(shopDomain string) string {
	if strings.TrimSpace(c.baseURL) != "" {
		return strings.TrimRight(c.baseURL, "/")
	}

	return fmt.Sprintf("https://%s/admin/api/%s", shopifyport.NormalizeShopDomain(shopDomain), apiVersion)
}

func (c *Client) buildOAuthAccessTokenURL(shopDomain string) string {
	if strings.TrimSpace(c.baseURL) != "" {
		return strings.TrimRight(c.baseURL, "/") + "/admin/oauth/access_token"
	}

	return fmt.Sprintf("https://%s/admin/oauth/access_token", shopifyport.NormalizeShopDomain(shopDomain))
}

func handleResponse(httpResponse *http.Response, response any) (*statusError, time.Duration, error) {
	defer httpResponse.Body.Close()

	payload, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, 0, err
	}
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return &statusError{Code: httpResponse.StatusCode, Body: string(payload)}, parseRetryAfter(httpResponse.Header.Get("Retry-After")), nil
	}
	if response == nil || len(payload) == 0 {
		return nil, 0, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(response); err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func marshalRequest(requestBody any) ([]byte, error) {
	if requestBody == nil {
		return nil, nil
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func isRetryableTransportError(err error) bool {
	return err != nil
}

func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= http.StatusInternalServerError
}

func retryDelay(attempt int) time.Duration {
	base := 200 * time.Millisecond
	return time.Duration(attempt+1) * base
}

func waitWithContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func parseRetryAfter(value string) time.Duration {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	seconds, err := strconv.Atoi(trimmed)
	if err != nil || seconds <= 0 {
		return 0
	}

	return time.Duration(seconds) * time.Second
}

func normalizeCustomer(payload customerPayload) shopifyport.ShopifyCustomer {
	customer := shopifyport.ShopifyCustomer{
		ID:             extractID(payload.ID),
		Email:          strings.TrimSpace(payload.Email),
		FirstName:      strings.TrimSpace(payload.FirstName),
		LastName:       strings.TrimSpace(payload.LastName),
		Phone:          strings.TrimSpace(payload.Phone),
		Tags:           strings.TrimSpace(payload.Tags),
		Note:           strings.TrimSpace(payload.Note),
		NoteAttributes: normalizeNoteAttributes(payload.NoteAttributes),
		CreatedAt:      payload.CreatedAt.UTC(),
	}
	if payload.EmailMarketingConsent != nil {
		customer.EmailMarketingState = normalizeMarketingState(*payload.EmailMarketingConsent)
		customer.EmailMarketingConsentUpdatedAt = normalizeMarketingTime(*payload.EmailMarketingConsent)
	}
	if payload.SMSMarketingConsent != nil {
		customer.SMSMarketingState = normalizeMarketingState(*payload.SMSMarketingConsent)
		customer.SMSMarketingConsentUpdatedAt = normalizeMarketingTime(*payload.SMSMarketingConsent)
	}
	if payload.DefaultAddress != nil {
		address := normalizeAddress(*payload.DefaultAddress)
		customer.DefaultAddress = &address
	}

	return customer
}

func normalizeOrder(payload orderPayload) shopifyport.ShopifyOrder {
	order := shopifyport.ShopifyOrder{
		ID:                  extractID(payload.ID),
		Name:                strings.TrimSpace(payload.Name),
		ContactEmail:        strings.TrimSpace(payload.Email),
		FinancialStatus:     strings.TrimSpace(payload.FinancialStatus),
		FulfillmentStatus:   strings.TrimSpace(payload.FulfillmentStatus),
		Currency:            strings.TrimSpace(payload.Currency),
		TotalPrice:          strings.TrimSpace(payload.TotalPrice),
		TotalTax:            strings.TrimSpace(payload.TotalTax),
		TotalDiscounts:      strings.TrimSpace(payload.TotalDiscounts),
		Note:                strings.TrimSpace(payload.Note),
		Tags:                strings.TrimSpace(payload.Tags),
		CancelReason:        strings.TrimSpace(payload.CancelReason),
		CancelledAt:         payload.CancelledAt,
		PaymentGatewayNames: cloneStrings(payload.PaymentGatewayNames),
		NoteAttributes:      normalizeNoteAttributes(payload.NoteAttributes),
		LineItems:           normalizeLineItems(payload.LineItems),
		ShippingLines:       normalizeShippingLines(payload.ShippingLines),
		DiscountCodes:       normalizeDiscountCodes(payload.DiscountCodes),
		CreatedAt:           payload.CreatedAt.UTC(),
	}
	if payload.Customer != nil {
		customer := normalizeCustomer(*payload.Customer)
		order.Customer = &customer
	}
	if payload.BillingAddress != nil {
		address := normalizeAddress(*payload.BillingAddress)
		order.BillingAddress = &address
	}
	if payload.ShippingAddress != nil {
		address := normalizeAddress(*payload.ShippingAddress)
		order.ShippingAddress = &address
	}

	return order
}

func normalizeAddress(payload addressPayload) shopifyport.ShopifyAddress {
	return shopifyport.ShopifyAddress{
		FirstName: strings.TrimSpace(payload.FirstName),
		LastName:  strings.TrimSpace(payload.LastName),
		Company:   strings.TrimSpace(payload.Company),
		Address1:  strings.TrimSpace(payload.Address1),
		Address2:  strings.TrimSpace(payload.Address2),
		City:      strings.TrimSpace(payload.City),
		Province:  strings.TrimSpace(payload.Province),
		Country:   strings.TrimSpace(payload.Country),
		Zip:       strings.TrimSpace(payload.Zip),
		Phone:     strings.TrimSpace(payload.Phone),
	}
}

func normalizeNoteAttributes(values []noteAttributePayload) []shopifyport.ShopifyNoteAttribute {
	attributes := make([]shopifyport.ShopifyNoteAttribute, 0, len(values))
	for _, value := range values {
		attributes = append(attributes, shopifyport.ShopifyNoteAttribute{Name: strings.TrimSpace(value.Name), Value: strings.TrimSpace(value.Value)})
	}

	return attributes
}

func normalizeLineItems(values []lineItemPayload) []shopifyport.ShopifyLineItem {
	items := make([]shopifyport.ShopifyLineItem, 0, len(values))
	for _, value := range values {
		items = append(items, shopifyport.ShopifyLineItem{
			ID:                extractID(value.ID),
			SKU:               strings.TrimSpace(value.SKU),
			Title:             strings.TrimSpace(value.Title),
			VariantTitle:      strings.TrimSpace(value.VariantTitle),
			ProductID:         shopifyProductGID(value.ProductID),
			VariantID:         shopifyVariantGID(value.VariantID),
			MannaiahProductID: extractProperty(value.Properties, "mannaiah_product_id"),
			Quantity:          value.Quantity,
			Price:             strings.TrimSpace(value.Price),
		})
	}

	return items
}

func normalizeMarketingState(payload marketingConsentPayload) string {
	return strings.TrimSpace(preferNonEmpty(payload.MarketingState, payload.State))
}

func normalizeMarketingTime(payload marketingConsentPayload) *time.Time {
	if payload.ConsentUpdatedAt != nil && !payload.ConsentUpdatedAt.IsZero() {
		resolved := payload.ConsentUpdatedAt.UTC()
		return &resolved
	}
	if payload.UpdatedAt != nil && !payload.UpdatedAt.IsZero() {
		resolved := payload.UpdatedAt.UTC()
		return &resolved
	}
	return nil
}

func extractProperty(values []noteAttributePayload, key string) string {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value.Name), strings.TrimSpace(key)) {
			return strings.TrimSpace(value.Value)
		}
	}
	return ""
}

func shopifyProductGID(value any) string {
	id := extractID(value)
	if id == "" || strings.HasPrefix(id, "gid://") {
		return id
	}
	return "gid://shopify/Product/" + id
}

func shopifyVariantGID(value any) string {
	id := extractID(value)
	if id == "" || strings.HasPrefix(id, "gid://") {
		return id
	}
	return "gid://shopify/ProductVariant/" + id
}

func preferNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeShippingLines(values []shippingLinePayload) []shopifyport.ShopifyShippingLine {
	rows := make([]shopifyport.ShopifyShippingLine, 0, len(values))
	for _, value := range values {
		rows = append(rows, shopifyport.ShopifyShippingLine{Code: strings.TrimSpace(value.Code), Title: strings.TrimSpace(value.Title), Price: strings.TrimSpace(value.Price)})
	}

	return rows
}

func normalizeDiscountCodes(values []discountCodePayload) []shopifyport.ShopifyDiscountCode {
	rows := make([]shopifyport.ShopifyDiscountCode, 0, len(values))
	for _, value := range values {
		rows = append(rows, shopifyport.ShopifyDiscountCode{Code: strings.TrimSpace(value.Code), Amount: strings.TrimSpace(value.Amount), Type: strings.TrimSpace(value.Type)})
	}

	return rows
}

func extractID(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}

	return result
}
