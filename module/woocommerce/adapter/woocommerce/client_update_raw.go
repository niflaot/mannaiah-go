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
	"time"
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

	endpoint := c.baseURL + "/wp-json/wc/v3/products?" + query.Encode()
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if requestErr != nil {
		return 0, fmt.Errorf("create raw products request: %w", requestErr)
	}
	c.applyWooAuth(request)

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

// wooOrderUpdateState defines existing WooCommerce order rows used for idempotent update payload mapping.
type wooOrderUpdateState struct {
	// LineItems defines existing order line-item rows.
	LineItems []wooExistingLineItem
	// FeeLines defines existing order fee-line rows.
	FeeLines []wooExistingFeeLine
	// ShippingLines defines existing order shipping-line rows.
	ShippingLines []wooExistingShippingLine
}

// wooExistingLineItem defines existing WooCommerce line-item row values.
type wooExistingLineItem struct {
	// ID defines line-item identifier values.
	ID int
	// SKU defines line-item SKU values.
	SKU string
	// Name defines line-item display-name values.
	Name string
	// ProductID defines line-item product identifier values.
	ProductID int
}

// wooExistingFeeLine defines existing WooCommerce fee-line row values.
type wooExistingFeeLine struct {
	// ID defines fee-line identifier values.
	ID int
	// Name defines fee-line display-name values.
	Name string
}

// wooExistingShippingLine defines existing WooCommerce shipping-line row values.
type wooExistingShippingLine struct {
	// ID defines shipping-line identifier values.
	ID int
	// MethodID defines method identifier values.
	MethodID string
	// MethodTitle defines method display-title values.
	MethodTitle string
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

	endpoint := fmt.Sprintf("%s/wp-json/wc/v3/orders/%d", c.baseURL, orderID)

	request, requestErr := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if requestErr != nil {
		return fmt.Errorf("create raw order update request: %w", requestErr)
	}
	request.Header.Set("Content-Type", "application/json")
	c.applyWooAuth(request)

	httpClient := c.rawHTTPClient()
	if httpClient.Timeout < 20*time.Second {
		httpClient.Timeout = 20 * time.Second
	}
	response, responseErr := httpClient.Do(request)
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

// getOrderUpdateStateRaw retrieves current WooCommerce order rows required for idempotent updates.
func (c *Client) getOrderUpdateStateRaw(ctx context.Context, orderID int) (wooOrderUpdateState, error) {
	if err := ctx.Err(); err != nil {
		return wooOrderUpdateState{}, err
	}

	// Use full-order fetch for update-state matching. Nested _fields filtering is not consistently
	// supported across WooCommerce installations and may omit line-item identity fields.
	endpoint := fmt.Sprintf("%s/wp-json/wc/v3/orders/%d", c.baseURL, orderID)
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if requestErr != nil {
		return wooOrderUpdateState{}, fmt.Errorf("create raw order state request: %w", requestErr)
	}
	c.applyWooAuth(request)

	response, responseErr := c.rawHTTPClient().Do(request)
	if responseErr != nil {
		return wooOrderUpdateState{}, fmt.Errorf("execute raw order state request: %w", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(response.Body)
		message, code := parseWooErrorResponse(responseBody)
		return wooOrderUpdateState{}, &wooAPIError{
			StatusCode: response.StatusCode,
			Code:       code,
			Message:    message,
			Body:       compactError(fmt.Errorf("%s", strings.TrimSpace(string(responseBody))), 280),
		}
	}

	type rawOrderState struct {
		// LineItems defines existing order line-item values.
		LineItems []struct {
			ID int `json:"id"`
			// SKU defines existing line-item SKU values.
			SKU string `json:"sku"`
			// Name defines existing line-item name values.
			Name string `json:"name"`
			// ProductID defines existing line-item product-id values.
			ProductID int `json:"product_id"`
		} `json:"line_items"`
		// FeeLines defines existing order fee-line values.
		FeeLines []struct {
			ID int `json:"id"`
			// Name defines existing fee-line name values.
			Name string `json:"name"`
		} `json:"fee_lines"`
		// ShippingLines defines existing order shipping-line values.
		ShippingLines []struct {
			ID int `json:"id"`
			// MethodID defines existing shipping-line method identifier values.
			MethodID string `json:"method_id"`
			// MethodTitle defines existing shipping-line method title values.
			MethodTitle string `json:"method_title"`
		} `json:"shipping_lines"`
	}

	var payload rawOrderState
	if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
		return wooOrderUpdateState{}, fmt.Errorf("decode raw order state response: %w", decodeErr)
	}

	state := wooOrderUpdateState{
		LineItems:     make([]wooExistingLineItem, 0, len(payload.LineItems)),
		FeeLines:      make([]wooExistingFeeLine, 0, len(payload.FeeLines)),
		ShippingLines: make([]wooExistingShippingLine, 0, len(payload.ShippingLines)),
	}
	for _, row := range payload.LineItems {
		if row.ID <= 0 {
			continue
		}
		state.LineItems = append(state.LineItems, wooExistingLineItem{
			ID:        row.ID,
			SKU:       strings.TrimSpace(row.SKU),
			Name:      strings.TrimSpace(row.Name),
			ProductID: row.ProductID,
		})
	}
	for _, row := range payload.FeeLines {
		if row.ID <= 0 {
			continue
		}
		state.FeeLines = append(state.FeeLines, wooExistingFeeLine{
			ID:   row.ID,
			Name: strings.TrimSpace(row.Name),
		})
	}
	for _, row := range payload.ShippingLines {
		if row.ID <= 0 {
			continue
		}
		state.ShippingLines = append(state.ShippingLines, wooExistingShippingLine{
			ID:          row.ID,
			MethodID:    strings.TrimSpace(row.MethodID),
			MethodTitle: strings.TrimSpace(row.MethodTitle),
		})
	}

	return state, nil
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

// applyWooAuth applies WooCommerce authentication headers for raw endpoints.
func (c *Client) applyWooAuth(request *http.Request) {
	if request == nil {
		return
	}
	request.SetBasicAuth(c.consumerKey, c.consumerSecret)
}
