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
	record := toOrderRecord(entity)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&orderRecord{}, "id = ?", entity.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ordersport.ErrNotFound
			}
			return fmt.Errorf("lock order record: %w", err)
		}

		updateTx := tx.Model(&orderRecord{}).Where("id = ?", entity.ID).Updates(map[string]any{
			"updated_at":             time.Now().UTC(),
			"payment_method":         record.PaymentMethod,
			"coupon_code":            record.CouponCode,
			"coupon_discount_amount": record.CouponDiscountAmount,
			"coupon_discount_type":   record.CouponDiscountType,
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
		} else {
			if err := tx.Where("order_id = ?", entity.ID).Delete(&orderShippingAddressRecord{}).Error; err != nil {
				return fmt.Errorf("delete order shipping record: %w", err)
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
