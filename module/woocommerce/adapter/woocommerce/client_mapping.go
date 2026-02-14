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
		if sku == "" {
			continue
		}

		metadata := map[string]string{}
		for _, meta := range value.MetaData {
			key := strings.TrimSpace(meta.Key)
			if key == "" {
				continue
			}
			metadata[key] = strings.TrimSpace(meta.Value)
		}
		if len(metadata) == 0 {
			metadata = nil
		}

		items = append(items, port.WooOrderItem{
			SKU:      sku,
			Name:     strings.TrimSpace(value.Name),
			Quantity: value.Quantity,
			Metadata: metadata,
		})
	}

	return items
}

// mapRawOrderItems maps raw line-item values to transport order item values.
func mapRawOrderItems(values []rawLineItem) []port.WooOrderItem {
	items := make([]port.WooOrderItem, 0, len(values))
	for _, value := range values {
		sku := strings.TrimSpace(value.SKU)
		if sku == "" {
			continue
		}

		metadata := map[string]string{}
		for _, meta := range value.MetaData {
			key := strings.TrimSpace(meta.Key)
			if key == "" {
				continue
			}
			metadata[key] = normalizeMetadataValue(meta.Value)
		}
		if len(metadata) == 0 {
			metadata = nil
		}

		items = append(items, port.WooOrderItem{
			SKU:      sku,
			Name:     strings.TrimSpace(value.Name),
			Quantity: value.Quantity,
			Metadata: metadata,
		})
	}

	return items
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
