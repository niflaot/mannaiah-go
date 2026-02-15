package woocommerce

import (
	"context"
	"errors"
	"fmt"
	"strings"

	wcentity "github.com/jmolboy/woocommerce-go/entity"
	messagingplatform "mannaiah/module/core/messaging/platform"
	"mannaiah/module/woocommerce/port"
)

// UpdateOrderFromMainstream updates WooCommerce order mutable values from mainstream-origin payloads.
func (c *Client) UpdateOrderFromMainstream(ctx context.Context, command port.MainstreamOrderUpdateCommand) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	orderID, err := parseWooOrderID(command.Identifier)
	if err != nil {
		return err
	}

	request, err := c.toUpdateOrderRawRequest(ctx, command)
	if err != nil {
		return err
	}
	if len(request) == 0 {
		return nil
	}

	if err := c.updateOrderRaw(ctx, orderID, request); err != nil {
		wrapped := fmt.Errorf("update woocommerce order %d: %w", orderID, err)
		if isWooClientError(err) {
			return messagingplatform.NonRetriable(wrapped)
		}
		return wrapped
	}
	return nil
}

// toUpdateOrderRawRequest maps mainstream order update commands to WooCommerce raw update payloads.
func (c *Client) toUpdateOrderRawRequest(ctx context.Context, command port.MainstreamOrderUpdateCommand) (map[string]any, error) {
	lineItems, feeLines, err := c.resolveOrderItemsForUpdate(ctx, command.Items)
	if err != nil {
		return nil, err
	}

	request := map[string]any{}

	mappedLineItems := mapLineItemsForRawUpdate(lineItems)
	if len(mappedLineItems) > 0 {
		request["line_items"] = mappedLineItems
	}

	mappedShippingLines := mapShippingLinesForRawUpdate(mapShippingLinesForUpdate(command.ShippingCharges))
	if len(mappedShippingLines) > 0 {
		request["shipping_lines"] = mappedShippingLines
	}

	mappedFeeLines := mapFeeLinesForRawUpdate(feeLines)
	if len(mappedFeeLines) > 0 {
		request["fee_lines"] = mappedFeeLines
	}

	if command.ShippingAddress != nil {
		shipping := mapShippingAddressForUpdate(*command.ShippingAddress)
		request["shipping"] = map[string]any{
			"address_1": strings.TrimSpace(shipping.Address1),
			"address_2": strings.TrimSpace(shipping.Address2),
			"city":      strings.TrimSpace(shipping.City),
		}

		billing := mapBillingAddressForUpdate(*command.ShippingAddress)
		request["billing"] = map[string]any{
			"address_1": strings.TrimSpace(billing.Address1),
			"address_2": strings.TrimSpace(billing.Address2),
			"city":      strings.TrimSpace(billing.City),
			"phone":     strings.TrimSpace(billing.Phone),
		}
	}

	return request, nil
}

// resolveOrderItemsForUpdate resolves line-item and fee-line payload values.
func (c *Client) resolveOrderItemsForUpdate(ctx context.Context, items []port.OrderSyncItem) ([]wcentity.LineItem, []wcentity.FeeLine, error) {
	if len(items) == 0 {
		return nil, nil, nil
	}

	lineItems := make([]wcentity.LineItem, 0, len(items))
	feeLines := make([]wcentity.FeeLine, 0, len(items))
	productIDsBySKU := map[string]int{}

	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		name := strings.TrimSpace(item.Name)
		quantity := item.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		value := item.Value
		if value < 0 {
			value = 0
		}

		productID := 0
		if sku != "" {
			if resolved, ok := productIDsBySKU[sku]; ok {
				productID = resolved
			} else {
				resolved, resolveErr := c.resolveWooProductIDBySKU(ctx, sku)
				if resolveErr != nil {
					return nil, nil, resolveErr
				}
				productIDsBySKU[sku] = resolved
				productID = resolved
			}
		}
		if productID > 0 {
			lineItems = append(lineItems, wcentity.LineItem{
				ProductId: productID,
				Quantity:  quantity,
				Total:     value,
				SKU:       sku,
			})
			continue
		}

		fallbackName := name
		if fallbackName == "" {
			fallbackName = sku
		}
		if fallbackName == "" {
			continue
		}

		feeLines = append(feeLines, wcentity.FeeLine{
			Name:  fallbackName,
			Total: value,
		})
	}

	return lineItems, feeLines, nil
}

// resolveWooProductIDBySKU resolves WooCommerce product IDs by SKU values.
func (c *Client) resolveWooProductIDBySKU(ctx context.Context, sku string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	resolvedID, err := c.resolveWooProductIDBySKURaw(ctx, strings.TrimSpace(sku))
	if err != nil {
		return 0, fmt.Errorf("resolve woocommerce product by sku %q: %w", sku, err)
	}

	return resolvedID, nil
}

// isWooClientError reports whether WooCommerce client errors are HTTP 4xx and should not be retried.
func isWooClientError(err error) bool {
	var apiErr *wooAPIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 400 && apiErr.StatusCode < 500
	}

	return false
}

// mapLineItemsForRawUpdate maps SDK line-item values to WooCommerce raw update payload values.
func mapLineItemsForRawUpdate(values []wcentity.LineItem) []map[string]any {
	if len(values) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if value.ProductId <= 0 {
			continue
		}

		quantity := value.Quantity
		if quantity <= 0 {
			quantity = 1
		}

		rows = append(rows, map[string]any{
			"product_id": value.ProductId,
			"quantity":   quantity,
			"total":      formatWooDecimal(value.Total),
		})
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// mapShippingLinesForRawUpdate maps SDK shipping-line values to WooCommerce raw update payload values.
func mapShippingLinesForRawUpdate(values []wcentity.ShippingLine) []map[string]any {
	if len(values) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(values))
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodId)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		if methodID == "" && methodTitle == "" {
			continue
		}

		rows = append(rows, map[string]any{
			"method_id":    methodID,
			"method_title": methodTitle,
			"total":        formatWooDecimal(value.Total),
		})
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// mapFeeLinesForRawUpdate maps SDK fee-line values to WooCommerce raw update payload values.
func mapFeeLinesForRawUpdate(values []wcentity.FeeLine) []map[string]any {
	if len(values) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name == "" {
			continue
		}

		rows = append(rows, map[string]any{
			"name":  name,
			"total": formatWooDecimal(value.Total),
		})
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// mapShippingLinesForUpdate maps shipping charge payload values to WooCommerce shipping-line values.
func mapShippingLinesForUpdate(values []port.OrderSyncShippingCharge) []wcentity.ShippingLine {
	if len(values) == 0 {
		return nil
	}

	lines := make([]wcentity.ShippingLine, 0, len(values))
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodID)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		price := value.Price
		if price < 0 {
			price = 0
		}
		if methodID == "" && methodTitle == "" && price == 0 {
			continue
		}
		lines = append(lines, wcentity.ShippingLine{
			MethodId:    methodID,
			MethodTitle: methodTitle,
			Total:       price,
		})
	}
	if len(lines) == 0 {
		return nil
	}

	return lines
}

// mapShippingAddressForUpdate maps shipping-address payload values to WooCommerce shipping values.
func mapShippingAddressForUpdate(value port.OrderSyncShippingAddress) wcentity.Shipping {
	return wcentity.Shipping{
		Address1: strings.TrimSpace(value.Address),
		Address2: strings.TrimSpace(value.Address2),
		City:     strings.TrimSpace(value.CityCode),
	}
}

// mapBillingAddressForUpdate maps shipping-address payload values to WooCommerce billing values.
func mapBillingAddressForUpdate(value port.OrderSyncShippingAddress) wcentity.Billing {
	return wcentity.Billing{
		Address1: strings.TrimSpace(value.Address),
		Address2: strings.TrimSpace(value.Address2),
		City:     strings.TrimSpace(value.CityCode),
		Phone:    strings.TrimSpace(value.Phone),
	}
}
