package store

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"gorm.io/gorm"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

// loadRelationsByOrderIDs loads order child rows grouped by order identifiers.
func loadRelationsByOrderIDs(ctx context.Context, db *gorm.DB, orderIDs []string) (
	map[string][]orderItemRecord,
	map[string][]orderStatusRecord,
	map[string]orderShippingAddressRecord,
	map[string][]orderShippingChargeRecord,
	map[string]map[string]string,
	error,
) {
	itemMap, err := loadOrderItemsByOrderIDs(ctx, db, orderIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	statusMap, err := loadOrderStatusesByOrderIDs(ctx, db, orderIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	shippingMap, err := loadShippingByOrderIDs(ctx, db, orderIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	shippingChargeMap, err := loadShippingChargesByOrderIDs(ctx, db, orderIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	orderMetadataMap, err := loadOrderMetadataByOrderIDs(ctx, db, orderIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return itemMap, statusMap, shippingMap, shippingChargeMap, orderMetadataMap, nil
}

// loadOrderItemsByOrderIDs loads order-item rows grouped by order identifiers.
func loadOrderItemsByOrderIDs(ctx context.Context, db *gorm.DB, orderIDs []string) (map[string][]orderItemRecord, error) {
	result := make(map[string][]orderItemRecord, len(orderIDs))
	if len(orderIDs) == 0 {
		return result, nil
	}

	rows := make([]orderItemRecord, 0)
	if err := db.WithContext(ctx).
		Model(&orderItemRecord{}).
		Where("order_id IN ?", orderIDs).
		Order("order_id ASC, position ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list order item records: %w", err)
	}
	for _, row := range rows {
		result[row.OrderID] = append(result[row.OrderID], row)
	}

	return result, nil
}

// loadOrderStatusesByOrderIDs loads order-status rows grouped by order identifiers.
func loadOrderStatusesByOrderIDs(ctx context.Context, db *gorm.DB, orderIDs []string) (map[string][]orderStatusRecord, error) {
	result := make(map[string][]orderStatusRecord, len(orderIDs))
	if len(orderIDs) == 0 {
		return result, nil
	}

	rows := make([]orderStatusRecord, 0)
	if err := db.WithContext(ctx).
		Model(&orderStatusRecord{}).
		Where("order_id IN ?", orderIDs).
		Order("order_id ASC, occurred_at ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list order status records: %w", err)
	}
	for _, row := range rows {
		result[row.OrderID] = append(result[row.OrderID], row)
	}

	return result, nil
}

// loadShippingByOrderIDs loads shipping rows grouped by order identifiers.
func loadShippingByOrderIDs(ctx context.Context, db *gorm.DB, orderIDs []string) (map[string]orderShippingAddressRecord, error) {
	result := make(map[string]orderShippingAddressRecord, len(orderIDs))
	if len(orderIDs) == 0 {
		return result, nil
	}

	rows := make([]orderShippingAddressRecord, 0)
	if err := db.WithContext(ctx).
		Model(&orderShippingAddressRecord{}).
		Where("order_id IN ?", orderIDs).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list order shipping records: %w", err)
	}
	for _, row := range rows {
		result[row.OrderID] = row
	}

	return result, nil
}

// loadShippingChargesByOrderIDs loads shipping-charge rows grouped by order identifiers.
func loadShippingChargesByOrderIDs(ctx context.Context, db *gorm.DB, orderIDs []string) (map[string][]orderShippingChargeRecord, error) {
	result := make(map[string][]orderShippingChargeRecord, len(orderIDs))
	if len(orderIDs) == 0 {
		return result, nil
	}

	rows := make([]orderShippingChargeRecord, 0)
	if err := db.WithContext(ctx).
		Model(&orderShippingChargeRecord{}).
		Where("order_id IN ?", orderIDs).
		Order("order_id ASC, position ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list order shipping charge records: %w", err)
	}
	for _, row := range rows {
		result[row.OrderID] = append(result[row.OrderID], row)
	}

	return result, nil
}

// loadOrderMetadataByOrderIDs loads order metadata rows grouped by order identifiers.
func loadOrderMetadataByOrderIDs(ctx context.Context, db *gorm.DB, orderIDs []string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string, len(orderIDs))
	if len(orderIDs) == 0 {
		return result, nil
	}

	rows := make([]orderMetadataRecord, 0)
	if err := db.WithContext(ctx).
		Model(&orderMetadataRecord{}).
		Where("order_id IN ?", orderIDs).
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list order metadata records: %w", err)
	}

	for _, orderID := range orderIDs {
		result[orderID] = map[string]string{}
	}
	for _, row := range rows {
		group := result[row.OrderID]
		if group == nil {
			group = map[string]string{}
		}
		group[row.Key] = row.Value
		result[row.OrderID] = group
	}

	return result, nil
}

// applyListQuery applies list filters over root order storage queries.
func applyListQuery(tx *gorm.DB, query ordersport.ListQuery) *gorm.DB {
	next := tx
	if value := strings.TrimSpace(query.Realm); value != "" {
		next = next.Where("realm = ?", value)
	}
	if value := strings.TrimSpace(query.ContactID); value != "" {
		next = next.Where("contact_id = ?", value)
	}
	if value := strings.TrimSpace(query.Identifier); value != "" {
		next = next.Where("identifier = ?", value)
	}
	if value := strings.TrimSpace(string(query.Status)); value != "" {
		next = next.Where(
			`EXISTS (
					SELECT 1
					FROM order_status_history AS osh
					WHERE osh.order_id = orders.id
						AND osh.id = (
							SELECT inner_osh.id
							FROM order_status_history AS inner_osh
							WHERE inner_osh.order_id = orders.id
							ORDER BY inner_osh.occurred_at DESC, inner_osh.id DESC
							LIMIT 1
						)
						AND osh.status = ?
				)`,
			value,
		)
	}

	return next
}

// collectOrderIDs collects ordered root identifiers from root rows.
func collectOrderIDs(rows []orderRecord) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids
}

// mapRowsToEntities maps root rows and grouped child rows to aggregate values.
func mapRowsToEntities(
	rows []orderRecord,
	itemMap map[string][]orderItemRecord,
	statusMap map[string][]orderStatusRecord,
	shippingMap map[string]orderShippingAddressRecord,
	shippingChargeMap map[string][]orderShippingChargeRecord,
	orderMetadataMap map[string]map[string]string,
) []ordersdomain.Order {
	entities := make([]ordersdomain.Order, 0, len(rows))
	for _, row := range rows {
		var shipping *orderShippingAddressRecord
		if value, ok := shippingMap[row.ID]; ok {
			copyValue := value
			shipping = &copyValue
		}
		entities = append(entities, toOrderEntity(row, itemMap[row.ID], statusMap[row.ID], shipping, shippingChargeMap[row.ID], orderMetadataMap[row.ID]))
	}

	return entities
}

// normalizeOrderIDs trims and deduplicates order identifiers.
func normalizeOrderIDs(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || slices.Contains(result, trimmed) {
			continue
		}
		result = append(result, trimmed)
	}

	return result
}
