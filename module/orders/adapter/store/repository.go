package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("orders db must not be nil")
	// ErrNilOrder is returned when a nil order entity is provided.
	ErrNilOrder = errors.New("order entity must not be nil")
)

// Repository implements order persistence using GORM.
type Repository struct {
	// db defines underlying GORM handles.
	db *gorm.DB
}

var (
	// _ ensures Repository satisfies order repository contracts.
	_ ordersport.Repository = (*Repository)(nil)
)

// NewRepository creates an order repository over GORM.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// EnsureSchema is a no-op because schema evolution is managed by SQL migrations.
func (r *Repository) EnsureSchema(ctx context.Context) error {
	_ = ctx

	return nil
}

// Create persists a new order aggregate.
func (r *Repository) Create(ctx context.Context, order *ordersdomain.Order) error {
	if order == nil {
		return ErrNilOrder
	}

	entity := *order
	entity.Normalize()
	if err := entity.Validate(); err != nil {
		return err
	}

	record := toOrderRecord(entity)
	if strings.TrimSpace(record.ID) == "" {
		record.ID = generateID()
	}
	itemRows := toOrderItemRecords(record.ID, entity.Items)
	statusRows := toOrderStatusRecords(record.ID, entity.StatusHistory)
	commentRows := toOrderCommentRecords(record.ID, entity.Comments)
	shippingChargeRows := toShippingChargeRecords(record.ID, entity.ShippingCharges)
	orderMetadataRows := toOrderMetadataRecords(record.ID, entity.Metadata)
	appliedCouponRows := toAppliedCouponRecords(record.ID, entity.AppliedCoupons)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return wrapWriteError("create order record", err)
		}
		if len(itemRows) > 0 {
			if err := tx.Create(&itemRows).Error; err != nil {
				return fmt.Errorf("create order item records: %w", err)
			}
		}
		if len(orderMetadataRows) > 0 {
			if err := tx.Create(&orderMetadataRows).Error; err != nil {
				return fmt.Errorf("create order metadata records: %w", err)
			}
		}
		if len(statusRows) > 0 {
			if err := tx.Create(&statusRows).Error; err != nil {
				return fmt.Errorf("create order status records: %w", err)
			}
		}
		if len(commentRows) > 0 {
			if err := tx.Create(&commentRows).Error; err != nil {
				return fmt.Errorf("create order comment records: %w", err)
			}
		}
		if len(shippingChargeRows) > 0 {
			if err := tx.Create(&shippingChargeRows).Error; err != nil {
				return fmt.Errorf("create order shipping charge records: %w", err)
			}
		}
		if entity.HasCustomShippingAddress {
			shipping := toShippingRecord(record.ID, entity.ShippingAddress)
			if err := tx.Create(&shipping).Error; err != nil {
				return fmt.Errorf("create order shipping record: %w", err)
			}
		}
		if len(appliedCouponRows) > 0 {
			if err := tx.Create(&appliedCouponRows).Error; err != nil {
				return fmt.Errorf("create order applied coupon records: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	latest, err := r.GetByID(ctx, record.ID)
	if err != nil {
		return err
	}
	*order = *latest

	return nil
}

// GetByID retrieves order aggregate values by identifier.
func (r *Repository) GetByID(ctx context.Context, id string) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)

	var row orderRecord
	if err := r.db.WithContext(ctx).First(&row, "id = ?", trimmedID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ordersport.ErrNotFound
		}
		return nil, fmt.Errorf("get order record: %w", err)
	}

	itemMap, statusMap, commentMap, shippingMap, shippingChargeMap, orderMetadataMap, appliedCouponMap, err := loadRelationsByOrderIDs(ctx, r.db, []string{trimmedID})
	if err != nil {
		return nil, err
	}

	var shipping *orderShippingAddressRecord
	if value, ok := shippingMap[trimmedID]; ok {
		copyValue := value
		shipping = &copyValue
	}
	entity := toOrderEntity(row, itemMap[trimmedID], statusMap[trimmedID], commentMap[trimmedID], shipping, shippingChargeMap[trimmedID], orderMetadataMap[trimmedID], appliedCouponMap[trimmedID])

	return &entity, nil
}

// List retrieves paginated order aggregate values.
func (r *Repository) List(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error) {
	page, limit := normalizePagination(query.Page, query.Limit)

	base := r.db.WithContext(ctx).Model(&orderRecord{})
	base = applyListQuery(base, query)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count order records: %w", err)
	}

	rows := make([]orderRecord, 0)
	if err := base.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list order records: %w", err)
	}

	orderIDs := collectOrderIDs(rows)
	itemMap, statusMap, commentMap, shippingMap, shippingChargeMap, orderMetadataMap, appliedCouponMap, err := loadRelationsByOrderIDs(ctx, r.db, orderIDs)
	if err != nil {
		return nil, 0, err
	}

	return mapRowsToEntities(rows, itemMap, statusMap, commentMap, shippingMap, shippingChargeMap, orderMetadataMap, appliedCouponMap), total, nil
}

// AppendStatus appends status rows for order identifiers.
func (r *Repository) AppendStatus(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row orderRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", trimmedID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ordersport.ErrNotFound
			}
			return fmt.Errorf("lock order record: %w", err)
		}

		position, err := nextStatusPosition(ctx, tx, trimmedID)
		if err != nil {
			return err
		}

		statusRow := orderStatusRecord{
			OrderID:     trimmedID,
			Position:    position,
			Status:      strings.TrimSpace(string(entry.Status)),
			Author:      strings.TrimSpace(entry.Author),
			Description: strings.TrimSpace(entry.Description),
			NoteOwner:   strings.TrimSpace(entry.NoteOwner),
			Note:        strings.TrimSpace(entry.Note),
			OccurredAt:  entry.OccurredAt.UTC(),
		}
		if err := tx.Create(&statusRow).Error; err != nil {
			return fmt.Errorf("append order status record: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, trimmedID)
}

// AppendComment appends comment rows for order identifiers.
func (r *Repository) AppendComment(ctx context.Context, id string, comment ordersdomain.Comment) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&orderRecord{}, "id = ?", trimmedID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ordersport.ErrNotFound
			}
			return fmt.Errorf("lock order record: %w", err)
		}

		row := orderCommentRecord{
			OrderID:    trimmedID,
			Author:     strings.TrimSpace(comment.Author),
			Comment:    strings.TrimSpace(comment.Comment),
			Internal:   comment.Internal,
			OccurredAt: comment.OccurredAt.UTC(),
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("append order comment record: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, trimmedID)
}

// nextStatusPosition resolves next status history position values for target orders.
func nextStatusPosition(ctx context.Context, tx *gorm.DB, orderID string) (int, error) {
	var currentMax int
	if err := tx.WithContext(ctx).Model(&orderStatusRecord{}).Where("order_id = ?", orderID).Select("COALESCE(MAX(position), -1)").Scan(&currentMax).Error; err != nil {
		return 0, fmt.Errorf("resolve order status position: %w", err)
	}

	return currentMax + 1, nil
}

// normalizePagination resolves list pagination defaults.
func normalizePagination(page int, limit int) (int, int) {
	resolvedPage := page
	if resolvedPage <= 0 {
		resolvedPage = 1
	}

	resolvedLimit := limit
	if resolvedLimit <= 0 {
		resolvedLimit = 10
	}

	return resolvedPage, resolvedLimit
}

// wrapWriteError normalizes persistence write errors to stable repository errors.
func wrapWriteError(operation string, err error) error {
	if mapped := mapDuplicateError(err); mapped != nil {
		return fmt.Errorf("%s: %w", operation, mapped)
	}

	return fmt.Errorf("%s: %w", operation, err)
}

// mapDuplicateError maps duplicate-key persistence failures into repository-level conflict errors.
func mapDuplicateError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if !errors.Is(err, gorm.ErrDuplicatedKey) && !isUniqueConstraintError(message) {
		return nil
	}
	if strings.Contains(message, "idx_orders_realm_identifier") ||
		strings.Contains(message, "orders.identifier") ||
		strings.Contains(message, "orders.realm") {
		return ordersport.ErrDuplicateIdentifier
	}

	return nil
}

// isUniqueConstraintError reports whether persistence errors represent uniqueness conflicts.
func isUniqueConstraintError(message string) bool {
	return strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "duplicated key") ||
		strings.Contains(message, "unique constraint failed") ||
		strings.Contains(message, "unique failed")
}
