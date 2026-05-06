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

	ordersdomain "mannaiah/module/orders/domain"
	shopifyport "mannaiah/module/shopify/port"
)

const (
	apiVersion                    = "2026-04"
	defaultTimeout                = 5 * time.Second
	defaultAdminRateLimitInterval = 600 * time.Millisecond
	defaultTooManyRequestsDelay   = 1100 * time.Millisecond
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

// CreateCustomerFromMainstream creates one Shopify customer from mainstream contact values.
func (c *Client) CreateCustomerFromMainstream(ctx context.Context, command shopifyport.MainstreamCustomerUpsertCommand) (shopifyport.ShopifyCustomer, error) {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return shopifyport.ShopifyCustomer{}, err
	}

	requestBody := updateCustomerRequest{
		Customer: buildCustomerWritePayload(shopifyport.ShopifyCustomer{}, command),
	}
	var response customerResponse
	if err := c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodPost, "/customers.json", requestBody, &response); err != nil {
		return shopifyport.ShopifyCustomer{}, err
	}

	customer := normalizeCustomer(response.Customer)
	customer.ShopDomain = installation.ShopDomain
	return customer, nil
}

// UpdateCustomerFromMainstream updates one Shopify customer from mainstream contact values.
func (c *Client) UpdateCustomerFromMainstream(ctx context.Context, id string, command shopifyport.MainstreamCustomerUpsertCommand) error {
	customer, err := c.GetCustomer(ctx, id)
	if err != nil {
		return err
	}
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return err
	}

	requestBody := updateCustomerRequest{
		Customer: buildCustomerWritePayload(customer, command),
	}
	path := fmt.Sprintf("/customers/%s.json", url.PathEscape(strings.TrimSpace(id)))
	return c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodPut, path, requestBody, nil)
}

// UpdateCustomerTags merges one or more customer tags into Shopify.
func (c *Client) UpdateCustomerTags(ctx context.Context, id string, tags []string) error {
	customer, err := c.GetCustomer(ctx, id)
	if err != nil {
		return err
	}
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return err
	}

	requestBody := updateCustomerRequest{
		Customer: updateCustomerPayload{
			Tags: mergeTags(customer.Tags, tags),
		},
	}
	path := fmt.Sprintf("/customers/%s.json", url.PathEscape(strings.TrimSpace(id)))
	return c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodPut, path, requestBody, nil)
}

// AppendCustomerNote appends one customer note entry when it is not already present.
func (c *Client) AppendCustomerNote(ctx context.Context, id string, note string) error {
	customer, err := c.GetCustomer(ctx, id)
	if err != nil {
		return err
	}
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return err
	}

	requestBody := updateCustomerRequest{
		Customer: updateCustomerPayload{
			Note: appendNote(customer.Note, note),
		},
	}
	path := fmt.Sprintf("/customers/%s.json", url.PathEscape(strings.TrimSpace(id)))
	return c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodPut, path, requestBody, nil)
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

// CreateOrderFromMainstream creates one Shopify order from mainstream order values.
func (c *Client) CreateOrderFromMainstream(ctx context.Context, command shopifyport.MainstreamOrderCreateCommand) (shopifyport.ShopifyOrder, error) {
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return shopifyport.ShopifyOrder{}, err
	}

	requestBody := createOrderRequest{
		Order: buildOrderCreatePayload(command),
	}
	var response orderResponse
	if err := c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodPost, "/orders.json", requestBody, &response); err != nil {
		return shopifyport.ShopifyOrder{}, err
	}

	order := normalizeOrder(response.Order)
	order.ShopDomain = installation.ShopDomain
	if order.Customer != nil {
		order.Customer.ShopDomain = installation.ShopDomain
	}
	return order, nil
}

// UpdateOrderFromMainstream pushes one mainstream order-status update back to Shopify.
func (c *Client) UpdateOrderFromMainstream(ctx context.Context, shopifyID string, command shopifyport.MainstreamOrderUpdateCommand) error {
	order, err := c.GetOrder(ctx, shopifyID)
	if err != nil {
		return err
	}
	installation, err := c.resolveInstallation(ctx)
	if err != nil {
		return err
	}

	note := buildOutboundNote(command.Status)
	tags := buildOutboundTags(order.Tags, command.Status)
	requestBody := updateOrderRequest{
		Order: updateOrderPayload{
			Note: appendNote(order.Note, note),
			Tags: tags,
		},
	}

	path := fmt.Sprintf("/orders/%s.json", url.PathEscape(strings.TrimSpace(shopifyID)))
	if err := c.doJSONWithToken(ctx, installation.ShopDomain, installation.AccessToken, http.MethodPut, path, requestBody, nil); err != nil {
		return err
	}

	return nil
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

type statusError struct {
	Code int
	Body string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("shopify api returned status %d: %s", e.Code, strings.TrimSpace(e.Body))
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
	ID             any                    `json:"id"`
	Email          string                 `json:"email"`
	FirstName      string                 `json:"first_name"`
	LastName       string                 `json:"last_name"`
	Phone          string                 `json:"phone"`
	Tags           string                 `json:"tags"`
	Note           string                 `json:"note"`
	DefaultAddress *addressPayload        `json:"default_address"`
	NoteAttributes []noteAttributePayload `json:"note_attributes"`
	CreatedAt      time.Time              `json:"created_at"`
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

type lineItemPayload struct {
	SKU          string `json:"sku"`
	Title        string `json:"title"`
	VariantTitle string `json:"variant_title"`
	Quantity     int    `json:"quantity"`
	Price        string `json:"price"`
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

type updateOrderRequest struct {
	Order updateOrderPayload `json:"order"`
}

type createOrderRequest struct {
	Order createOrderPayload `json:"order"`
}

type createOrderPayload struct {
	Customer               *orderCustomerReferencePayload `json:"customer,omitempty"`
	FinancialStatus        string                         `json:"financial_status,omitempty"`
	InventoryBehaviour     string                         `json:"inventory_behaviour,omitempty"`
	LineItems              []createLineItemPayload        `json:"line_items,omitempty"`
	ShippingLines          []createShippingLinePayload    `json:"shipping_lines,omitempty"`
	Note                   string                         `json:"note,omitempty"`
	Tags                   string                         `json:"tags,omitempty"`
	CreatedAt              string                         `json:"created_at,omitempty"`
	SendReceipt            bool                           `json:"send_receipt"`
	SendFulfillmentReceipt bool                           `json:"send_fulfillment_receipt"`
}

type orderCustomerReferencePayload struct {
	ID any `json:"id"`
}

type createLineItemPayload struct {
	SKU      string `json:"sku,omitempty"`
	Title    string `json:"title,omitempty"`
	Quantity int    `json:"quantity,omitempty"`
	Price    string `json:"price,omitempty"`
}

type createShippingLinePayload struct {
	Code  string `json:"code,omitempty"`
	Title string `json:"title,omitempty"`
	Price string `json:"price,omitempty"`
}

type updateOrderPayload struct {
	Note string `json:"note,omitempty"`
	Tags string `json:"tags,omitempty"`
}

type updateCustomerRequest struct {
	Customer updateCustomerPayload `json:"customer"`
}

type updateCustomerPayload struct {
	ID        string           `json:"id,omitempty"`
	Email     string           `json:"email,omitempty"`
	FirstName string           `json:"first_name,omitempty"`
	LastName  string           `json:"last_name,omitempty"`
	Phone     string           `json:"phone,omitempty"`
	Note      string           `json:"note,omitempty"`
	Tags      string           `json:"tags,omitempty"`
	Addresses []addressPayload `json:"addresses,omitempty"`
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
			SKU:          strings.TrimSpace(value.SKU),
			Title:        strings.TrimSpace(value.Title),
			VariantTitle: strings.TrimSpace(value.VariantTitle),
			Quantity:     value.Quantity,
			Price:        strings.TrimSpace(value.Price),
		})
	}

	return items
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

func appendNote(existing string, entry string) string {
	trimmedExisting := strings.TrimSpace(existing)
	trimmedEntry := strings.TrimSpace(entry)
	if trimmedEntry == "" {
		return trimmedExisting
	}
	if strings.Contains(trimmedExisting, trimmedEntry) {
		return trimmedExisting
	}
	if trimmedExisting == "" {
		return trimmedEntry
	}

	return trimmedExisting + "\n" + trimmedEntry
}

func buildCustomerWritePayload(current shopifyport.ShopifyCustomer, command shopifyport.MainstreamCustomerUpsertCommand) updateCustomerPayload {
	payload := updateCustomerPayload{
		ID:        strings.TrimSpace(current.ID),
		Email:     strings.TrimSpace(command.Email),
		FirstName: resolveCustomerFirstName(command),
		LastName:  resolveCustomerLastName(command),
		Phone:     preferNonEmpty(command.Phone, current.Phone),
		Note:      appendNote(current.Note, buildCustomerContactNote(command.ContactID)),
		Tags:      mergeTags(current.Tags, []string{"mannaiah:synced"}),
		Addresses: buildCustomerAddresses(current, command),
	}
	if strings.TrimSpace(payload.Note) == "" {
		payload.Note = current.Note
	}
	if strings.TrimSpace(payload.Tags) == "" {
		payload.Tags = current.Tags
	}
	return payload
}

func buildCustomerAddresses(current shopifyport.ShopifyCustomer, command shopifyport.MainstreamCustomerUpsertCommand) []addressPayload {
	firstName := resolveCustomerFirstName(command)
	lastName := resolveCustomerLastName(command)
	address := addressPayload{
		FirstName: firstName,
		LastName:  lastName,
		Company:   strings.TrimSpace(command.DocumentNumber),
		Address1:  strings.TrimSpace(command.Address),
		Address2:  strings.TrimSpace(command.AddressExtra),
		City:      strings.TrimSpace(command.CityCode),
		Phone:     preferNonEmpty(command.Phone, current.Phone),
	}
	if current.DefaultAddress != nil {
		address.Province = strings.TrimSpace(current.DefaultAddress.Province)
		address.Country = strings.TrimSpace(current.DefaultAddress.Country)
		address.Zip = strings.TrimSpace(current.DefaultAddress.Zip)
	}
	if strings.TrimSpace(address.FirstName) == "" &&
		strings.TrimSpace(address.LastName) == "" &&
		strings.TrimSpace(address.Company) == "" &&
		strings.TrimSpace(address.Address1) == "" &&
		strings.TrimSpace(address.Address2) == "" &&
		strings.TrimSpace(address.City) == "" &&
		strings.TrimSpace(address.Phone) == "" &&
		strings.TrimSpace(address.Province) == "" &&
		strings.TrimSpace(address.Country) == "" &&
		strings.TrimSpace(address.Zip) == "" {
		return nil
	}

	return []addressPayload{address}
}

func buildCustomerContactNote(contactID string) string {
	trimmedID := strings.TrimSpace(contactID)
	if trimmedID == "" {
		return ""
	}
	return fmt.Sprintf("[Mannaiah] contact_id=%s", trimmedID)
}

func buildOrderCreatePayload(command shopifyport.MainstreamOrderCreateCommand) createOrderPayload {
	payload := createOrderPayload{
		Customer: &orderCustomerReferencePayload{
			ID: encodeShopifyID(command.CustomerID),
		},
		FinancialStatus:        mapOutboundFinancialStatus(command.Status),
		InventoryBehaviour:     "bypass",
		LineItems:              buildOrderCreateLineItems(command),
		ShippingLines:          buildOrderCreateShippingLines(command.ShippingCharges),
		Note:                   buildOrderCreateNote(command),
		Tags:                   mergeTags(buildOutboundTags("", command.Status), []string{"mannaiah:synced"}),
		SendReceipt:            false,
		SendFulfillmentReceipt: false,
	}
	if command.CreatedAt.IsZero() {
		return payload
	}
	payload.CreatedAt = command.CreatedAt.UTC().Format(time.RFC3339)
	return payload
}

func buildOrderCreateLineItems(command shopifyport.MainstreamOrderCreateCommand) []createLineItemPayload {
	items := make([]createLineItemPayload, 0, len(command.Items))
	for _, value := range command.Items {
		title := preferNonEmpty(value.Title, value.SKU, "Mannaiah item")
		quantity := value.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		items = append(items, createLineItemPayload{
			SKU:      strings.TrimSpace(value.SKU),
			Title:    strings.TrimSpace(title),
			Quantity: quantity,
			Price:    formatShopifyMoney(value.Price),
		})
	}
	if len(items) > 0 {
		return items
	}

	return []createLineItemPayload{{
		Title:    preferNonEmpty(command.Identifier, command.OrderID, "Mannaiah order"),
		Quantity: 1,
		Price:    "0.00",
	}}
}

func buildOrderCreateShippingLines(values []shopifyport.MainstreamOrderCreateShippingCharge) []createShippingLinePayload {
	lines := make([]createShippingLinePayload, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		title := strings.TrimSpace(value.Title)
		price := value.Price
		if code == "" && title == "" && price == 0 {
			continue
		}
		lines = append(lines, createShippingLinePayload{
			Code:  code,
			Title: preferNonEmpty(title, code, "Shipping"),
			Price: formatShopifyMoney(price),
		})
	}
	if len(lines) == 0 {
		return nil
	}

	return lines
}

func buildOrderCreateNote(command shopifyport.MainstreamOrderCreateCommand) string {
	parts := []string{}
	if strings.TrimSpace(command.OrderID) != "" {
		parts = append(parts, fmt.Sprintf("order_id=%s", strings.TrimSpace(command.OrderID)))
	}
	if strings.TrimSpace(command.Identifier) != "" {
		parts = append(parts, fmt.Sprintf("identifier=%s", strings.TrimSpace(command.Identifier)))
	}
	if strings.TrimSpace(string(command.Status)) != "" {
		parts = append(parts, fmt.Sprintf("status=%s", strings.TrimSpace(string(command.Status))))
	}
	if len(parts) == 0 {
		return "[Mannaiah] Order created from Mannaiah"
	}

	return "[Mannaiah] " + strings.Join(parts, " ")
}

func mapOutboundFinancialStatus(status ordersdomain.Status) string {
	switch status {
	case ordersdomain.StatusCompleted:
		return "paid"
	case ordersdomain.StatusCancelled:
		return "voided"
	default:
		return "pending"
	}
}

func formatShopifyMoney(value float64) string {
	if value < 0 {
		value = 0
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func encodeShopifyID(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if _, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return json.Number(trimmed)
	}
	return trimmed
}

func resolveCustomerFirstName(command shopifyport.MainstreamCustomerUpsertCommand) string {
	return preferNonEmpty(command.FirstName, command.LegalName, "Mannaiah")
}

func resolveCustomerLastName(command shopifyport.MainstreamCustomerUpsertCommand) string {
	return preferNonEmpty(command.LastName, "Contact")
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

func buildOutboundNote(status ordersdomain.Status) string {
	switch status {
	case ordersdomain.StatusCompleted:
		return "[Mannaiah] Order marked as completed"
	case ordersdomain.StatusCancelled:
		return "[Mannaiah] Order marked as cancelled"
	case ordersdomain.StatusHold:
		return "[Mannaiah] Order placed on hold"
	case ordersdomain.StatusPending:
		return "[Mannaiah] Order marked as pending"
	default:
		return "[Mannaiah] Order created in Mannaiah"
	}
}

func buildOutboundTags(existing string, status ordersdomain.Status) string {
	add, remove := outboundStatusTagTransition(status)
	return replaceTags(existing, add, remove)
}

func outboundStatusTagTransition(status ordersdomain.Status) (string, []string) {
	switch status {
	case ordersdomain.StatusCompleted:
		return "mannaiah:completed", []string{"mannaiah:pending", "mannaiah:hold"}
	case ordersdomain.StatusCancelled:
		return "mannaiah:cancelled", []string{"mannaiah:pending", "mannaiah:completed", "mannaiah:hold"}
	case ordersdomain.StatusPending:
		return "mannaiah:pending", nil
	case ordersdomain.StatusHold:
		return "mannaiah:hold", []string{"mannaiah:pending"}
	default:
		return "mannaiah:created", nil
	}
}

func mergeTags(existing string, tags []string) string {
	return replaceTags(existing, "", nil, tags...)
}

func replaceTags(existing string, add string, remove []string, extraAdds ...string) string {
	ordered := parseTags(existing)
	removed := map[string]struct{}{}
	for _, value := range remove {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		removed[trimmed] = struct{}{}
	}
	filtered := make([]string, 0, len(ordered)+len(extraAdds)+1)
	set := map[string]struct{}{}
	for _, value := range ordered {
		if _, shouldRemove := removed[value]; shouldRemove {
			continue
		}
		if _, exists := set[value]; exists {
			continue
		}
		set[value] = struct{}{}
		filtered = append(filtered, value)
	}
	for _, value := range append([]string{add}, extraAdds...) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := set[trimmed]; exists {
			continue
		}
		set[trimmed] = struct{}{}
		filtered = append(filtered, trimmed)
	}

	return strings.Join(filtered, ", ")
}

func parseTags(existing string) []string {
	values := strings.Split(existing, ",")
	ordered := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		ordered = append(ordered, trimmed)
	}
	return ordered
}
