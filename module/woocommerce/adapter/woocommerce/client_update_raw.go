package woocommerce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// resolveWooProductIDBySKURaw resolves WooCommerce product IDs by SKU values using tolerant raw endpoint decoding.
func (c *Client) resolveWooProductIDBySKURaw(ctx context.Context, sku string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	trimmedSKU := strings.TrimSpace(sku)
	if trimmedSKU == "" {
		return 0, nil
	}

	query := url.Values{}
	query.Set("page", "1")
	query.Set("per_page", "1")
	query.Set("sku", trimmedSKU)
	query.Set("_fields", "id,sku")
	query.Set("consumer_key", c.consumerKey)
	query.Set("consumer_secret", c.consumerSecret)

	endpoint := c.baseURL + "/wp-json/wc/v3/products?" + query.Encode()
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if requestErr != nil {
		return 0, fmt.Errorf("create raw products request: %w", requestErr)
	}

	response, responseErr := c.rawHTTPClient().Do(request)
	if responseErr != nil {
		return 0, fmt.Errorf("execute raw products request: %w", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("raw products request returned status %d", response.StatusCode)
	}

	type rawProduct struct {
		// ID defines WooCommerce product identifier values.
		ID int `json:"id"`
	}
	var payload []rawProduct
	if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
		return 0, fmt.Errorf("decode raw products response: %w", decodeErr)
	}
	if len(payload) == 0 {
		return 0, nil
	}
	if payload[0].ID <= 0 {
		return 0, fmt.Errorf("decode raw products response id %d", payload[0].ID)
	}

	return payload[0].ID, nil
}

// parseWooOrderID parses WooCommerce numeric order identifiers.
func parseWooOrderID(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("resolve woocommerce order id: empty identifier")
	}

	id, err := strconv.Atoi(trimmed)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("resolve woocommerce order id from identifier %q", trimmed)
	}

	return id, nil
}
