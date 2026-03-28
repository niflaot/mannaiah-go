package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// MarkRepository defines shipping-mark repository behavior.
type MarkRepository struct {
	// db defines gorm database dependencies.
	db *gorm.DB
}

// BatchRepository defines dispatch-batch repository behavior.
type BatchRepository struct {
	// db defines gorm database dependencies.
	db *gorm.DB
}

// QuotationRepository defines quotation repository behavior.
type QuotationRepository struct {
	// db defines gorm database dependencies.
	db *gorm.DB
}

var (
	// _ ensures MarkRepository satisfies shipping mark repository contracts.
	_ port.ShippingMarkRepository = (*MarkRepository)(nil)
	// _ ensures BatchRepository satisfies dispatch batch repository contracts.
	_ port.DispatchBatchRepository = (*BatchRepository)(nil)
	// _ ensures QuotationRepository satisfies quotation repository contracts.
	_ port.QuotationRepository = (*QuotationRepository)(nil)
)

// NewMarkRepository creates shipping-mark repositories.
func NewMarkRepository(db *gorm.DB) (*MarkRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("shipping database is required")
	}

	return &MarkRepository{db: db}, nil
}

// NewBatchRepository creates dispatch-batch repositories.
func NewBatchRepository(db *gorm.DB) (*BatchRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("shipping database is required")
	}

	return &BatchRepository{db: db}, nil
}

// NewQuotationRepository creates quotation repositories.
func NewQuotationRepository(db *gorm.DB) (*QuotationRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("shipping database is required")
	}

	return &QuotationRepository{db: db}, nil
}

// NewRepositories creates all shipping repository adapters.
func NewRepositories(db *gorm.DB) (*MarkRepository, *BatchRepository, *QuotationRepository, error) {
	marks, err := NewMarkRepository(db)
	if err != nil {
		return nil, nil, nil, err
	}
	batches, err := NewBatchRepository(db)
	if err != nil {
		return nil, nil, nil, err
	}
	quotations, err := NewQuotationRepository(db)
	if err != nil {
		return nil, nil, nil, err
	}

	return marks, batches, quotations, nil
}

// Create creates one shipping mark.
func (r *MarkRepository) Create(ctx context.Context, mark *domain.ShippingMark) error {
	if mark == nil {
		return domain.ErrInvalidID
	}
	row := mapMarkDomain(mark.Normalize())
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	row.UpdatedAt = time.Now().UTC()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Create(&row).Error; err != nil {
			return fmt.Errorf("create shipping mark: %w", err)
		}
		if len(row.Units) == 0 {
			return nil
		}
		if err := tx.Create(&row.Units).Error; err != nil {
			return fmt.Errorf("create shipping mark units: %w", err)
		}

		return nil
	})
}

// GetByID loads one shipping mark by identifier.
func (r *MarkRepository) GetByID(ctx context.Context, id string) (*domain.ShippingMark, error) {
	row := shippingMarkModel{}
	err := r.db.WithContext(ctx).Preload("Units").Where("id = ?", strings.TrimSpace(id)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load shipping mark by id: %w", err)
	}
	mapped := mapMarkModel(row)

	return &mapped, nil
}

// GetByTrackingNumber loads one shipping mark by tracking number.
func (r *MarkRepository) GetByTrackingNumber(ctx context.Context, trackingNumber string) (*domain.ShippingMark, error) {
	row := shippingMarkModel{}
	err := r.db.WithContext(ctx).Preload("Units").Where("tracking_number = ?", strings.TrimSpace(trackingNumber)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load shipping mark by tracking number: %w", err)
	}
	mapped := mapMarkModel(row)

	return &mapped, nil
}

// ListByOrderID loads shipping marks by order identifier.
func (r *MarkRepository) ListByOrderID(ctx context.Context, orderID string) ([]domain.ShippingMark, error) {
	rows := make([]shippingMarkModel, 0)
	if err := r.db.WithContext(ctx).Preload("Units").Where("order_id = ?", strings.TrimSpace(orderID)).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list shipping marks by order id: %w", err)
	}

	return mapMarkModels(rows), nil
}

// ListByBatchID loads shipping marks by batch identifier.
func (r *MarkRepository) ListByBatchID(ctx context.Context, batchID string) ([]domain.ShippingMark, error) {
	rows := make([]shippingMarkModel, 0)
	if err := r.db.WithContext(ctx).Preload("Units").Where("dispatch_batch_id = ?", strings.TrimSpace(batchID)).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list shipping marks by batch id: %w", err)
	}

	return mapMarkModels(rows), nil
}

// Update updates one shipping mark.
func (r *MarkRepository) Update(ctx context.Context, mark *domain.ShippingMark) error {
	if mark == nil {
		return domain.ErrInvalidID
	}
	row := mapMarkDomain(mark.Normalize())
	row.UpdatedAt = time.Now().UTC()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&shippingMarkModel{}).Where("id = ?", row.ID).Updates(map[string]any{
			"carrier_id":                         row.CarrierID,
			"tracking_number":                    row.TrackingNumber,
			"status":                             row.Status,
			"document_type":                      row.DocumentType,
			"document_ref":                       row.DocumentRef,
			"manifest_type":                      row.ManifestType,
			"manifest_ref":                       row.ManifestRef,
			"sender_name":                        row.SenderName,
			"sender_legal_name":                  row.SenderLegalName,
			"sender_id":                          row.SenderID,
			"sender_id_type":                     row.SenderIDType,
			"sender_address":                     row.SenderAddress,
			"sender_city_code":                   row.SenderCityCode,
			"sender_phone":                       row.SenderPhone,
			"sender_email":                       row.SenderEmail,
			"recipient_name":                     row.RecipientName,
			"recipient_legal_name":               row.RecipientLegalName,
			"recipient_id":                       row.RecipientID,
			"recipient_id_type":                  row.RecipientIDType,
			"recipient_address":                  row.RecipientAddress,
			"recipient_city_code":                row.RecipientCityCode,
			"recipient_phone":                    row.RecipientPhone,
			"recipient_email":                    row.RecipientEmail,
			"total_weight":                       row.TotalWeight,
			"total_volumetric_weight":            row.TotalVolumetricWeight,
			"declared_value":                     row.DeclaredValue,
			"payment_form":                       row.PaymentForm,
			"collect_on_delivery_amount":         row.CollectOnDeliveryAmount,
			"collect_on_delivery_fee_percent":    row.CollectOnDeliveryFeePercent,
			"collect_on_delivery_charged_amount": row.CollectOnDeliveryChargedAmount,
			"observations":                       row.Observations,
			"dispatch_batch_id":                  row.DispatchBatchID,
			"quotation_id":                       row.QuotationID,
			"quoted_freight_cost":                row.QuotedFreightCost,
			"draft_snapshot":                     row.DraftSnapshot,
			"shipment_mode":                      row.ShipmentMode,
			"failure_reason":                     row.FailureReason,
			"updated_at":                         row.UpdatedAt,
		})
		if result.Error != nil {
			return fmt.Errorf("update shipping mark: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		if err := tx.Where("shipping_mark_id = ?", row.ID).Delete(&shippingMarkUnitModel{}).Error; err != nil {
			return fmt.Errorf("delete shipping mark units: %w", err)
		}
		if len(row.Units) > 0 {
			if err := tx.Create(&row.Units).Error; err != nil {
				return fmt.Errorf("insert shipping mark units: %w", err)
			}
		}

		return nil
	})
}

// Delete permanently deletes one shipping mark and its units by identifier.
func (r *MarkRepository) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return domain.ErrInvalidID
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("shipping_mark_id = ?", trimmedID).Delete(&shippingMarkUnitModel{}).Error; err != nil {
			return fmt.Errorf("delete shipping mark units: %w", err)
		}
		result := tx.Where("id = ?", trimmedID).Delete(&shippingMarkModel{})
		if result.Error != nil {
			return fmt.Errorf("delete shipping mark: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}

		return nil
	})
}

// List lists marks using pagination and filters.
func (r *MarkRepository) List(ctx context.Context, query port.MarkListQuery) ([]domain.ShippingMark, int64, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	builder := r.db.WithContext(ctx).Model(&shippingMarkModel{})
	if strings.TrimSpace(query.OrderID) != "" {
		builder = builder.Where("order_id = ?", strings.TrimSpace(query.OrderID))
	}
	if strings.TrimSpace(query.BatchID) != "" {
		builder = builder.Where("dispatch_batch_id = ?", strings.TrimSpace(query.BatchID))
	}
	var total int64
	if err := builder.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count shipping marks: %w", err)
	}
	rows := make([]shippingMarkModel, 0, limit)
	if err := builder.Preload("Units").Order("created_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list shipping marks: %w", err)
	}

	return mapMarkModels(rows), total, nil
}

// Create creates one dispatch batch.
func (r *BatchRepository) Create(ctx context.Context, batch *domain.DispatchBatch) error {
	if batch == nil {
		return domain.ErrInvalidID
	}
	normalized := batch.Normalize()
	row := dispatchBatchModel{
		ID:        normalized.ID,
		CarrierID: normalized.CarrierID,
		Status:    string(normalized.Status),
		CreatedBy: normalized.CreatedBy,
		CreatedAt: normalized.CreatedAt,
		ClosedAt:  normalized.ClosedAt,
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	if row.Status == "" {
		row.Status = string(domain.BatchStatusOpen)
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("create dispatch batch: %w", err)
	}

	return nil
}

// GetByID loads one dispatch batch by identifier.
func (r *BatchRepository) GetByID(ctx context.Context, id string) (*domain.DispatchBatch, error) {
	row := dispatchBatchModel{}
	err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load dispatch batch by id: %w", err)
	}
	markIDs, err := r.listMarkIDs(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	mapped := mapBatchModel(row, markIDs)

	return &mapped, nil
}

// Close closes one dispatch batch.
func (r *BatchRepository) Close(ctx context.Context, id string) error {
	closedAt := time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&dispatchBatchModel{}).
		Where("id = ? AND status = ?", strings.TrimSpace(id), string(domain.BatchStatusOpen)).
		Updates(map[string]any{"status": string(domain.BatchStatusClosed), "closed_at": &closedAt})
	if result.Error != nil {
		return fmt.Errorf("close dispatch batch: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		row := dispatchBatchModel{}
		err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load dispatch batch before close: %w", err)
		}

		return domain.ErrBatchClosed
	}

	return nil
}

// AddMark assigns one mark to the batch.
func (r *BatchRepository) AddMark(ctx context.Context, batchID string, markID string) error {
	trimmedBatchID := strings.TrimSpace(batchID)
	trimmedMarkID := strings.TrimSpace(markID)
	if trimmedBatchID == "" || trimmedMarkID == "" {
		return domain.ErrInvalidID
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		batch := dispatchBatchModel{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", trimmedBatchID).First(&batch).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}

			return fmt.Errorf("load dispatch batch: %w", err)
		}
		if batch.Status != string(domain.BatchStatusOpen) {
			return domain.ErrBatchClosed
		}
		result := tx.Model(&shippingMarkModel{}).
			Where("id = ?", trimmedMarkID).
			Update("dispatch_batch_id", &trimmedBatchID)
		if result.Error != nil {
			return fmt.Errorf("assign mark to batch: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}

		return nil
	})
}

// RemoveMark removes one mark from the batch.
func (r *BatchRepository) RemoveMark(ctx context.Context, batchID string, markID string) error {
	trimmedBatchID := strings.TrimSpace(batchID)
	trimmedMarkID := strings.TrimSpace(markID)
	if trimmedBatchID == "" || trimmedMarkID == "" {
		return domain.ErrInvalidID
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		batch := dispatchBatchModel{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", trimmedBatchID).First(&batch).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}

			return fmt.Errorf("load dispatch batch: %w", err)
		}
		if batch.Status != string(domain.BatchStatusOpen) {
			return domain.ErrBatchClosed
		}

		result := tx.Model(&shippingMarkModel{}).
			Where("id = ? AND dispatch_batch_id = ?", trimmedMarkID, trimmedBatchID).
			Update("dispatch_batch_id", nil)
		if result.Error != nil {
			return fmt.Errorf("remove mark from batch: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}

		return nil
	})
}

// List lists dispatch batches using pagination and filters.
func (r *BatchRepository) List(ctx context.Context, query port.BatchListQuery) ([]domain.DispatchBatch, int64, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	builder := r.db.WithContext(ctx).Model(&dispatchBatchModel{})
	if strings.TrimSpace(query.CarrierID) != "" {
		builder = builder.Where("carrier_id = ?", strings.TrimSpace(query.CarrierID))
	}
	if strings.TrimSpace(string(query.Status)) != "" {
		builder = builder.Where("status = ?", strings.TrimSpace(string(query.Status)))
	}
	var total int64
	if err := builder.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count dispatch batches: %w", err)
	}
	rows := make([]dispatchBatchModel, 0, limit)
	if err := builder.Order("created_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list dispatch batches: %w", err)
	}
	result := make([]domain.DispatchBatch, 0, len(rows))
	for _, row := range rows {
		markIDs, err := r.listMarkIDs(ctx, row.ID)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, mapBatchModel(row, markIDs))
	}

	return result, total, nil
}

// Create creates one quotation audit record.
func (r *QuotationRepository) Create(ctx context.Context, record port.QuotationRecord) error {
	row := mapQuotationModel(record)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("create quotation: %w", err)
	}

	return nil
}

// DeleteExpired deletes all quotation records whose expiration timestamp is in the past.
func (r *QuotationRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Where("expires_at > '0001-01-01 00:00:00' AND expires_at <= ?", time.Now().UTC()).Delete(&quotationModel{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete expired quotations: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// ListByOrderID lists quotation records by order identifier.
func (r *QuotationRepository) ListByOrderID(ctx context.Context, orderID string) ([]port.QuotationRecord, error) {
	rows := make([]quotationModel, 0)
	if err := r.db.WithContext(ctx).Where("order_id = ?", strings.TrimSpace(orderID)).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list quotations by order id: %w", err)
	}
	result := make([]port.QuotationRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapQuotationRecord(row))
	}

	return result, nil
}

// GetLatestByOrderAndCarrier returns the most recent non-expired quotation for the given order and carrier.
func (r *QuotationRepository) GetLatestByOrderAndCarrier(ctx context.Context, orderID string, carrierID string) (*port.QuotationRecord, error) {
	var row quotationModel
	err := r.db.WithContext(ctx).
		Where("order_id = ? AND carrier_id = ? AND expires_at > ?", strings.TrimSpace(orderID), strings.TrimSpace(carrierID), time.Now().UTC()).
		Order("created_at DESC").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("get latest quotation by order and carrier: %w", err)
	}
	record := mapQuotationRecord(row)

	return &record, nil
}

// listMarkIDs lists mark identifiers belonging to one batch.
func (r *BatchRepository) listMarkIDs(ctx context.Context, batchID string) ([]string, error) {
	rows := make([]shippingMarkModel, 0)
	if err := r.db.WithContext(ctx).Select("id").Where("dispatch_batch_id = ?", strings.TrimSpace(batchID)).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list mark ids by batch id: %w", err)
	}
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, strings.TrimSpace(row.ID))
	}

	return result, nil
}

// mapMarkModels maps shipping mark rows to domain values.
func mapMarkModels(rows []shippingMarkModel) []domain.ShippingMark {
	result := make([]domain.ShippingMark, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapMarkModel(row))
	}

	return result
}
