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
	"time"

	ordersdomain "mannaiah/module/orders/domain"
	shopifyport "mannaiah/module/shopify/port"
)

const (
	apiVersion     = "2026-01"
	defaultTimeout = 5 * time.Second
	maxRetries     = 2
)

var (
	// ErrDomainRequired is returned when the Shopify shop domain is empty.
	ErrDomainRequired = errors.New("shopify domain is required")
	// ErrAccessTokenRequired is returned when the Shopify access token is empty.
	ErrAccessTokenRequired = errors.New("shopify access token is required")
)

// Config defines Shopify Admin API client configuration values.
type Config struct {
	// Domain defines the Shopify store domain.
	Domain string
	// AccessToken defines private/admin access token values.
	AccessToken string
	// Timeout defines request timeout values.
	Timeout time.Duration
	// BaseURL overrides the computed Shopify Admin base URL for testing.
	BaseURL string
}

// Client defines the Shopify Admin REST adapter.
type Client struct {
	// baseURL defines Shopify Admin API base URLs.
	baseURL string
	// accessToken defines Admin API authentication tokens.
	accessToken string
	// httpClient defines HTTP transport dependencies.
	httpClient *http.Client
}

// NewClient creates Shopify Admin REST clients.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.Domain) == "" && strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, ErrDomainRequired
	}
	if strings.TrimSpace(cfg.AccessToken) == "" {
		return nil, ErrAccessTokenRequired
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s/admin/api/%s", normalizeDomain(cfg.Domain), apiVersion)
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL:     baseURL,
		accessToken: strings.TrimSpace(cfg.AccessToken),
		httpClient:  &http.Client{Timeout: timeout},
	}, nil
}

// Validate verifies connectivity and credentials against the Shopify Admin API.
func (c *Client) Validate(ctx context.Context) error {
	var response map[string]any
	return c.doJSON(ctx, http.MethodGet, "/shop.json", nil, &response)
}

// GetCustomer resolves one Shopify customer by identifier.
func (c *Client) GetCustomer(ctx context.Context, id string) (shopifyport.ShopifyCustomer, error) {
	var response customerResponse
	path := fmt.Sprintf("/customers/%s.json", url.PathEscape(strings.TrimSpace(id)))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		if statusErr := (*statusError)(nil); errors.As(err, &statusErr) && statusErr.Code == http.StatusNotFound {
			return shopifyport.ShopifyCustomer{}, shopifyport.ErrCustomerNotFound
		}
		return shopifyport.ShopifyCustomer{}, err
	}

	return normalizeCustomer(response.Customer), nil
}

// GetOrder resolves one Shopify order by identifier.
func (c *Client) GetOrder(ctx context.Context, id string) (shopifyport.ShopifyOrder, error) {
	var response orderResponse
	path := fmt.Sprintf("/orders/%s.json?status=any", url.PathEscape(strings.TrimSpace(id)))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		if statusErr := (*statusError)(nil); errors.As(err, &statusErr) && statusErr.Code == http.StatusNotFound {
			return shopifyport.ShopifyOrder{}, shopifyport.ErrOrderNotFound
		}
		return shopifyport.ShopifyOrder{}, err
	}

	return normalizeOrder(response.Order), nil
}

// UpdateOrderFromMainstream pushes one mainstream order-status update back to Shopify.
func (c *Client) UpdateOrderFromMainstream(ctx context.Context, shopifyID string, command shopifyport.MainstreamOrderUpdateCommand) error {
	order, err := c.GetOrder(ctx, shopifyID)
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
	if err := c.doJSON(ctx, http.MethodPut, path, requestBody, nil); err != nil {
		return err
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

type orderResponse struct {
	Order orderPayload `json:"order"`
}

type customerPayload struct {
	ID             any                    `json:"id"`
	Email          string                 `json:"email"`
	FirstName      string                 `json:"first_name"`
	LastName       string                 `json:"last_name"`
	Phone          string                 `json:"phone"`
	Tags           string                 `json:"tags"`
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

type updateOrderPayload struct {
	Note string `json:"note,omitempty"`
	Tags string `json:"tags,omitempty"`
}

func (c *Client) doJSON(ctx context.Context, method string, path string, requestBody any, response any) error {
	body, err := marshalRequest(requestBody)
	if err != nil {
		return err
	}

	requestURL := c.baseURL + path
	for attempt := 0; ; attempt++ {
		request, requestErr := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
		if requestErr != nil {
			return requestErr
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("X-Shopify-Access-Token", c.accessToken)
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
		}
		if waitErr := waitWithContext(ctx, waitFor); waitErr != nil {
			return waitErr
		}
	}
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

func normalizeDomain(domain string) string {
	trimmed := strings.TrimSpace(domain)
	trimmed = strings.TrimPrefix(trimmed, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	return strings.TrimRight(trimmed, "/")
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
	set := map[string]struct{}{}
	for _, value := range strings.Split(existing, ",") {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	if status == ordersdomain.StatusCompleted {
		set["mannaiah:completed"] = struct{}{}
	}

	ordered := make([]string, 0, len(set))
	for value := range set {
		ordered = append(ordered, value)
	}
	if len(ordered) == 0 {
		return ""
	}

	return strings.Join(ordered, ", ")
}
