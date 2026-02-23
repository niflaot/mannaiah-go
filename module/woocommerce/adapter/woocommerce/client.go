package woocommerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	wc "github.com/jmolboy/woocommerce-go"
	wcconfig "github.com/jmolboy/woocommerce-go/config"
	wcentity "github.com/jmolboy/woocommerce-go/entity"
	coretelemetry "mannaiah/module/core/telemetry"
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
	// _ ensures Client satisfies WooCommerce destination contracts.
	_ port.OrderDestination = (*Client)(nil)
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
	startedAt := time.Now()
	spanCtx, span := coretelemetry.StartSpan(ctx, "mannaiah/dependency", "woocommerce.validate")
	defer span.End()

	if err := ctx.Err(); err != nil {
		coretelemetry.RecordDependency("woocommerce", "validate", startedAt, err)
		return err
	}

	params := wc.OrdersQueryParams{}
	params.Page = 1
	params.PerPage = 1

	_, _, _, _, err := c.client.Services.Order.All(params)
	if err != nil {
		coretelemetry.RecordDependency("woocommerce", "validate", startedAt, err)
		return fmt.Errorf("validate woocommerce integration: %w", err)
	}

	finalErr := spanCtx.Err()
	coretelemetry.RecordDependency("woocommerce", "validate", startedAt, finalErr)
	return finalErr
}

// ListOrders retrieves paginated order values and reports whether additional pages exist.
func (c *Client) ListOrders(ctx context.Context, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error) {
	startedAt := time.Now()
	spanCtx, span := coretelemetry.StartSpan(ctx, "mannaiah/dependency", "woocommerce.list_orders")
	defer func() {
		coretelemetry.RecordDependency("woocommerce", "list_orders", startedAt, err)
		coretelemetry.EndSpan(span, err)
	}()

	if err := spanCtx.Err(); err != nil {
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

	result := mapSDKOrders(items)

	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	return result, resolveHasNextPage(page, pageSize, len(items), totalPages, isLastPage), nil
}

// SearchOrders retrieves paginated order values filtered by search terms.
func (c *Client) SearchOrders(ctx context.Context, search string, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error) {
	startedAt := time.Now()
	spanCtx, span := coretelemetry.StartSpan(ctx, "mannaiah/dependency", "woocommerce.search_orders")
	defer func() {
		coretelemetry.RecordDependency("woocommerce", "search_orders", startedAt, err)
		coretelemetry.EndSpan(span, err)
	}()

	if err := spanCtx.Err(); err != nil {
		return nil, false, err
	}

	params := wc.OrdersQueryParams{}
	params.Page = page
	params.PerPage = pageSize
	params.Order = wc.SortAsc
	params.OrderBy = "id"
	params.Search = strings.TrimSpace(search)

	items, _, totalPages, isLastPage, listErr := c.client.Services.Order.All(params)
	if listErr != nil {
		if shouldUseRawOrderFallback(listErr) {
			rawItems, rawHasNext, rawErr := c.listOrdersRawWithQuery(ctx, page, pageSize, params.Search)
			if rawErr == nil {
				return rawItems, rawHasNext, nil
			}

			return nil, false, fmt.Errorf(
				"search woocommerce orders: strict SDK decode failed (%s); raw fallback failed: %w",
				compactError(listErr, 280),
				rawErr,
			)
		}

		return nil, false, fmt.Errorf("search woocommerce orders: %w", listErr)
	}

	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	return mapSDKOrders(items), resolveHasNextPage(page, pageSize, len(items), totalPages, isLastPage), nil
}

// GetOrderByID retrieves one order by WooCommerce identifier values.
func (c *Client) GetOrderByID(ctx context.Context, orderID int) (order port.WooOrder, err error) {
	startedAt := time.Now()
	spanCtx, span := coretelemetry.StartSpan(ctx, "mannaiah/dependency", "woocommerce.get_order_by_id")
	defer func() {
		coretelemetry.RecordDependency("woocommerce", "get_order_by_id", startedAt, err)
		coretelemetry.EndSpan(span, err)
	}()

	if err := spanCtx.Err(); err != nil {
		return port.WooOrder{}, err
	}
	if orderID <= 0 {
		return port.WooOrder{}, errors.New("order id must be greater than zero")
	}

	item, lookupErr := c.client.Services.Order.One(orderID)
	if lookupErr != nil {
		if shouldUseRawOrderFallback(lookupErr) {
			rawOrder, rawErr := c.getOrderRaw(ctx, orderID)
			if rawErr == nil {
				return rawOrder, nil
			}

			return port.WooOrder{}, fmt.Errorf(
				"get woocommerce order by id: strict SDK decode failed (%s); raw fallback failed: %w",
				compactError(lookupErr, 280),
				rawErr,
			)
		}

		return port.WooOrder{}, fmt.Errorf("get woocommerce order by id: %w", lookupErr)
	}

	if err := spanCtx.Err(); err != nil {
		return port.WooOrder{}, err
	}

	return mapSDKOrder(item), nil
}

// mapSDKOrders maps Woo SDK order values to transport order values.
func mapSDKOrders(values []wcentity.Order) []port.WooOrder {
	result := make([]port.WooOrder, 0, len(values))
	for _, value := range values {
		result = append(result, mapSDKOrder(value))
	}

	return result
}

// mapSDKOrder maps one Woo SDK order value to transport order values.
func mapSDKOrder(item wcentity.Order) port.WooOrder {
	metadata := map[string]string{}
	for _, meta := range item.MetaData {
		key := strings.TrimSpace(meta.Key)
		if key == "" {
			continue
		}
		metadata[key] = strings.TrimSpace(meta.Value)
	}

	return port.WooOrder{
		ID:                     item.ID,
		Status:                 strings.TrimSpace(item.Status),
		BillingEmail:           strings.TrimSpace(item.Billing.Email),
		BillingFirstName:       strings.TrimSpace(item.Billing.FirstName),
		BillingLastName:        strings.TrimSpace(item.Billing.LastName),
		BillingCompany:         strings.TrimSpace(item.Billing.Company),
		BillingPhone:           strings.TrimSpace(item.Billing.Phone),
		BillingAddress1:        strings.TrimSpace(item.Billing.Address1),
		BillingAddress2:        strings.TrimSpace(item.Billing.Address2),
		BillingCity:            strings.TrimSpace(item.Billing.City),
		BillingAddressLine1:    strings.TrimSpace(item.Billing.Address1),
		BillingAddressLine2:    strings.TrimSpace(item.Billing.Address2),
		BillingCityCode:        strings.TrimSpace(item.Billing.City),
		BillingPhoneNormalized: strings.TrimSpace(item.Billing.Phone),
		ShippingFirstName:      strings.TrimSpace(item.Shipping.FirstName),
		ShippingLastName:       strings.TrimSpace(item.Shipping.LastName),
		ShippingCompany:        strings.TrimSpace(item.Shipping.Company),
		ShippingAddressLine1:   strings.TrimSpace(item.Shipping.Address1),
		ShippingAddressLine2:   strings.TrimSpace(item.Shipping.Address2),
		ShippingCityCode:       strings.TrimSpace(item.Shipping.City),
		Items:                  append(mapSDKOrderItems(item.LineItems), mapSDKFeeItems(item.FeeLines)...),
		ShippingCharges:        mapSDKShippingCharges(item.ShippingLines),
		Comments:               mapSDKOrderComments(item.CustomerNote, item.DateModified, item.DateCreated),
		CreatedAt:              parseWooOrderTime(item.DateCreated),
		Metadata:               metadata,
	}
}
