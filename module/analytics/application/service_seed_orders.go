package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/analytics/port"
)

// orderSeedRow holds an order row read from the transactional database during seed.
type orderSeedRow struct {
	// ID is the order UUID.
	ID string `gorm:"column:id"`
	// Identifier is the external order identifier.
	Identifier string `gorm:"column:identifier"`
	// Realm is the order source realm.
	Realm string `gorm:"column:realm"`
	// ContactID is the linked contact UUID.
	ContactID string `gorm:"column:contact_id"`
	// CreatedAt is the order creation timestamp.
	CreatedAt *time.Time `gorm:"column:created_at"`
	// UpdatedAt is the order last-updated timestamp.
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

// orderItemSeedRow holds an order item row read from the transactional database during seed.
type orderItemSeedRow struct {
	// OrderID is the parent order UUID.
	OrderID string `gorm:"column:order_id"`
	// SKU is the product SKU.
	SKU string `gorm:"column:sku"`
	// AlternateName is the resolved product alternate name.
	AlternateName string `gorm:"column:alternate_name"`
	// Quantity is the item quantity.
	Quantity int `gorm:"column:quantity"`
	// Value is the item monetary value.
	Value float64 `gorm:"column:value"`
	// ProductID is the linked product UUID.
	ProductID string `gorm:"column:product_id"`
	// ResolutionSource is the product name resolution source.
	ResolutionSource string `gorm:"column:resolution_source"`
}

// orderStatusSeedRow holds the latest status row for an order during seed.
type orderStatusSeedRow struct {
	// OrderID is the order UUID.
	OrderID string `gorm:"column:order_id"`
	// Status is the latest order status.
	Status string `gorm:"column:status"`
}

// seedOrders reads orders in batches from the transactional database and upserts them into the analytics store.
func (s *AnalyticsService) seedOrders(ctx context.Context, summary *SeedSummary) error {
	lastID := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		orders := make([]orderSeedRow, 0, seedBatchSize)
		query := s.db.WithContext(ctx).
			Table("orders").
			Select("id", "identifier", "realm", "contact_id", "created_at", "updated_at").
			Where("deleted_at IS NULL")
		if lastID != "" {
			query = query.Where("id > ?", lastID)
		}
		if err := query.Order("id ASC").Limit(seedBatchSize).Scan(&orders).Error; err != nil {
			return fmt.Errorf("seed orders batch: %w", err)
		}
		if len(orders) == 0 {
			break
		}

		orderIDs, orderByID := indexOrders(orders)
		itemRows, statusByOrder, err := s.loadOrderRelations(ctx, orderIDs)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		ordersPayload, orderItemsPayload := buildOrderPayloads(orders, orderIDs, orderByID, itemRows, statusByOrder, now)

		if err := s.store.UpsertOrders(ctx, ordersPayload); err != nil {
			return fmt.Errorf("upsert orders fact batch: %w", err)
		}
		if err := s.store.UpsertOrderItems(ctx, orderItemsPayload); err != nil {
			return fmt.Errorf("upsert order items fact batch: %w", err)
		}
		summary.Orders += int64(len(ordersPayload))
		summary.OrderItems += int64(len(orderItemsPayload))
		lastID = orders[len(orders)-1].ID
	}

	return nil
}

// indexOrders builds an ID slice and ID→row index from an order batch.
func indexOrders(orders []orderSeedRow) ([]string, map[string]orderSeedRow) {
	orderIDs := make([]string, 0, len(orders))
	orderByID := map[string]orderSeedRow{}
	for _, row := range orders {
		orderID := strings.TrimSpace(row.ID)
		orderIDs = append(orderIDs, orderID)
		orderByID[orderID] = row
	}

	return orderIDs, orderByID
}

// loadOrderRelations fetches order items and latest statuses for the given order IDs.
func (s *AnalyticsService) loadOrderRelations(ctx context.Context, orderIDs []string) ([]orderItemSeedRow, map[string]string, error) {
	itemRows := make([]orderItemSeedRow, 0, len(orderIDs)*2)
	if err := s.db.WithContext(ctx).
		Table("order_items").
		Select("order_id", "sku", "alternate_name", "quantity", "value", "product_id", "resolution_source").
		Where("order_id IN ?", orderIDs).
		Order("order_id ASC").
		Scan(&itemRows).Error; err != nil {
		return nil, nil, fmt.Errorf("seed order items batch: %w", err)
	}

	statusRows := make([]orderStatusSeedRow, 0, len(orderIDs))
	statusQuery := `SELECT osh.order_id, osh.status
		FROM order_status_history osh
		INNER JOIN (
			SELECT order_id, max(occurred_at) AS latest_occurred_at
			FROM order_status_history
			WHERE order_id IN ?
			GROUP BY order_id
		) latest ON latest.order_id = osh.order_id AND latest.latest_occurred_at = osh.occurred_at`
	if err := s.db.WithContext(ctx).Raw(statusQuery, orderIDs).Scan(&statusRows).Error; err != nil {
		return nil, nil, fmt.Errorf("seed order statuses batch: %w", err)
	}
	statusByOrder := map[string]string{}
	for _, row := range statusRows {
		statusByOrder[strings.TrimSpace(row.OrderID)] = strings.TrimSpace(row.Status)
	}

	return itemRows, statusByOrder, nil
}

// buildOrderPayloads maps order and item seed rows into analytics fact payloads.
func buildOrderPayloads(
	orders []orderSeedRow,
	orderIDs []string,
	orderByID map[string]orderSeedRow,
	itemRows []orderItemSeedRow,
	statusByOrder map[string]string,
	now time.Time,
) ([]port.OrderFact, []port.OrderItemFact) {
	totalByOrder := map[string]float64{}
	countByOrder := map[string]int{}
	orderItemsPayload := make([]port.OrderItemFact, 0, len(itemRows))

	for _, row := range itemRows {
		orderID := strings.TrimSpace(row.OrderID)
		orderSeed, exists := orderByID[orderID]
		if !exists {
			continue
		}
		createdAt, updatedAt := resolveOrderTimestamps(orderSeed.CreatedAt, orderSeed.UpdatedAt, now)
		totalByOrder[orderID] += row.Value
		countByOrder[orderID] += row.Quantity
		orderItemsPayload = append(orderItemsPayload, port.OrderItemFact{
			OrderID:          orderID,
			ContactID:        strings.TrimSpace(orderSeed.ContactID),
			SKU:              strings.TrimSpace(row.SKU),
			AlternateName:    strings.TrimSpace(row.AlternateName),
			ProductID:        strings.TrimSpace(row.ProductID),
			Quantity:         row.Quantity,
			Value:            row.Value,
			ResolutionSource: strings.TrimSpace(row.ResolutionSource),
			OrderCreatedAt:   createdAt,
			OrderUpdatedAt:   updatedAt,
		})
	}

	_ = orderIDs
	ordersPayload := make([]port.OrderFact, 0, len(orders))
	for _, row := range orders {
		createdAt, updatedAt := resolveOrderTimestamps(row.CreatedAt, row.UpdatedAt, now)
		orderID := strings.TrimSpace(row.ID)
		ordersPayload = append(ordersPayload, port.OrderFact{
			OrderID:       orderID,
			Identifier:    strings.TrimSpace(row.Identifier),
			Realm:         strings.TrimSpace(row.Realm),
			ContactID:     strings.TrimSpace(row.ContactID),
			CurrentStatus: strings.TrimSpace(statusByOrder[orderID]),
			TotalValue:    totalByOrder[orderID],
			ItemCount:     countByOrder[orderID],
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		})
	}

	return ordersPayload, orderItemsPayload
}

// resolveOrderTimestamps resolves non-nil timestamps from order seed rows, falling back to now.
func resolveOrderTimestamps(createdAt *time.Time, updatedAt *time.Time, now time.Time) (time.Time, time.Time) {
	ca := now
	ua := now
	if createdAt != nil {
		ca = createdAt.UTC()
	}
	if updatedAt != nil {
		ua = updatedAt.UTC()
	}

	return ca, ua
}
