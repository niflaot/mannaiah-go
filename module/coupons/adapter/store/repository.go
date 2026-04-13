// Package store defines coupon persistence adapter implementations.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"mannaiah/module/coupons/domain"
	"mannaiah/module/coupons/port"
)

// Repository defines coupon and usage GORM persistence adapters.
type Repository struct {
	// db defines the GORM database connection.
	db *gorm.DB
}

// NewRepository creates coupon persistence adapter instances.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("coupon repository: db must not be nil")
	}
	return &Repository{db: db}, nil
}

// Create persists a new coupon aggregate and all its child rows.
func (r *Repository) Create(ctx context.Context, coupon *domain.Coupon) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := toCouponRecord(*coupon)
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("insert coupon: %w", err)
		}
		return r.upsertChildren(ctx, tx, coupon.ID, *coupon)
	})
}

// GetByID retrieves a coupon aggregate by its unique identifier.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Coupon, error) {
	return r.loadOne(ctx, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("id = ? AND deleted_at IS NULL", strings.TrimSpace(id))
	})
}

// GetByCode retrieves a coupon aggregate by its unique code.
func (r *Repository) GetByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	return r.loadOne(ctx, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("code = ? AND deleted_at IS NULL", strings.ToUpper(strings.TrimSpace(code)))
	})
}

// GetByWooCommerceID retrieves a coupon aggregate by its WooCommerce identifier.
func (r *Repository) GetByWooCommerceID(ctx context.Context, wooID int) (*domain.Coupon, error) {
	return r.loadOne(ctx, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("woocommerce_id = ? AND deleted_at IS NULL", wooID)
	})
}

// Update replaces all coupon fields and child rows within a transaction.
func (r *Repository) Update(ctx context.Context, coupon *domain.Coupon) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := toCouponRecord(*coupon)
		if err := tx.Model(&couponRecord{}).Where("id = ?", coupon.ID).Save(&record).Error; err != nil {
			return fmt.Errorf("update coupon root: %w", err)
		}

		if err := r.deleteChildren(ctx, tx, coupon.ID); err != nil {
			return err
		}
		return r.upsertChildren(ctx, tx, coupon.ID, *coupon)
	})
}

// Delete soft-deletes a coupon by setting deleted_at.
func (r *Repository) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC()
	if err := r.db.WithContext(ctx).
		Model(&couponRecord{}).
		Where("id = ? AND deleted_at IS NULL", strings.TrimSpace(id)).
		Update("deleted_at", now).Error; err != nil {
		return fmt.Errorf("soft-delete coupon: %w", err)
	}
	return nil
}

// List retrieves paginated coupons matching the provided query.
func (r *Repository) List(ctx context.Context, query port.ListQuery) ([]domain.Coupon, int64, error) {
	tx := r.db.WithContext(ctx).Model(&couponRecord{}).Where("deleted_at IS NULL")

	if v := strings.TrimSpace(query.Origin); v != "" {
		tx = tx.Where("origin = ?", v)
	}
	if v := strings.TrimSpace(query.Code); v != "" {
		tx = tx.Where("code = ?", strings.ToUpper(v))
	}
	if query.Active != nil {
		if *query.Active {
			tx = tx.Where("active = ?", true)
		} else {
			tx = tx.Where("active = ?", false)
		}
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count coupons: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	var records []couponRecord
	if err := tx.Limit(limit).Offset(offset).Order("created_at DESC, id DESC").Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list coupons: %w", err)
	}

	ids := make([]string, 0, len(records))
	for _, rec := range records {
		ids = append(ids, rec.ID)
	}

	emails, contacts, products, categories, tags, err := r.loadChildrenByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	result := make([]domain.Coupon, 0, len(records))
	for _, rec := range records {
		result = append(result, toCouponEntity(rec, emails[rec.ID], contacts[rec.ID], products[rec.ID], categories[rec.ID], tags[rec.ID]))
	}

	return result, total, nil
}

// CodeExists reports whether a coupon code is already in use.
func (r *Repository) CodeExists(ctx context.Context, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&couponRecord{}).
		Where("code = ? AND deleted_at IS NULL", strings.ToUpper(strings.TrimSpace(code))).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check coupon code existence: %w", err)
	}
	return count > 0, nil
}

// RecordUsage persists a coupon redemption event.
func (r *Repository) RecordUsage(ctx context.Context, record port.UsageRecord) error {
	row := couponUsageRecord{
		CouponID: strings.TrimSpace(record.CouponID),
		OrderID:  strings.TrimSpace(record.OrderID),
		Email:    strings.ToLower(strings.TrimSpace(record.Email)),
		UsedAt:   record.UsedAt.UTC(),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("record coupon usage: %w", err)
	}
	return nil
}

// CountGlobalUsage counts all redemptions for a coupon.
func (r *Repository) CountGlobalUsage(ctx context.Context, couponID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&couponUsageRecord{}).
		Where("coupon_id = ?", strings.TrimSpace(couponID)).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count global coupon usage: %w", err)
	}
	return count, nil
}

// CountUsageByEmail counts redemptions for a coupon by email.
func (r *Repository) CountUsageByEmail(ctx context.Context, couponID string, email string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&couponUsageRecord{}).
		Where("coupon_id = ? AND email = ?", strings.TrimSpace(couponID), strings.ToLower(strings.TrimSpace(email))).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count email coupon usage: %w", err)
	}
	return count, nil
}

// UsageExistsForOrder reports whether a coupon was already applied to a specific order.
func (r *Repository) UsageExistsForOrder(ctx context.Context, couponID string, orderID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&couponUsageRecord{}).
		Where("coupon_id = ? AND order_id = ?", strings.TrimSpace(couponID), strings.TrimSpace(orderID)).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check coupon order usage: %w", err)
	}
	return count > 0, nil
}

// loadOne retrieves a single coupon aggregate with all child rows.
func (r *Repository) loadOne(ctx context.Context, where func(*gorm.DB) *gorm.DB) (*domain.Coupon, error) {
	var record couponRecord
	tx := where(r.db.WithContext(ctx).Model(&couponRecord{}))
	if err := tx.First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load coupon: %w", err)
	}

	emails, contacts, products, categories, tags, err := r.loadChildrenByIDs(ctx, []string{record.ID})
	if err != nil {
		return nil, err
	}

	c := toCouponEntity(record, emails[record.ID], contacts[record.ID], products[record.ID], categories[record.ID], tags[record.ID])
	return &c, nil
}

// loadChildrenByIDs loads all child rows for the given coupon IDs grouped by coupon ID.
func (r *Repository) loadChildrenByIDs(ctx context.Context, ids []string) (
	map[string][]couponAssignedEmailRecord,
	map[string][]couponAssignedContactRecord,
	map[string][]couponIncludedProductRecord,
	map[string][]couponIncludedCategoryRecord,
	map[string][]couponIncludedTagRecord,
	error,
) {
	emailMap := make(map[string][]couponAssignedEmailRecord)
	contactMap := make(map[string][]couponAssignedContactRecord)
	productMap := make(map[string][]couponIncludedProductRecord)
	categoryMap := make(map[string][]couponIncludedCategoryRecord)
	tagMap := make(map[string][]couponIncludedTagRecord)

	if len(ids) == 0 {
		return emailMap, contactMap, productMap, categoryMap, tagMap, nil
	}

	var emails []couponAssignedEmailRecord
	if err := r.db.WithContext(ctx).Where("coupon_id IN ?", ids).Find(&emails).Error; err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("load coupon emails: %w", err)
	}
	for _, row := range emails {
		emailMap[row.CouponID] = append(emailMap[row.CouponID], row)
	}

	var contacts []couponAssignedContactRecord
	if err := r.db.WithContext(ctx).Where("coupon_id IN ?", ids).Find(&contacts).Error; err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("load coupon contacts: %w", err)
	}
	for _, row := range contacts {
		contactMap[row.CouponID] = append(contactMap[row.CouponID], row)
	}

	var products []couponIncludedProductRecord
	if err := r.db.WithContext(ctx).Where("coupon_id IN ?", ids).Find(&products).Error; err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("load coupon products: %w", err)
	}
	for _, row := range products {
		productMap[row.CouponID] = append(productMap[row.CouponID], row)
	}

	var categories []couponIncludedCategoryRecord
	if err := r.db.WithContext(ctx).Where("coupon_id IN ?", ids).Find(&categories).Error; err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("load coupon categories: %w", err)
	}
	for _, row := range categories {
		categoryMap[row.CouponID] = append(categoryMap[row.CouponID], row)
	}

	var tags []couponIncludedTagRecord
	if err := r.db.WithContext(ctx).Where("coupon_id IN ?", ids).Find(&tags).Error; err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("load coupon tags: %w", err)
	}
	for _, row := range tags {
		tagMap[row.CouponID] = append(tagMap[row.CouponID], row)
	}

	return emailMap, contactMap, productMap, categoryMap, tagMap, nil
}

// upsertChildren inserts all child rows for a coupon within an active transaction.
func (r *Repository) upsertChildren(ctx context.Context, tx *gorm.DB, couponID string, coupon domain.Coupon) error {
	if rows := toAssignedEmailRecords(couponID, coupon.AssignedEmails); len(rows) > 0 {
		if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
			return fmt.Errorf("insert coupon emails: %w", err)
		}
	}
	if rows := toAssignedContactRecords(couponID, coupon.AssignedContactIDs); len(rows) > 0 {
		if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
			return fmt.Errorf("insert coupon contacts: %w", err)
		}
	}
	if rows := toIncludedProductRecords(couponID, coupon.IncludedProductIDs); len(rows) > 0 {
		if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
			return fmt.Errorf("insert coupon products: %w", err)
		}
	}
	if rows := toIncludedCategoryRecords(couponID, coupon.IncludedCategoryIDs); len(rows) > 0 {
		if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
			return fmt.Errorf("insert coupon categories: %w", err)
		}
	}
	if rows := toIncludedTagRecords(couponID, coupon.IncludedTagIDs); len(rows) > 0 {
		if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
			return fmt.Errorf("insert coupon tags: %w", err)
		}
	}
	return nil
}

// deleteChildren removes all child rows for a coupon within an active transaction.
func (r *Repository) deleteChildren(ctx context.Context, tx *gorm.DB, couponID string) error {
	tables := []any{
		&couponAssignedEmailRecord{},
		&couponAssignedContactRecord{},
		&couponIncludedProductRecord{},
		&couponIncludedCategoryRecord{},
		&couponIncludedTagRecord{},
	}
	for _, model := range tables {
		if err := tx.WithContext(ctx).Where("coupon_id = ?", couponID).Delete(model).Error; err != nil {
			return fmt.Errorf("delete coupon children: %w", err)
		}
	}
	return nil
}
