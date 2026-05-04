package woocommerce

import (
	"fmt"
	"strings"

	wcentity "github.com/jmolboy/woocommerce-go/entity"
	"mannaiah/module/woocommerce/port"
)

// mapLineItemsForRawUpdate maps SDK line-item values to WooCommerce raw update payload values.
func mapLineItemsForRawUpdate(values []wcentity.LineItem, existing []wooExistingLineItem) []map[string]any {
	if len(values) == 0 && len(existing) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(values)+len(existing))
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
			continue
		}

		rows = append(rows, map[string]any{
			"product_id": value.ProductId,
			"quantity":   quantity,
			"total":      formatWooDecimal(value.Total),
		})
	}

	// Remove stale rows so Mannaiah remains the source of truth for order items.
	for _, row := range existing {
		if row.ID <= 0 {
			continue
		}
		if _, used := usedIDs[row.ID]; used {
			continue
		}
		rows = append(rows, map[string]any{
			"id":       row.ID,
			"quantity": 0,
			"total":    formatWooDecimal(0),
		})
	}

	if len(rows) == 0 {
		return nil
	}

	return rows
}

// mapShippingLinesForRawUpdate maps SDK shipping-line values to WooCommerce raw update payload values.
func mapShippingLinesForRawUpdate(values []wcentity.ShippingLine, existing []wooExistingShippingLine) []map[string]any {
	if len(values) == 0 && len(existing) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(values)+len(existing))
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
			continue
		}

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

	// Remove stale shipping lines while keeping required method identifiers.
	for _, row := range existing {
		if row.ID <= 0 {
			continue
		}
		if _, used := usedIDs[row.ID]; used {
			continue
		}
		methodID := strings.TrimSpace(row.MethodID)
		if methodID == "" {
			continue
		}

		deleteRow := map[string]any{
			"id":        row.ID,
			"method_id": methodID,
			"total":     formatWooDecimal(0),
		}
		if strings.TrimSpace(row.MethodTitle) != "" {
			deleteRow["method_title"] = strings.TrimSpace(row.MethodTitle)
		}
		rows = append(rows, deleteRow)
	}

	if len(rows) == 0 {
		return nil
	}

	return rows
}

// mapFeeLinesForRawUpdate maps SDK fee-line values to WooCommerce raw update payload values.
func mapFeeLinesForRawUpdate(values []wcentity.FeeLine, existing []wooExistingFeeLine) []map[string]any {
	if len(values) == 0 && len(existing) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(values)+len(existing))
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
			continue
		}

		rows = append(rows, map[string]any{
			"name":  name,
			"total": formatWooDecimal(value.Total),
		})
	}

	// Remove stale fee lines.
	for _, row := range existing {
		if row.ID <= 0 {
			continue
		}
		if _, used := usedIDs[row.ID]; used {
			continue
		}
		rows = append(rows, map[string]any{
			"id":    row.ID,
			"total": formatWooDecimal(0),
		})
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
