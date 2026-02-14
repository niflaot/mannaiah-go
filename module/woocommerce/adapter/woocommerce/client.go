package woocommerce

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	wc "github.com/jmolboy/woocommerce-go"
	wcconfig "github.com/jmolboy/woocommerce-go/config"
	"mannaiah/module/woocommerce/port"
)

var (
	// ErrInvalidURL is returned when WooCommerce URL values are blank.
	ErrInvalidURL = errors.New("woocommerce url must not be empty")
	// ErrInvalidConsumerKey is returned when WooCommerce consumer keys are blank.
	ErrInvalidConsumerKey = errors.New("woocommerce consumer key must not be empty")
	// ErrInvalidConsumerSecret is returned when WooCommerce consumer secrets are blank.
	ErrInvalidConsumerSecret = errors.New("woocommerce consumer secret must not be empty")
)

// Config defines WooCommerce API client configuration values.
type Config struct {
	// URL defines WooCommerce store base URLs.
	URL string
	// ConsumerKey defines WooCommerce API consumer key values.
	ConsumerKey string
	// ConsumerSecret defines WooCommerce API consumer secret values.
	ConsumerSecret string
	// Timeout defines API request timeout values.
	Timeout time.Duration
	// VerifySSL controls TLS verification behavior.
	VerifySSL bool
}

// Client defines WooCommerce order source behavior.
type Client struct {
	// client defines underlying WooCommerce SDK clients.
	client *wc.WooCommerce
	// baseURL defines normalized WooCommerce base URL values.
	baseURL string
	// consumerKey defines WooCommerce API consumer key values.
	consumerKey string
	// consumerSecret defines WooCommerce API consumer secret values.
	consumerSecret string
	// timeout defines HTTP timeout values for raw-order fallback requests.
	timeout time.Duration
	// verifySSL controls TLS verification behavior for raw-order fallback requests.
	verifySSL bool
}

var (
	// _ ensures Client satisfies WooCommerce source contracts.
	_ port.OrderSource = (*Client)(nil)
)

// NewClient creates WooCommerce order source adapters.
func NewClient(cfg Config) (*Client, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	client := wc.NewClient(wcconfig.Config{
		URL:                    baseURL,
		Version:                "v3",
		ConsumerKey:            strings.TrimSpace(cfg.ConsumerKey),
		ConsumerSecret:         strings.TrimSpace(cfg.ConsumerSecret),
		AddAuthenticationToURL: false,
		Timeout:                timeout / time.Second,
		VerifySSL:              cfg.VerifySSL,
	})

	return &Client{
		client:         client,
		baseURL:        baseURL,
		consumerKey:    strings.TrimSpace(cfg.ConsumerKey),
		consumerSecret: strings.TrimSpace(cfg.ConsumerSecret),
		timeout:        timeout,
		verifySSL:      cfg.VerifySSL,
	}, nil
}

// Validate verifies source connectivity and credentials.
func (c *Client) Validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	params := wc.OrdersQueryParams{}
	params.Page = 1
	params.PerPage = 1

	_, _, _, _, err := c.client.Services.Order.All(params)
	if err != nil {
		return fmt.Errorf("validate woocommerce integration: %w", err)
	}

	return ctx.Err()
}

// ListOrders retrieves paginated order values and reports whether additional pages exist.
func (c *Client) ListOrders(ctx context.Context, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	params := wc.OrdersQueryParams{}
	params.Page = page
	params.PerPage = pageSize
	params.Order = wc.SortAsc
	params.OrderBy = "id"

	items, _, totalPages, isLastPage, listErr := c.client.Services.Order.All(params)
	if listErr != nil {
		if shouldUseRawOrderFallback(listErr) {
			rawItems, rawHasNext, rawErr := c.listOrdersRaw(ctx, page, pageSize)
			if rawErr == nil {
				return rawItems, rawHasNext, nil
			}

			return nil, false, fmt.Errorf(
				"list woocommerce orders: strict SDK decode failed (%s); raw fallback failed: %w",
				compactError(listErr, 280),
				rawErr,
			)
		}

		return nil, false, fmt.Errorf("list woocommerce orders: %w", listErr)
	}

	result := make([]port.WooOrder, 0, len(items))
	for _, item := range items {
		metadata := map[string]string{}
		for _, meta := range item.MetaData {
			key := strings.TrimSpace(meta.Key)
			if key == "" {
				continue
			}
			metadata[key] = strings.TrimSpace(meta.Value)
		}

		result = append(result, port.WooOrder{
			ID:               item.ID,
			BillingEmail:     strings.TrimSpace(item.Billing.Email),
			BillingFirstName: strings.TrimSpace(item.Billing.FirstName),
			BillingLastName:  strings.TrimSpace(item.Billing.LastName),
			BillingCompany:   strings.TrimSpace(item.Billing.Company),
			BillingPhone:     strings.TrimSpace(item.Billing.Phone),
			BillingAddress1:  strings.TrimSpace(item.Billing.Address1),
			BillingAddress2:  strings.TrimSpace(item.Billing.Address2),
			BillingCity:      strings.TrimSpace(item.Billing.City),
			CreatedAt:        parseWooOrderTime(item.DateCreated),
			Metadata:         metadata,
		})
	}

	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	return result, resolveHasNextPage(page, pageSize, len(items), totalPages, isLastPage), nil
}

// listOrdersRaw performs tolerant order decoding for metadata values unsupported by SDK structs.
func (c *Client) listOrdersRaw(ctx context.Context, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error) {
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(pageSize))
	query.Set("order", "asc")
	query.Set("orderby", "id")
	query.Set("consumer_key", c.consumerKey)
	query.Set("consumer_secret", c.consumerSecret)

	endpoint := c.baseURL + "/wp-json/wc/v3/orders?" + query.Encode()
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if requestErr != nil {
		return nil, false, fmt.Errorf("create raw orders request: %w", requestErr)
	}

	httpClient := &http.Client{
		Timeout: c.timeout,
	}
	if !c.verifySSL {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	response, responseErr := httpClient.Do(request)
	if responseErr != nil {
		return nil, false, fmt.Errorf("execute raw orders request: %w", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, false, fmt.Errorf("raw orders request returned status %d", response.StatusCode)
	}

	type rawMeta struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	type rawOrder struct {
		ID          int    `json:"id"`
		DateCreated string `json:"date_created"`
		Billing     struct {
			Email     string `json:"email"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Company   string `json:"company"`
			Phone     string `json:"phone"`
			Address1  string `json:"address_1"`
			Address2  string `json:"address_2"`
			City      string `json:"city"`
		} `json:"billing"`
		MetaData []rawMeta `json:"meta_data"`
	}

	var payload []rawOrder
	if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
		return nil, false, fmt.Errorf("decode raw orders response: %w", decodeErr)
	}

	result := make([]port.WooOrder, 0, len(payload))
	for _, item := range payload {
		metadata := map[string]string{}
		for _, meta := range item.MetaData {
			key := strings.TrimSpace(meta.Key)
			if key == "" {
				continue
			}
			metadata[key] = normalizeMetadataValue(meta.Value)
		}

		result = append(result, port.WooOrder{
			ID:               item.ID,
			BillingEmail:     strings.TrimSpace(item.Billing.Email),
			BillingFirstName: strings.TrimSpace(item.Billing.FirstName),
			BillingLastName:  strings.TrimSpace(item.Billing.LastName),
			BillingCompany:   strings.TrimSpace(item.Billing.Company),
			BillingPhone:     strings.TrimSpace(item.Billing.Phone),
			BillingAddress1:  strings.TrimSpace(item.Billing.Address1),
			BillingAddress2:  strings.TrimSpace(item.Billing.Address2),
			BillingCity:      strings.TrimSpace(item.Billing.City),
			CreatedAt:        parseWooOrderTime(item.DateCreated),
			Metadata:         metadata,
		})
	}

	totalPages, _ := strconv.Atoi(response.Header.Get("X-Wp-Totalpages"))
	isLastPage := page >= totalPages && totalPages > 0
	return result, resolveHasNextPage(page, pageSize, len(result), totalPages, isLastPage), nil
}

// shouldUseRawOrderFallback reports whether strict SDK decode failures should use tolerant raw decoding.
func shouldUseRawOrderFallback(err error) bool {
	if err == nil {
		return false
	}

	value := strings.ToLower(err.Error())
	markers := [...]string{
		"fuzzystringdecoder",
		"entity.order.meta",
		"entity.meta.value",
		"meta_data",
		"not number or string",
		"cannot unmarshal",
		"json:",
	}
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}

	return false
}

// normalizeMetadataValue converts dynamic metadata values to stable string representations.
func normalizeMetadataValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	case []any:
		if len(typed) == 1 {
			return normalizeMetadataValue(typed[0])
		}
		payload, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(payload)
	case map[string]any:
		payload, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(payload)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

// compactError normalizes and truncates error text for concise diagnostics.
func compactError(err error, limit int) string {
	if err == nil {
		return ""
	}

	value := strings.Join(strings.Fields(strings.TrimSpace(err.Error())), " ")
	if limit <= 0 || len(value) <= limit {
		return value
	}

	return value[:limit] + "..."
}

// parseWooOrderTime parses WooCommerce order date values.
func parseWooOrderTime(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}

	layouts := [...]string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UTC()
		}
	}

	return time.Time{}
}

// validateConfig validates WooCommerce client configuration values.
func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.URL) == "" {
		return ErrInvalidURL
	}
	if strings.TrimSpace(cfg.ConsumerKey) == "" {
		return ErrInvalidConsumerKey
	}
	if strings.TrimSpace(cfg.ConsumerSecret) == "" {
		return ErrInvalidConsumerSecret
	}

	return nil
}

// resolveHasNextPage resolves pagination continuation behavior from header and payload signals.
func resolveHasNextPage(page int, pageSize int, itemCount int, totalPages int, isLastPage bool) bool {
	if totalPages > 0 && page < totalPages {
		return true
	}

	if pageSize > 0 && itemCount >= pageSize {
		return true
	}

	return !isLastPage
}
