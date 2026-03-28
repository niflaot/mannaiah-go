package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	dispatchevent "mannaiah/module/shipping/application/dispatch/event"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// MarkMaterializer defines carrier-submission behavior invoked at batch close.
type MarkMaterializer interface {
	// Materialize submits one QUOTED draft mark to the carrier and updates its status.
	Materialize(ctx context.Context, mark *domain.ShippingMark) error
}

// CreateBatchCommand defines dispatch batch creation input values.
type CreateBatchCommand struct {
	// CarrierID defines batch carrier identifier values.
	CarrierID string
	// CreatedBy defines the subject identifier of the caller creating the batch.
	CreatedBy string
}

// DraftMarkCommand defines draft mark creation input values.
type DraftMarkCommand struct {
	// BatchID defines the target batch identifier.
	BatchID string
	// QuotationID defines the optional quotation reference attached to this draft.
	QuotationID string
	// QuotedFreightCost defines the freight cost snapshot from the quotation.
	QuotedFreightCost float64
	// OrderID defines order identifier values.
	OrderID string
	// Sender defines sender address values.
	Sender domain.Address
	// Recipient defines recipient address values.
	Recipient domain.Address
	// Units defines shipment package units.
	Units []domain.PackageUnit
	// DeclaredValue defines declared shipment value amounts.
	DeclaredValue float64
	// PaymentForm defines payment arrangement values.
	PaymentForm string
	// CollectOnDeliveryAmount defines requested cash-on-delivery collection amounts.
	CollectOnDeliveryAmount float64
	// ShipmentMode defines the delivery mode for this draft mark.
	ShipmentMode domain.ShipmentMode
	// Observations defines optional observation values.
	Observations string
}

// CreateBatchMarkCommand defines one batch mark creation input values for quoted or direct flows.
type CreateBatchMarkCommand struct {
	// BatchID defines the target batch identifier.
	BatchID string
	// Direct defines whether the mark should be materialized immediately.
	Direct bool
	// QuotationID defines the optional quotation reference attached to this draft.
	QuotationID string
	// QuotedFreightCost defines the freight cost snapshot from the quotation.
	QuotedFreightCost float64
	// OrderID defines order identifier values.
	OrderID string
	// Sender defines sender address values.
	Sender domain.Address
	// Recipient defines recipient address values.
	Recipient domain.Address
	// Units defines shipment package units.
	Units []domain.PackageUnit
	// DeclaredValue defines declared shipment value amounts.
	DeclaredValue float64
	// PaymentForm defines payment arrangement values.
	PaymentForm string
	// CollectOnDeliveryAmount defines requested cash-on-delivery collection amounts.
	CollectOnDeliveryAmount float64
	// ShipmentMode defines the delivery mode for this draft mark.
	ShipmentMode domain.ShipmentMode
	// Observations defines optional observation values.
	Observations string
}

// CreateBatchMarkFromQuotationCommand defines batch-mark creation values resolved from one quotation.
type CreateBatchMarkFromQuotationCommand struct {
	// BatchID defines the target batch identifier.
	BatchID string
	// QuotationID defines the quotation identifier used to seed mark values.
	QuotationID string
	// Direct defines whether the mark should be materialized immediately.
	Direct bool
}

// ListQuery defines dispatch batch listing query values.
type ListQuery struct {
	// CarrierID filters rows by carrier identifier.
	CarrierID string
	// Status filters rows by batch status.
	Status domain.BatchStatus
	// Page defines 1-based page values.
	Page int
	// Limit defines page-size values.
	Limit int
}

// Service defines dispatch batch orchestration behavior.
type Service struct {
	// batchRepository defines dispatch batch persistence dependencies.
	batchRepository port.DispatchBatchRepository
	// markRepository defines mark persistence dependencies.
	markRepository port.ShippingMarkRepository
	// publisher defines integration event publisher dependencies.
	publisher port.IntegrationEventPublisher
	// materializer defines optional carrier-submission dependencies used at batch close.
	materializer MarkMaterializer
	// quotationRepository defines quotation lookup dependencies for quotation-seeded batch marks.
	quotationRepository port.QuotationRepository
	// orderSource defines optional order lookup dependencies used to enrich quotation-seeded recipient data.
	orderSource port.OrderQuotationSource
	// defaultSender defines fallback sender values used by quotation-seeded batch mark creation.
	defaultSender domain.Address
	// manifestDocuments defines on-demand batch manifest document generation dependencies.
	manifestDocuments *batchManifestDocumentBuilder
}

// NewService creates dispatch batch services.
func NewService(batchRepository port.DispatchBatchRepository, markRepository port.ShippingMarkRepository, publisher port.IntegrationEventPublisher, materializers ...MarkMaterializer) *Service {
	var materializer MarkMaterializer
	if len(materializers) > 0 {
		materializer = materializers[0]
	}

	return &Service{
		batchRepository:   batchRepository,
		markRepository:    markRepository,
		publisher:         publisher,
		materializer:      materializer,
		manifestDocuments: newBatchManifestDocumentBuilder(),
	}
}

// SetQuotationRepository configures quotation lookup dependencies used by quotation-seeded batch mark creation.
func (s *Service) SetQuotationRepository(repository port.QuotationRepository) {
	if s == nil {
		return
	}

	s.quotationRepository = repository
}

// SetOrderSource configures optional order lookup dependencies used to enrich recipient fields.
func (s *Service) SetOrderSource(source port.OrderQuotationSource) {
	if s == nil {
		return
	}

	s.orderSource = source
}

// SetDefaultSender configures fallback sender values used by quotation-seeded batch mark creation.
func (s *Service) SetDefaultSender(sender domain.Address) {
	if s == nil {
		return
	}

	s.defaultSender = sender.Normalize()
}

// Create creates one dispatch batch.
func (s *Service) Create(ctx context.Context, command CreateBatchCommand) (*domain.DispatchBatch, error) {
	if s == nil || s.batchRepository == nil {
		return nil, domain.ErrInvalidID
	}
	batch := domain.DispatchBatch{
		ID:        uuid.NewString(),
		CarrierID: strings.TrimSpace(command.CarrierID),
		Status:    domain.BatchStatusOpen,
		CreatedBy: strings.TrimSpace(command.CreatedBy),
		CreatedAt: time.Now().UTC(),
	}.Normalize()
	if err := batch.Validate(); err != nil {
		return nil, err
	}
	if err := s.batchRepository.Create(ctx, &batch); err != nil {
		return nil, err
	}
	s.publish(ctx, dispatchevent.BuildBatchCreated(batch))

	return &batch, nil
}

// Get resolves one dispatch batch by identifier.
func (s *Service) Get(ctx context.Context, id string) (*domain.DispatchBatch, error) {
	if s == nil || s.batchRepository == nil {
		return nil, domain.ErrNotFound
	}

	return s.batchRepository.GetByID(ctx, strings.TrimSpace(id))
}

// List resolves dispatch batches with pagination and filters.
func (s *Service) List(ctx context.Context, query ListQuery) ([]domain.DispatchBatch, int64, error) {
	if s == nil || s.batchRepository == nil {
		return []domain.DispatchBatch{}, 0, nil
	}

	return s.batchRepository.List(ctx, port.BatchListQuery{
		CarrierID: strings.TrimSpace(query.CarrierID),
		Status:    query.Status,
		Page:      query.Page,
		Limit:     query.Limit,
	})
}

// DraftMark creates one QUOTED draft mark and assigns it to an open batch.
func (s *Service) DraftMark(ctx context.Context, command DraftMarkCommand) (*domain.ShippingMark, error) {
	if s == nil || s.batchRepository == nil || s.markRepository == nil {
		return nil, domain.ErrInvalidID
	}
	batch, err := s.batchRepository.GetByID(ctx, strings.TrimSpace(command.BatchID))
	if err != nil {
		return nil, err
	}
	if batch.Status != domain.BatchStatusOpen {
		return nil, domain.ErrBatchClosed
	}
	var quotationID *string
	if trimmed := strings.TrimSpace(command.QuotationID); trimmed != "" {
		quotationID = &trimmed
	}
	batchID := batch.ID
	mark := domain.ShippingMark{
		ID:                      uuid.NewString(),
		OrderID:                 strings.TrimSpace(command.OrderID),
		CarrierID:               batch.CarrierID,
		Status:                  domain.MarkStatusQuoted,
		Sender:                  command.Sender,
		Recipient:               command.Recipient,
		Units:                   command.Units,
		DeclaredValue:           command.DeclaredValue,
		PaymentForm:             strings.TrimSpace(command.PaymentForm),
		CollectOnDeliveryAmount: command.CollectOnDeliveryAmount,
		ShipmentMode:            command.ShipmentMode,
		Observations:            strings.TrimSpace(command.Observations),
		QuotationID:             quotationID,
		QuotedFreightCost:       command.QuotedFreightCost,
		CreatedAt:               time.Now().UTC(),
		UpdatedAt:               time.Now().UTC(),
	}.Normalize()
	if err := mark.Validate(); err != nil {
		return nil, err
	}
	if err := s.markRepository.Create(ctx, &mark); err != nil {
		return nil, err
	}
	if err := s.batchRepository.AddMark(ctx, batchID, mark.ID); err != nil {
		return nil, err
	}
	mark.DispatchBatchID = &batchID

	return &mark, nil
}

// CreateBatchMark creates one batch mark as draft (quoted) or direct (materialized immediately).
func (s *Service) CreateBatchMark(ctx context.Context, command CreateBatchMarkCommand) (*domain.ShippingMark, error) {
	if strings.TrimSpace(command.BatchID) == "" {
		return nil, domain.ErrInvalidID
	}
	if command.Direct {
		return s.createDirectBatchMark(ctx, command)
	}

	return s.DraftMark(ctx, DraftMarkCommand{
		BatchID:                 strings.TrimSpace(command.BatchID),
		QuotationID:             strings.TrimSpace(command.QuotationID),
		QuotedFreightCost:       command.QuotedFreightCost,
		OrderID:                 strings.TrimSpace(command.OrderID),
		Sender:                  command.Sender,
		Recipient:               command.Recipient,
		Units:                   command.Units,
		DeclaredValue:           command.DeclaredValue,
		PaymentForm:             strings.TrimSpace(command.PaymentForm),
		CollectOnDeliveryAmount: command.CollectOnDeliveryAmount,
		ShipmentMode:            command.ShipmentMode,
		Observations:            strings.TrimSpace(command.Observations),
	})
}

// CreateBatchMarkFromQuotation creates one batch mark from stored quotation data.
func (s *Service) CreateBatchMarkFromQuotation(ctx context.Context, command CreateBatchMarkFromQuotationCommand) (*domain.ShippingMark, error) {
	trimmedBatchID := strings.TrimSpace(command.BatchID)
	trimmedQuotationID := strings.TrimSpace(command.QuotationID)
	if trimmedBatchID == "" || trimmedQuotationID == "" {
		return nil, domain.ErrInvalidID
	}
	if s == nil || s.quotationRepository == nil {
		return nil, domain.ErrNotFound
	}

	record, err := s.quotationRepository.GetByID(ctx, trimmedQuotationID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, domain.ErrNotFound
	}
	requestSnapshot, err := decodeQuotationSnapshot(record.RequestSnapshot)
	if err != nil {
		return nil, err
	}
	units := record.Units
	if len(units) == 0 {
		units = requestSnapshot.Units
	}
	if len(units) == 0 {
		return nil, domain.ErrInvalidID
	}
	shipmentMode := requestSnapshot.ShipmentMode
	if shipmentMode != domain.ShipmentModeExpress && shipmentMode != domain.ShipmentModeParcel {
		shipmentMode = resolveShipmentMode(units)
	}
	declaredValue := requestSnapshot.DeclaredValue
	if declaredValue <= 0 {
		declaredValue = resolveDeclaredValue(units)
	}
	collectOnDeliveryAmount := requestSnapshot.CollectOnDeliveryAmount
	if collectOnDeliveryAmount < 0 {
		collectOnDeliveryAmount = 0
	}

	orderID := strings.TrimSpace(record.OrderID)
	if orderID == "" {
		orderID = strings.TrimSpace(requestSnapshot.OrderID)
	}
	recipient := domain.Address{
		Name:        "Cliente",
		CityCode:    firstNonEmptyString(strings.TrimSpace(record.DestCityCode), strings.TrimSpace(requestSnapshot.DestCityCode)),
		AddressLine: "",
	}
	lookupIdentifier := firstNonEmptyString(orderID, strings.TrimSpace(record.OrderIdentifier))
	if s.orderSource != nil && lookupIdentifier != "" {
		orderData, orderErr := s.orderSource.GetByIDOrIdentifier(ctx, lookupIdentifier)
		if orderErr == nil && orderData != nil {
			if orderID == "" {
				orderID = strings.TrimSpace(orderData.OrderID)
			}
			recipient.Name = firstNonEmptyString(strings.TrimSpace(orderData.RecipientName), recipient.Name)
			recipient.ID = strings.TrimSpace(orderData.RecipientID)
			recipient.IDType = strings.TrimSpace(orderData.RecipientIDType)
			recipient.AddressLine = strings.TrimSpace(orderData.RecipientAddressLine)
			recipient.CityCode = firstNonEmptyString(strings.TrimSpace(orderData.DestCityCode), recipient.CityCode)
			recipient.Phone = strings.TrimSpace(orderData.RecipientPhone)
			recipient.Email = strings.TrimSpace(orderData.RecipientEmail)
			if collectOnDeliveryAmount <= 0 {
				collectOnDeliveryAmount = maxZero(orderData.CollectOnDeliveryAmount)
			}
		}
	}
	if orderID == "" {
		return nil, domain.ErrInvalidID
	}

	return s.CreateBatchMark(ctx, CreateBatchMarkCommand{
		BatchID:                 trimmedBatchID,
		Direct:                  command.Direct,
		QuotationID:             trimmedQuotationID,
		QuotedFreightCost:       record.FreightCost,
		OrderID:                 orderID,
		Sender:                  s.defaultSender,
		Recipient:               recipient,
		Units:                   units,
		DeclaredValue:           declaredValue,
		PaymentForm:             "",
		CollectOnDeliveryAmount: collectOnDeliveryAmount,
		ShipmentMode:            shipmentMode,
		Observations:            "",
	})
}

// createDirectBatchMark creates one mark in a batch and materializes it immediately, even when the batch is closed.
func (s *Service) createDirectBatchMark(ctx context.Context, command CreateBatchMarkCommand) (*domain.ShippingMark, error) {
	if s == nil || s.batchRepository == nil || s.markRepository == nil || s.materializer == nil {
		return nil, domain.ErrCarrierNotSupported
	}
	batch, err := s.batchRepository.GetByID(ctx, strings.TrimSpace(command.BatchID))
	if err != nil {
		return nil, err
	}
	var quotationID *string
	if trimmed := strings.TrimSpace(command.QuotationID); trimmed != "" {
		quotationID = &trimmed
	}
	batchID := batch.ID
	mark := domain.ShippingMark{
		ID:                      uuid.NewString(),
		OrderID:                 strings.TrimSpace(command.OrderID),
		CarrierID:               batch.CarrierID,
		Status:                  domain.MarkStatusQuoted,
		Sender:                  command.Sender,
		Recipient:               command.Recipient,
		Units:                   command.Units,
		DeclaredValue:           command.DeclaredValue,
		PaymentForm:             strings.TrimSpace(command.PaymentForm),
		CollectOnDeliveryAmount: command.CollectOnDeliveryAmount,
		ShipmentMode:            command.ShipmentMode,
		Observations:            strings.TrimSpace(command.Observations),
		QuotationID:             quotationID,
		QuotedFreightCost:       command.QuotedFreightCost,
		DispatchBatchID:         &batchID,
		CreatedAt:               time.Now().UTC(),
		UpdatedAt:               time.Now().UTC(),
	}.Normalize()
	if err := mark.Validate(); err != nil {
		return nil, err
	}
	if err := s.markRepository.Create(ctx, &mark); err != nil {
		return nil, err
	}
	if err := s.batchRepository.AddMark(ctx, batchID, mark.ID); err != nil {
		return nil, err
	}
	if err := s.materializer.Materialize(ctx, &mark); err != nil {
		return nil, err
	}
	persisted, err := s.markRepository.GetByID(ctx, mark.ID)
	if err != nil {
		return nil, err
	}

	return persisted, nil
}

// RemoveDraftMark permanently deletes one QUOTED draft mark from a batch.
// Only marks in QUOTED status may be deleted; marks in any other status are rejected.
func (s *Service) RemoveDraftMark(ctx context.Context, batchID string, markID string) (*domain.DispatchBatch, error) {
	if s == nil || s.batchRepository == nil || s.markRepository == nil {
		return nil, domain.ErrInvalidID
	}
	mark, err := s.markRepository.GetByID(ctx, strings.TrimSpace(markID))
	if err != nil {
		return nil, err
	}
	if mark.Status != domain.MarkStatusQuoted {
		return nil, domain.ErrMarkNotDraft
	}
	if err := s.markRepository.Delete(ctx, mark.ID); err != nil {
		return nil, err
	}

	return s.batchRepository.GetByID(ctx, strings.TrimSpace(batchID))
}

// Close materializes all QUOTED marks in the batch then closes it.
func (s *Service) Close(ctx context.Context, batchID string) (*domain.DispatchBatch, error) {
	if s == nil || s.batchRepository == nil {
		return nil, domain.ErrInvalidID
	}
	trimmedID := strings.TrimSpace(batchID)
	if s.materializer != nil {
		marks, err := s.markRepository.ListByBatchID(ctx, trimmedID)
		if err != nil {
			return nil, err
		}
		for i := range marks {
			if marks[i].Status != domain.MarkStatusQuoted {
				continue
			}
			if err := s.materializer.Materialize(ctx, &marks[i]); err != nil {
				zap.L().Error("mark materialization failed", zap.String("batch_id", trimmedID), zap.String("mark_id", marks[i].ID), zap.String("order_id", marks[i].OrderID), zap.Error(err))
			}
		}
	}
	if err := s.batchRepository.Close(ctx, trimmedID); err != nil {
		return nil, err
	}
	s.invalidateBatchManifestDocumentCache(ctx, trimmedID)
	batch, err := s.batchRepository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	s.publish(ctx, dispatchevent.BuildBatchClosed(*batch))

	return batch, nil
}

// publish publishes one integration event and suppresses publication errors.
func (s *Service) publish(ctx context.Context, event port.IntegrationEvent) {
	if s == nil || s.publisher == nil {
		return
	}
	_ = s.publisher.Publish(ctx, event)
}

func decodeQuotationSnapshot(payload string) (domain.QuotationRequest, error) {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return domain.QuotationRequest{}, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return domain.QuotationRequest{}, err
	}
	request := domain.QuotationRequest{}
	if err := json.Unmarshal(decoded, &request); err != nil {
		return domain.QuotationRequest{}, err
	}

	return request.Normalize(), nil
}

func resolveShipmentMode(units []domain.PackageUnit) domain.ShipmentMode {
	if len(units) <= 1 {
		return domain.ShipmentModeExpress
	}

	return domain.ShipmentModeParcel
}

func resolveDeclaredValue(units []domain.PackageUnit) float64 {
	total := 0.0
	for _, unit := range units {
		total += unit.Normalize().Dimensions.DeclaredValueCOP
	}
	if total < 0 {
		return 0
	}

	return total
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func maxZero(value float64) float64 {
	if value < 0 {
		return 0
	}

	return value
}
