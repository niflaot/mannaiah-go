package woocommerce

import (
	"strings"
	"time"

	wcentity "github.com/jmolboy/woocommerce-go/entity"
	"mannaiah/module/woocommerce/port"
)

// mapSDKOrderItems maps SDK line-item values to transport order item values.
func mapSDKOrderItems(values []wcentity.LineItem) []port.WooOrderItem {
	items := make([]port.WooOrderItem, 0, len(values))
	for _, value := range values {
		sku := strings.TrimSpace(value.SKU)
		name := strings.TrimSpace(value.Name)
		if sku == "" && name == "" {
			continue
		}

		items = append(items, port.WooOrderItem{
			SKU:      sku,
			Name:     name,
			Quantity: int(value.Quantity),
			Value:    float64(value.Total),
		})
	}

	return items
}

// mapRawOrderItems maps raw line-item values to transport order item values.
func mapRawOrderItems(values []rawLineItem) []port.WooOrderItem {
	items := make([]port.WooOrderItem, 0, len(values))
	for _, value := range values {
		sku := strings.TrimSpace(value.SKU)
		name := strings.TrimSpace(value.Name)
		if sku == "" && name == "" {
			continue
		}

		items = append(items, port.WooOrderItem{
			SKU:      sku,
			Name:     name,
			Quantity: int(value.Quantity),
			Value:    float64(value.Total),
		})
	}

	return items
}

// mapSDKShippingCharges maps SDK shipping-line values to transport shipping charge values.
func mapSDKShippingCharges(values []wcentity.ShippingLine) []port.WooOrderShippingCharge {
	charges := make([]port.WooOrderShippingCharge, 0, len(values))
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodId)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		if methodID == "" && methodTitle == "" && value.Total == 0 {
			continue
		}
		charges = append(charges, port.WooOrderShippingCharge{
			MethodID:    methodID,
			MethodTitle: methodTitle,
			Price:       float64(value.Total),
		})
	}

	return charges
}

// mapRawShippingCharges maps raw shipping-line values to transport shipping charge values.
func mapRawShippingCharges(values []rawShippingLine) []port.WooOrderShippingCharge {
	charges := make([]port.WooOrderShippingCharge, 0, len(values))
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodID)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		if methodID == "" && methodTitle == "" && value.Total == 0 {
			continue
		}
		charges = append(charges, port.WooOrderShippingCharge{
			MethodID:    methodID,
			MethodTitle: methodTitle,
			Price:       float64(value.Total),
		})
	}

	return charges
}

// mapSDKOrderComments maps SDK customer-note values to transport order comment values.
func mapSDKOrderComments(customerNote string, dateModified string, dateCreated string) []port.WooOrderComment {
	description := strings.TrimSpace(customerNote)
	if description == "" {
		return nil
	}

	occurredAt := parseWooOrderTime(dateModified)
	if occurredAt.IsZero() {
		occurredAt = parseWooOrderTime(dateCreated)
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return []port.WooOrderComment{
		{
			Author:      "system",
			Description: description,
			OccurredAt:  occurredAt,
		},
	}
}

// mapRawOrderComments maps raw customer-note values to transport order comment values.
func mapRawOrderComments(customerNote string, dateModified string, dateCreated string) []port.WooOrderComment {
	return mapSDKOrderComments(customerNote, dateModified, dateCreated)
}
