package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

// Update persists mutable order aggregate values.
func (r *Repository) Update(ctx context.Context, order *ordersdomain.Order) error {
	if order == nil {
		return ErrNilOrder
	}

	entity := *order
	entity.Normalize()
	if err := entity.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(entity.ID) == "" {
		return ordersport.ErrNotFound
	}

	itemRows := toOrderItemRecords(entity.ID, entity.Items)
	shippingChargeRows := toShippingChargeRecords(entity.ID, entity.ShippingCharges)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing orderRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&existing, "id = ?", entity.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ordersport.ErrNotFound
			}
			return fmt.Errorf("lock order record: %w", err)
		}

		updateTx := tx.Model(&orderRecord{}).Where("id = ?", entity.ID).Updates(map[string]any{
			"current_status":             strings.TrimSpace(string(entity.CurrentStatus)),
			"current_status_author":      resolveCurrentStatusAuthor(entity.StatusHistory, existing),
			"current_status_description": resolveCurrentStatusDescription(entity.StatusHistory, existing),
			"current_status_at":          resolveCurrentStatusAt(entity.StatusHistory, existing),
			"updated_at":                 time.Now().UTC(),
		})
		if updateTx.Error != nil {
			return fmt.Errorf("update order root record: %w", updateTx.Error)
		}

		if err := tx.Where("order_id = ?", entity.ID).Delete(&orderItemRecord{}).Error; err != nil {
			return fmt.Errorf("delete order item records: %w", err)
		}
		if len(itemRows) > 0 {
			if err := tx.Create(&itemRows).Error; err != nil {
				return fmt.Errorf("create order item records: %w", err)
			}
		}

		if err := tx.Where("order_id = ?", entity.ID).Delete(&orderShippingChargeRecord{}).Error; err != nil {
			return fmt.Errorf("delete order shipping charge records: %w", err)
		}
		if len(shippingChargeRows) > 0 {
			if err := tx.Create(&shippingChargeRows).Error; err != nil {
				return fmt.Errorf("create order shipping charge records: %w", err)
			}
		}

		if entity.HasCustomShippingAddress {
			shipping := toShippingRecord(entity.ID, entity.ShippingAddress)
			if err := tx.Where("order_id = ?", entity.ID).Delete(&orderShippingAddressRecord{}).Error; err != nil {
				return fmt.Errorf("delete order shipping record: %w", err)
			}
			if err := tx.Create(&shipping).Error; err != nil {
				return fmt.Errorf("create order shipping record: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	latest, err := r.GetByID(ctx, entity.ID)
	if err != nil {
		return err
	}
	*order = *latest

	return nil
}

// resolveCurrentStatusAuthor resolves current-status author values.
func resolveCurrentStatusAuthor(history []ordersdomain.StatusEntry, existing orderRecord) string {
	if len(history) == 0 {
		return strings.TrimSpace(existing.CurrentStatusAuthor)
	}

	return strings.TrimSpace(history[len(history)-1].Author)
}

// resolveCurrentStatusDescription resolves current-status description values.
func resolveCurrentStatusDescription(history []ordersdomain.StatusEntry, existing orderRecord) string {
	if len(history) == 0 {
		return strings.TrimSpace(existing.CurrentStatusDescription)
	}

	return strings.TrimSpace(history[len(history)-1].Description)
}

// resolveCurrentStatusAt resolves current-status timestamp values.
func resolveCurrentStatusAt(history []ordersdomain.StatusEntry, existing orderRecord) time.Time {
	if len(history) == 0 {
		return existing.CurrentStatusAt.UTC()
	}

	return history[len(history)-1].OccurredAt.UTC()
}
