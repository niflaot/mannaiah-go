package woocommerce

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// wooAPIError defines WooCommerce raw API error values.
type wooAPIError struct {
	// StatusCode defines HTTP status-code values.
	StatusCode int
	// Code defines WooCommerce error-code values.
	Code string
	// Message defines WooCommerce error message values.
	Message string
	// Body defines compact fallback response payload values.
	Body string
}

// Error returns formatted WooCommerce API error values.
func (e *wooAPIError) Error() string {
	parts := make([]string, 0, 3)
	if e.StatusCode > 0 {
		parts = append(parts, strconv.Itoa(e.StatusCode))
	}
	if strings.TrimSpace(e.Code) != "" {
		parts = append(parts, strings.TrimSpace(e.Code))
	}
	if strings.TrimSpace(e.Message) != "" {
		parts = append(parts, strings.TrimSpace(e.Message))
	}
	if strings.TrimSpace(e.Body) != "" {
		parts = append(parts, strings.TrimSpace(e.Body))
	}
	if len(parts) == 0 {
		return "woocommerce api error"
	}

	return strings.Join(parts, ": ")
}

// updateOrderRaw updates WooCommerce orders using tolerant raw endpoint payloads.
func (c *Client) updateOrderRaw(ctx context.Context, orderID int, payload map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	body, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return fmt.Errorf("marshal raw order update payload: %w", marshalErr)
	}

	query := url.Values{}
	query.Set("consumer_key", c.consumerKey)
	query.Set("consumer_secret", c.consumerSecret)
	endpoint := fmt.Sprintf("%s/wp-json/wc/v3/orders/%d?%s", c.baseURL, orderID, query.Encode())

	request, requestErr := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if requestErr != nil {
		return fmt.Errorf("create raw order update request: %w", requestErr)
	}
	request.Header.Set("Content-Type", "application/json")

	response, responseErr := c.rawHTTPClient().Do(request)
	if responseErr != nil {
		return fmt.Errorf("execute raw order update request: %w", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	responseBody, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return fmt.Errorf("read raw order update response body: %w", readErr)
	}

	message, code := parseWooErrorResponse(responseBody)
	return &wooAPIError{
		StatusCode: response.StatusCode,
		Code:       code,
		Message:    message,
		Body:       compactError(fmt.Errorf("%s", strings.TrimSpace(string(responseBody))), 280),
	}
}

// parseWooErrorResponse parses WooCommerce error payload values.
func parseWooErrorResponse(payload []byte) (message string, code string) {
	type errorPayload struct {
		// Code defines WooCommerce error-code values.
		Code string `json:"code"`
		// Message defines WooCommerce error message values.
		Message string `json:"message"`
	}
	var value errorPayload
	if err := json.Unmarshal(payload, &value); err != nil {
		return "", ""
	}

	return strings.TrimSpace(value.Message), strings.TrimSpace(value.Code)
}

// formatWooDecimal formats decimal values to WooCommerce-compatible string values.
func formatWooDecimal(value float64) string {
	if value < 0 {
		value = 0
	}

	return strconv.FormatFloat(value, 'f', 2, 64)
}
