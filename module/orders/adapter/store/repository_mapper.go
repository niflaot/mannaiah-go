package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
)

// toOrderRecord maps order aggregate values to root storage records.
func toOrderRecord(order ordersdomain.Order) orderRecord {
	status := latestStatusEntry(order)

	return orderRecord{
		ID:                       strings.TrimSpace(order.ID),
		Identifier:               strings.TrimSpace(order.Identifier),
		Realm:                    strings.TrimSpace(order.Realm),
		ContactID:                strings.TrimSpace(order.ContactID),
		CurrentStatus:            strings.TrimSpace(string(order.CurrentStatus)),
		CurrentStatusAuthor:      strings.TrimSpace(status.Author),
		CurrentStatusDescription: strings.TrimSpace(status.Description),
		CurrentStatusAt:          status.OccurredAt,
		CreatedAt:                order.CreatedAt,
		UpdatedAt:                order.UpdatedAt,
	}
}

// latestStatusEntry resolves the latest status entry from order history values.
func latestStatusEntry(order ordersdomain.Order) ordersdomain.StatusEntry {
	if len(order.StatusHistory) == 0 {
		return ordersdomain.StatusEntry{
			Status:     order.CurrentStatus,
			Author:     "system",
			OccurredAt: time.Now().UTC(),
		}
	}

	return order.StatusHistory[len(order.StatusHistory)-1]
}

// toOrderItemRecords maps order items to storage child rows.
func toOrderItemRecords(orderID string, items []ordersdomain.Item) []orderItemRecord {
	rows := make([]orderItemRecord, 0, len(items))
	for index, item := range items {
		var productID *string
		if value := strings.TrimSpace(item.ProductID); value != "" {
			productID = &value
		}
		rows = append(rows, orderItemRecord{
			OrderID:          strings.TrimSpace(orderID),
			Position:         index,
			SKU:              strings.TrimSpace(item.SKU),
			AlternateName:    strings.TrimSpace(item.AlternateName),
			Quantity:         item.Quantity,
			ProductID:        productID,
			ResolutionSource: strings.TrimSpace(string(item.ResolutionSource)),
		})
	}

	return rows
}

// toOrderStatusRecords maps order status history values to storage child rows.
func toOrderStatusRecords(orderID string, statuses []ordersdomain.StatusEntry) []orderStatusRecord {
	rows := make([]orderStatusRecord, 0, len(statuses))
	for index, entry := range statuses {
		rows = append(rows, orderStatusRecord{
			OrderID:     strings.TrimSpace(orderID),
			Position:    index,
			Status:      strings.TrimSpace(string(entry.Status)),
			Author:      strings.TrimSpace(entry.Author),
			Description: strings.TrimSpace(entry.Description),
			OccurredAt:  entry.OccurredAt.UTC(),
		})
	}

	return rows
}

// toShippingRecord maps shipping address values to storage rows.
func toShippingRecord(orderID string, shipping ordersdomain.ShippingAddress) orderShippingAddressRecord {
	return orderShippingAddressRecord{
		OrderID:  strings.TrimSpace(orderID),
		Address:  strings.TrimSpace(shipping.Address),
		Address2: strings.TrimSpace(shipping.Address2),
		Phone:    strings.TrimSpace(shipping.Phone),
		CityCode: strings.TrimSpace(shipping.CityCode),
	}
}

// toOrderEntity maps root and child storage rows to order aggregate values.
func toOrderEntity(
	record orderRecord,
	items []orderItemRecord,
	statuses []orderStatusRecord,
	shipping *orderShippingAddressRecord,
	orderMetadata map[string]string,
	itemMetadata map[uint]map[string]string,
) ordersdomain.Order {
	entity := ordersdomain.Order{
		ID:              strings.TrimSpace(record.ID),
		Identifier:      strings.TrimSpace(record.Identifier),
		Realm:           strings.TrimSpace(record.Realm),
		ContactID:       strings.TrimSpace(record.ContactID),
		CurrentStatus:   ordersdomain.Status(strings.TrimSpace(record.CurrentStatus)),
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
		StatusHistory:   toStatusEntries(statuses),
		Items:           toItemEntities(items, itemMetadata),
		ShippingAddress: ordersdomain.ShippingAddress{},
		Metadata:        orderMetadata,
	}
	if shipping != nil {
		entity.HasCustomShippingAddress = true
		entity.ShippingAddress = ordersdomain.ShippingAddress{
			Address:  strings.TrimSpace(shipping.Address),
			Address2: strings.TrimSpace(shipping.Address2),
			Phone:    strings.TrimSpace(shipping.Phone),
			CityCode: strings.TrimSpace(shipping.CityCode),
		}
	}
	entity.Normalize()

	return entity
}

// toItemEntities maps storage item rows to order item aggregate values.
func toItemEntities(rows []orderItemRecord, metadata map[uint]map[string]string) []ordersdomain.Item {
	items := make([]ordersdomain.Item, 0, len(rows))
	for _, row := range rows {
		item := ordersdomain.Item{
			SKU:              strings.TrimSpace(row.SKU),
			AlternateName:    strings.TrimSpace(row.AlternateName),
			Quantity:         row.Quantity,
			ResolutionSource: ordersdomain.ItemResolutionSource(strings.TrimSpace(row.ResolutionSource)),
			Metadata:         metadata[row.ID],
		}
		if row.ProductID != nil {
			item.ProductID = strings.TrimSpace(*row.ProductID)
		}
		items = append(items, item)
	}

	return items
}

// toOrderMetadataRecords maps order metadata maps to storage rows.
func toOrderMetadataRecords(orderID string, metadata map[string]string) []orderMetadataRecord {
	keys := normalizedMetadataKeys(metadata)
	rows := make([]orderMetadataRecord, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, orderMetadataRecord{
			OrderID: strings.TrimSpace(orderID),
			Key:     key,
			Value:   strings.TrimSpace(metadata[key]),
		})
	}

	return rows
}

// toOrderItemMetadataRecords maps order-item metadata maps to storage rows.
func toOrderItemMetadataRecords(itemRows []orderItemRecord, items []ordersdomain.Item) []orderItemMetadataRecord {
	rows := make([]orderItemMetadataRecord, 0)
	for index := range itemRows {
		if index >= len(items) || itemRows[index].ID == 0 {
			continue
		}
		keys := normalizedMetadataKeys(items[index].Metadata)
		for _, key := range keys {
			rows = append(rows, orderItemMetadataRecord{
				OrderItemID: itemRows[index].ID,
				Key:         key,
				Value:       strings.TrimSpace(items[index].Metadata[key]),
			})
		}
	}

	return rows
}

// normalizedMetadataKeys normalizes metadata keys and returns sorted key values.
func normalizedMetadataKeys(metadata map[string]string) []string {
	if len(metadata) == 0 {
		return nil
	}

	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		keys = append(keys, trimmed)
	}
	sort.Strings(keys)
	return keys
}

// toStatusEntries maps storage status rows to order status aggregate values.
func toStatusEntries(rows []orderStatusRecord) []ordersdomain.StatusEntry {
	statuses := make([]ordersdomain.StatusEntry, 0, len(rows))
	for _, row := range rows {
		statuses = append(statuses, ordersdomain.StatusEntry{
			Status:      ordersdomain.Status(strings.TrimSpace(row.Status)),
			Author:      strings.TrimSpace(row.Author),
			Description: strings.TrimSpace(row.Description),
			OccurredAt:  row.OccurredAt,
		})
	}

	return statuses
}

// generateID creates random order identifier values.
func generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(bytes)
}
