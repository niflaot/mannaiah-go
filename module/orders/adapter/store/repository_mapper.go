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
	status := latestStatusEntry(order.StatusHistory)

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

// latestStatusEntry resolves the latest status entry from ordered status history values.
func latestStatusEntry(statuses []ordersdomain.StatusEntry) ordersdomain.StatusEntry {
	if len(statuses) == 0 {
		return ordersdomain.StatusEntry{
			Status:     ordersdomain.StatusCreated,
			Author:     "system",
			OccurredAt: time.Now().UTC(),
		}
	}

	return statuses[len(statuses)-1]
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
			Value:            item.Value,
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
			NoteOwner:   strings.TrimSpace(entry.NoteOwner),
			Note:        strings.TrimSpace(entry.Note),
			OccurredAt:  entry.OccurredAt.UTC(),
		})
	}

	return rows
}

// toShippingChargeRecords maps shipping charge values to storage rows.
func toShippingChargeRecords(orderID string, values []ordersdomain.ShippingCharge) []orderShippingChargeRecord {
	rows := make([]orderShippingChargeRecord, 0, len(values))
	for index, value := range values {
		rows = append(rows, orderShippingChargeRecord{
			OrderID:     strings.TrimSpace(orderID),
			Position:    index,
			MethodID:    strings.TrimSpace(value.MethodID),
			MethodTitle: strings.TrimSpace(value.MethodTitle),
			Price:       value.Price,
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
	shippingCharges []orderShippingChargeRecord,
	orderMetadata map[string]string,
) ordersdomain.Order {
	resolvedStatus := resolveCurrentStatus(record, statuses)
	entity := ordersdomain.Order{
		ID:              strings.TrimSpace(record.ID),
		Identifier:      strings.TrimSpace(record.Identifier),
		Realm:           strings.TrimSpace(record.Realm),
		ContactID:       strings.TrimSpace(record.ContactID),
		CurrentStatus:   resolvedStatus.Status,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
		StatusHistory:   toStatusEntries(statuses),
		Items:           toItemEntities(items),
		ShippingAddress: ordersdomain.ShippingAddress{},
		ShippingCharges: toShippingCharges(shippingCharges),
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

// resolveCurrentStatus resolves current status values from status history rows with root fallback values.
func resolveCurrentStatus(record orderRecord, statuses []orderStatusRecord) ordersdomain.StatusEntry {
	if len(statuses) > 0 {
		last := statuses[len(statuses)-1]
		return ordersdomain.StatusEntry{
			Status:      ordersdomain.Status(strings.TrimSpace(last.Status)),
			Author:      strings.TrimSpace(last.Author),
			Description: strings.TrimSpace(last.Description),
			NoteOwner:   strings.TrimSpace(last.NoteOwner),
			Note:        strings.TrimSpace(last.Note),
			OccurredAt:  last.OccurredAt.UTC(),
		}
	}

	return ordersdomain.StatusEntry{
		Status:      ordersdomain.Status(strings.TrimSpace(record.CurrentStatus)),
		Author:      strings.TrimSpace(record.CurrentStatusAuthor),
		Description: strings.TrimSpace(record.CurrentStatusDescription),
		OccurredAt:  record.CurrentStatusAt.UTC(),
	}
}

// toItemEntities maps storage item rows to order item aggregate values.
func toItemEntities(rows []orderItemRecord) []ordersdomain.Item {
	items := make([]ordersdomain.Item, 0, len(rows))
	for _, row := range rows {
		item := ordersdomain.Item{
			SKU:              strings.TrimSpace(row.SKU),
			AlternateName:    strings.TrimSpace(row.AlternateName),
			Quantity:         row.Quantity,
			Value:            row.Value,
			ResolutionSource: ordersdomain.ItemResolutionSource(strings.TrimSpace(row.ResolutionSource)),
		}
		if row.ProductID != nil {
			item.ProductID = strings.TrimSpace(*row.ProductID)
		}
		items = append(items, item)
	}

	return items
}

// toShippingCharges maps storage shipping-charge rows to order aggregate values.
func toShippingCharges(rows []orderShippingChargeRecord) []ordersdomain.ShippingCharge {
	if len(rows) == 0 {
		return nil
	}

	values := make([]ordersdomain.ShippingCharge, 0, len(rows))
	for _, row := range rows {
		values = append(values, ordersdomain.ShippingCharge{
			MethodID:    strings.TrimSpace(row.MethodID),
			MethodTitle: strings.TrimSpace(row.MethodTitle),
			Price:       row.Price,
		})
	}

	return values
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
			NoteOwner:   strings.TrimSpace(row.NoteOwner),
			Note:        strings.TrimSpace(row.Note),
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
