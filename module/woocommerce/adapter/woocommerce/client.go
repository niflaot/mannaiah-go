package woocommerce

import (
	"context"
	"errors"
	"fmt"
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

	client := wc.NewClient(wcconfig.Config{
		URL:                    strings.TrimRight(strings.TrimSpace(cfg.URL), "/"),
		Version:                "v3",
		ConsumerKey:            strings.TrimSpace(cfg.ConsumerKey),
		ConsumerSecret:         strings.TrimSpace(cfg.ConsumerSecret),
		AddAuthenticationToURL: false,
		Timeout:                timeout / time.Second,
		VerifySSL:              cfg.VerifySSL,
	})

	return &Client{client: client}, nil
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

	items, _, _, isLastPage, listErr := c.client.Services.Order.All(params)
	if listErr != nil {
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
			BillingPhone:     strings.TrimSpace(item.Billing.Phone),
			BillingAddress1:  strings.TrimSpace(item.Billing.Address1),
			BillingAddress2:  strings.TrimSpace(item.Billing.Address2),
			BillingCity:      strings.TrimSpace(item.Billing.City),
			Metadata:         metadata,
		})
	}

	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	return result, !isLastPage, nil
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
