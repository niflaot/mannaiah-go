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

	request, err := c.toUpdateOrderRawRequest(ctx, orderID, command)
	if err != nil {
		wrapped := fmt.Errorf("build woocommerce order update %d request: %w", orderID, err)
		if isWooClientError(err) {
			return messagingplatform.NonRetriable(wrapped)
		}
		return wrapped
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
func (c *Client) toUpdateOrderRawRequest(ctx context.Context, orderID int, command port.MainstreamOrderUpdateCommand) (map[string]any, error) {
	lineItems, feeLines, err := c.resolveOrderItemsForUpdate(ctx, command.Items)
	if err != nil {
		return nil, err
	}
	state := wooOrderUpdateState{}
	if len(lineItems) > 0 || len(feeLines) > 0 || len(command.ShippingCharges) > 0 {
		resolvedState, stateErr := c.getOrderUpdateStateRaw(ctx, orderID)
		if stateErr != nil {
			return nil, fmt.Errorf("resolve woocommerce order update state %d: %w", orderID, stateErr)
		}
		state = resolvedState
	}

	request := map[string]any{}

	mappedLineItems := mapLineItemsForRawUpdate(lineItems, state.LineItems)
	if len(mappedLineItems) > 0 {
		request["line_items"] = mappedLineItems
	}

	mappedShippingLines := mapShippingLinesForRawUpdate(mapShippingLinesForUpdate(command.ShippingCharges), state.ShippingLines)
	if len(mappedShippingLines) > 0 {
		request["shipping_lines"] = mappedShippingLines
	}

	mappedFeeLines := mapFeeLinesForRawUpdate(feeLines, state.FeeLines)
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
func mapLineItemsForRawUpdate(values []wcentity.LineItem, existing []wooExistingLineItem) []map[string]any {
	if len(values) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(values))
	usedIDs := map[int]struct{}{}
	for _, value := range mergeLineItems(values) {
		if value.ProductId <= 0 {
			continue
		}

		quantity := value.Quantity
		if quantity <= 0 {
			quantity = 1
		}

		if existingID, ok := matchExistingLineItemID(existing, usedIDs, value); ok {
			rows = append(rows, map[string]any{
				"id":       existingID,
				"quantity": quantity,
				"total":    formatWooDecimal(value.Total),
			})
		} else {
			rows = append(rows, map[string]any{
				"product_id": value.ProductId,
				"quantity":   quantity,
				"total":      formatWooDecimal(value.Total),
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// mapShippingLinesForRawUpdate maps SDK shipping-line values to WooCommerce raw update payload values.
func mapShippingLinesForRawUpdate(values []wcentity.ShippingLine, existing []wooExistingShippingLine) []map[string]any {
	if len(values) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(values))
	usedIDs := map[int]struct{}{}
	for _, value := range mergeShippingLines(values) {
		methodID := strings.TrimSpace(value.MethodId)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		if methodID == "" && methodTitle == "" {
			continue
		}

		if existingLine, ok := matchExistingShippingLine(existing, usedIDs, methodID, methodTitle); ok {
			if methodID == "" {
				methodID = strings.TrimSpace(existingLine.MethodID)
			}
			if methodTitle == "" {
				methodTitle = strings.TrimSpace(existingLine.MethodTitle)
			}
			if methodID == "" {
				continue
			}

			row := map[string]any{
				"id":        existingLine.ID,
				"method_id": methodID,
				"total":     formatWooDecimal(value.Total),
			}
			if methodTitle != "" {
				row["method_title"] = methodTitle
			}
			rows = append(rows, row)
		} else {
			if methodID == "" {
				continue
			}

			row := map[string]any{
				"method_id": methodID,
				"total":     formatWooDecimal(value.Total),
			}
			if methodTitle != "" {
				row["method_title"] = methodTitle
			}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// mapFeeLinesForRawUpdate maps SDK fee-line values to WooCommerce raw update payload values.
func mapFeeLinesForRawUpdate(values []wcentity.FeeLine, existing []wooExistingFeeLine) []map[string]any {
	if len(values) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(values))
	usedIDs := map[int]struct{}{}
	for _, value := range mergeFeeLines(values) {
		name := strings.TrimSpace(value.Name)
		if name == "" {
			continue
		}

		if existingID, ok := matchExistingFeeLineID(existing, usedIDs, name); ok {
			rows = append(rows, map[string]any{
				"id":    existingID,
				"name":  name,
				"total": formatWooDecimal(value.Total),
			})
		} else {
			rows = append(rows, map[string]any{
				"name":  name,
				"total": formatWooDecimal(value.Total),
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// matchExistingLineItemID resolves existing line-item ids by SKU, product id, or name values.
func matchExistingLineItemID(existing []wooExistingLineItem, usedIDs map[int]struct{}, value wcentity.LineItem) (int, bool) {
	normalizedSKU := strings.ToLower(strings.TrimSpace(value.SKU))
	normalizedName := strings.ToLower(strings.TrimSpace(value.Name))
	for _, row := range existing {
		if row.ID <= 0 {
			continue
		}
		if _, used := usedIDs[row.ID]; used {
			continue
		}
		if normalizedSKU != "" && strings.EqualFold(strings.TrimSpace(row.SKU), normalizedSKU) {
			usedIDs[row.ID] = struct{}{}
			return row.ID, true
		}
		if value.ProductId > 0 && row.ProductID == value.ProductId {
			usedIDs[row.ID] = struct{}{}
			return row.ID, true
		}
		if normalizedName != "" && strings.EqualFold(strings.TrimSpace(row.Name), normalizedName) {
			usedIDs[row.ID] = struct{}{}
			return row.ID, true
		}
	}

	return 0, false
}

// matchExistingShippingLine resolves existing shipping-line values by method id and title values.
func matchExistingShippingLine(existing []wooExistingShippingLine, usedIDs map[int]struct{}, methodID string, methodTitle string) (wooExistingShippingLine, bool) {
	normalizedMethodID := strings.ToLower(strings.TrimSpace(methodID))
	normalizedMethodTitle := strings.ToLower(strings.TrimSpace(methodTitle))
	for _, row := range existing {
		if row.ID <= 0 {
			continue
		}
		if _, used := usedIDs[row.ID]; used {
			continue
		}
		if normalizedMethodID != "" && strings.EqualFold(strings.TrimSpace(row.MethodID), normalizedMethodID) {
			usedIDs[row.ID] = struct{}{}
			return row, true
		}
		if normalizedMethodTitle != "" && strings.EqualFold(strings.TrimSpace(row.MethodTitle), normalizedMethodTitle) {
			usedIDs[row.ID] = struct{}{}
			return row, true
		}
	}

	return wooExistingShippingLine{}, false
}

// matchExistingFeeLineID resolves existing fee-line ids by name values.
func matchExistingFeeLineID(existing []wooExistingFeeLine, usedIDs map[int]struct{}, name string) (int, bool) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	for _, row := range existing {
		if row.ID <= 0 {
			continue
		}
		if _, used := usedIDs[row.ID]; used {
			continue
		}
		if normalized != "" && strings.EqualFold(strings.TrimSpace(row.Name), normalized) {
			usedIDs[row.ID] = struct{}{}
			return row.ID, true
		}
	}

	return 0, false
}

// mergeLineItems merges duplicate line-item rows by SKU/product key values.
func mergeLineItems(values []wcentity.LineItem) []wcentity.LineItem {
	if len(values) == 0 {
		return nil
	}

	merged := make([]wcentity.LineItem, 0, len(values))
	indexByKey := map[string]int{}
	for _, value := range values {
		sku := strings.TrimSpace(value.SKU)
		name := strings.TrimSpace(value.Name)
		key := ""
		switch {
		case sku != "":
			key = "sku:" + strings.ToLower(sku)
		case value.ProductId > 0:
			key = fmt.Sprintf("pid:%d", value.ProductId)
		case name != "":
			key = "name:" + strings.ToLower(name)
		default:
			continue
		}

		if index, ok := indexByKey[key]; ok {
			merged[index].Quantity += value.Quantity
			merged[index].Total += value.Total
			continue
		}

		indexByKey[key] = len(merged)
		merged = append(merged, value)
	}

	return merged
}

// mergeShippingLines merges duplicate shipping-line rows by method key values.
func mergeShippingLines(values []wcentity.ShippingLine) []wcentity.ShippingLine {
	if len(values) == 0 {
		return nil
	}

	merged := make([]wcentity.ShippingLine, 0, len(values))
	indexByKey := map[string]int{}
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodId)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		key := ""
		switch {
		case methodID != "":
			key = "mid:" + strings.ToLower(methodID)
		case methodTitle != "":
			key = "mtitle:" + strings.ToLower(methodTitle)
		default:
			continue
		}

		if index, ok := indexByKey[key]; ok {
			merged[index].Total += value.Total
			continue
		}

		indexByKey[key] = len(merged)
		merged = append(merged, value)
	}

	return merged
}

// mergeFeeLines merges duplicate fee-line rows by name values.
func mergeFeeLines(values []wcentity.FeeLine) []wcentity.FeeLine {
	if len(values) == 0 {
		return nil
	}

	merged := make([]wcentity.FeeLine, 0, len(values))
	indexByKey := map[string]int{}
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name == "" {
			continue
		}

		key := strings.ToLower(name)
		if index, ok := indexByKey[key]; ok {
			merged[index].Total += value.Total
			continue
		}

		indexByKey[key] = len(merged)
		merged = append(merged, value)
	}

	return merged
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
