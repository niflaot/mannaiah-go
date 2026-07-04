package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	// TrackingNumber defines optional manual tracking-number values.
	TrackingNumber string
	// DocumentType defines optional manual document-type values.
	DocumentType domain.MarkDocumentType
	// DocumentRef defines optional manual document-reference values.
	DocumentRef string
	// ManifestType defines optional manual manifest document-type values.
	ManifestType domain.MarkDocumentType
	// ManifestRef defines optional manual manifest document-reference values.
	ManifestRef string
	// CustomTrackingURL defines an optional operator-provided tracking URL override for this mark.
	CustomTrackingURL *string
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
	// TrackingNumber defines optional manual tracking-number values.
	TrackingNumber string
	// DocumentType defines optional manual document-type values.
	DocumentType domain.MarkDocumentType
	// DocumentRef defines optional manual document-reference values.
	DocumentRef string
	// ManifestType defines optional manual manifest document-type values.
	ManifestType domain.MarkDocumentType
	// ManifestRef defines optional manual manifest document-reference values.
	ManifestRef string
	// CustomTrackingURL defines an optional operator-provided tracking URL override for this mark.
	CustomTrackingURL *string
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
	existingOpenBatches, _, err := s.batchRepository.List(ctx, port.BatchListQuery{
		CarrierID: strings.TrimSpace(command.CarrierID),
		Status:    domain.BatchStatusOpen,
		Page:      1,
		Limit:     1,
	})
	if err != nil {
		return nil, err
	}
	if len(existingOpenBatches) > 0 {
		return nil, domain.ErrBatchOpenForCarrier
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
	enrichedCommand := normalizeDraftMarkCommandForCarrier(
		s.enrichDraftMarkCommand(ctx, command),
		batch.CarrierID,
	)
	existingMark, err := s.findExistingActiveMark(ctx, batch.ID, strings.TrimSpace(enrichedCommand.OrderID), strings.TrimSpace(enrichedCommand.QuotationID))
	if err != nil {
		return nil, err
	}
	if existingMark != nil && isMarkInBatch(*existingMark, batch.ID) {
		return existingMark, nil
	}
	var quotationID *string
	if trimmed := strings.TrimSpace(enrichedCommand.QuotationID); trimmed != "" {
		quotationID = &trimmed
	}
	batchID := batch.ID
	mark := domain.ShippingMark{
		ID:                      uuid.NewString(),
		OrderID:                 strings.TrimSpace(enrichedCommand.OrderID),
		CarrierID:               batch.CarrierID,
		Status:                  domain.MarkStatusQuoted,
		Sender:                  enrichedCommand.Sender,
		Recipient:               enrichedCommand.Recipient,
		Units:                   enrichedCommand.Units,
		DeclaredValue:           enrichedCommand.DeclaredValue,
		PaymentForm:             strings.TrimSpace(enrichedCommand.PaymentForm),
		CollectOnDeliveryAmount: enrichedCommand.CollectOnDeliveryAmount,
		ShipmentMode:            enrichedCommand.ShipmentMode,
		Observations:            strings.TrimSpace(enrichedCommand.Observations),
		TrackingNumber:          strings.TrimSpace(enrichedCommand.TrackingNumber),
		DocumentType:            enrichedCommand.DocumentType,
		DocumentRef:             strings.TrimSpace(enrichedCommand.DocumentRef),
		ManifestType:            enrichedCommand.ManifestType,
		ManifestRef:             strings.TrimSpace(enrichedCommand.ManifestRef),
		CustomTrackingURL:       normalizeOptionalURLPointer(enrichedCommand.CustomTrackingURL),
		QuotationID:             quotationID,
		QuotedFreightCost:       enrichedCommand.QuotedFreightCost,
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
	trimmedBatchID := strings.TrimSpace(command.BatchID)
	trimmedOrderID := strings.TrimSpace(command.OrderID)
	trimmedQuotationID := strings.TrimSpace(command.QuotationID)
	if trimmedBatchID == "" {
		return nil, domain.ErrInvalidID
	}
	if trimmedOrderID == "" {
		return nil, domain.ErrInvalidID
	}
	existingMark, err := s.findExistingActiveMark(ctx, trimmedBatchID, trimmedOrderID, trimmedQuotationID)
	if err != nil {
		return nil, err
	}
	if existingMark != nil && isMarkInBatch(*existingMark, trimmedBatchID) {
		if command.Direct && existingMark.Status == domain.MarkStatusQuoted {
			if s.materializer == nil {
				return existingMark, nil
			}
			copy := *existingMark
			if err := s.materializer.Materialize(ctx, &copy); err != nil {
				return nil, err
			}
			persisted, loadErr := s.markRepository.GetByID(ctx, copy.ID)
			if loadErr != nil {
				return &copy, nil
			}

			return persisted, nil
		}

		return existingMark, nil
	}
	if command.Direct {
		return s.createDirectBatchMark(ctx, command)
	}

	return s.DraftMark(ctx, DraftMarkCommand{
		BatchID:                 trimmedBatchID,
		QuotationID:             trimmedQuotationID,
		QuotedFreightCost:       command.QuotedFreightCost,
		OrderID:                 trimmedOrderID,
		Sender:                  command.Sender,
		Recipient:               command.Recipient,
		Units:                   command.Units,
		DeclaredValue:           command.DeclaredValue,
		PaymentForm:             strings.TrimSpace(command.PaymentForm),
		CollectOnDeliveryAmount: command.CollectOnDeliveryAmount,
		ShipmentMode:            command.ShipmentMode,
		Observations:            strings.TrimSpace(command.Observations),
		TrackingNumber:          strings.TrimSpace(command.TrackingNumber),
		DocumentType:            command.DocumentType,
		DocumentRef:             strings.TrimSpace(command.DocumentRef),
		ManifestType:            command.ManifestType,
		ManifestRef:             strings.TrimSpace(command.ManifestRef),
		CustomTrackingURL:       normalizeOptionalURLPointer(command.CustomTrackingURL),
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
			recipient.CityCode = firstNonEmptyString(recipient.CityCode, strings.TrimSpace(orderData.DestCityCode))
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
	enrichedCommand := normalizeCreateBatchMarkCommandForCarrier(
		s.enrichCreateBatchMarkCommand(ctx, command),
		batch.CarrierID,
	)
	var quotationID *string
	if trimmed := strings.TrimSpace(enrichedCommand.QuotationID); trimmed != "" {
		quotationID = &trimmed
	}
	batchID := batch.ID
	mark := domain.ShippingMark{
		ID:                      uuid.NewString(),
		OrderID:                 strings.TrimSpace(enrichedCommand.OrderID),
		CarrierID:               batch.CarrierID,
		Status:                  domain.MarkStatusQuoted,
		Sender:                  enrichedCommand.Sender,
		Recipient:               enrichedCommand.Recipient,
		Units:                   enrichedCommand.Units,
		DeclaredValue:           enrichedCommand.DeclaredValue,
		PaymentForm:             strings.TrimSpace(enrichedCommand.PaymentForm),
		CollectOnDeliveryAmount: enrichedCommand.CollectOnDeliveryAmount,
		ShipmentMode:            enrichedCommand.ShipmentMode,
		Observations:            strings.TrimSpace(enrichedCommand.Observations),
		TrackingNumber:          strings.TrimSpace(enrichedCommand.TrackingNumber),
		DocumentType:            enrichedCommand.DocumentType,
		DocumentRef:             strings.TrimSpace(enrichedCommand.DocumentRef),
		ManifestType:            enrichedCommand.ManifestType,
		ManifestRef:             strings.TrimSpace(enrichedCommand.ManifestRef),
		CustomTrackingURL:       normalizeOptionalURLPointer(enrichedCommand.CustomTrackingURL),
		QuotationID:             quotationID,
		QuotedFreightCost:       enrichedCommand.QuotedFreightCost,
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

func (s *Service) enrichCreateBatchMarkCommand(ctx context.Context, command CreateBatchMarkCommand) CreateBatchMarkCommand {
	base := CreateBatchMarkCommand{
		BatchID:                 strings.TrimSpace(command.BatchID),
		Direct:                  command.Direct,
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
		TrackingNumber:          strings.TrimSpace(command.TrackingNumber),
		DocumentType:            command.DocumentType,
		DocumentRef:             strings.TrimSpace(command.DocumentRef),
		ManifestType:            command.ManifestType,
		ManifestRef:             strings.TrimSpace(command.ManifestRef),
		CustomTrackingURL:       normalizeOptionalURLPointer(command.CustomTrackingURL),
	}
	enrichedDraft := DraftMarkCommand{
		OrderID:                 base.OrderID,
		Sender:                  base.Sender,
		Recipient:               base.Recipient,
		DeclaredValue:           base.DeclaredValue,
		CollectOnDeliveryAmount: base.CollectOnDeliveryAmount,
	}.enrichWithOrderData(ctx, s)
	base.OrderID = enrichedDraft.OrderID
	base.Sender = enrichedDraft.Sender
	base.Recipient = enrichedDraft.Recipient
	base.DeclaredValue = enrichedDraft.DeclaredValue
	base.CollectOnDeliveryAmount = enrichedDraft.CollectOnDeliveryAmount

	return base
}

func (s *Service) enrichDraftMarkCommand(ctx context.Context, command DraftMarkCommand) DraftMarkCommand {
	return DraftMarkCommand{
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
		TrackingNumber:          strings.TrimSpace(command.TrackingNumber),
		DocumentType:            command.DocumentType,
		DocumentRef:             strings.TrimSpace(command.DocumentRef),
		ManifestType:            command.ManifestType,
		ManifestRef:             strings.TrimSpace(command.ManifestRef),
		CustomTrackingURL:       normalizeOptionalURLPointer(command.CustomTrackingURL),
	}.enrichWithOrderData(ctx, s)
}

func (c DraftMarkCommand) enrichWithOrderData(ctx context.Context, service *Service) DraftMarkCommand {
	result := c
	result.Sender = resolveSenderWithDefault(c.Sender.Normalize(), service)
	result.Recipient = c.Recipient.Normalize()
	if service == nil || service.orderSource == nil || strings.TrimSpace(result.OrderID) == "" {
		return result
	}
	orderData, err := service.orderSource.GetByIDOrIdentifier(ctx, strings.TrimSpace(result.OrderID))
	if err != nil || orderData == nil {
		return result
	}
	if trimmedOrderID := strings.TrimSpace(orderData.OrderID); trimmedOrderID != "" {
		result.OrderID = trimmedOrderID
	}
	result.Recipient = mergeRecipientWithOrderData(result.Recipient, orderData)
	if result.CollectOnDeliveryAmount <= 0 {
		result.CollectOnDeliveryAmount = maxZero(orderData.CollectOnDeliveryAmount)
	}
	if result.DeclaredValue <= 0 {
		result.DeclaredValue = maxZero(orderData.TotalValue)
	}

	return result
}

func resolveSenderWithDefault(sender domain.Address, service *Service) domain.Address {
	if !isAddressEmpty(sender) {
		return sender
	}
	if service == nil || isAddressEmpty(service.defaultSender) {
		return sender
	}

	return service.defaultSender.Normalize()
}

func mergeRecipientWithOrderData(recipient domain.Address, orderData *port.OrderQuotationData) domain.Address {
	if orderData == nil {
		return recipient.Normalize()
	}
	normalized := recipient.Normalize()
	normalized.Name = firstNonEmptyString(normalized.Name, strings.TrimSpace(orderData.RecipientName), "Cliente")
	normalized.ID = firstNonEmptyString(normalized.ID, strings.TrimSpace(orderData.RecipientID))
	normalized.IDType = firstNonEmptyString(normalized.IDType, strings.TrimSpace(orderData.RecipientIDType))
	normalized.AddressLine = firstNonEmptyString(normalized.AddressLine, strings.TrimSpace(orderData.RecipientAddressLine))
	normalized.CityCode = firstNonEmptyString(normalized.CityCode, strings.TrimSpace(orderData.DestCityCode))
	normalized.Phone = firstNonEmptyString(normalized.Phone, strings.TrimSpace(orderData.RecipientPhone))
	normalized.Email = firstNonEmptyString(normalized.Email, strings.TrimSpace(orderData.RecipientEmail))

	return normalized.Normalize()
}

func isAddressEmpty(address domain.Address) bool {
	normalized := address.Normalize()
	return strings.TrimSpace(normalized.Name) == "" &&
		strings.TrimSpace(normalized.LegalName) == "" &&
		strings.TrimSpace(normalized.ID) == "" &&
		strings.TrimSpace(normalized.IDType) == "" &&
		strings.TrimSpace(normalized.AddressLine) == "" &&
		strings.TrimSpace(normalized.CityCode) == "" &&
		strings.TrimSpace(normalized.Phone) == "" &&
		strings.TrimSpace(normalized.Email) == ""
}

func normalizeOptionalURLPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func normalizeDraftMarkCommandForCarrier(command DraftMarkCommand, carrierID string) DraftMarkCommand {
	if !domain.IsManualCarrierID(carrierID) {
		return command
	}
	result := command
	result.Observations = domain.NormalizeCarrierSlug(result.Observations)
	if len(result.Units) == 0 {
		result.Units = []domain.PackageUnit{buildManualPlaceholderUnit()}
	}
	if result.ShipmentMode != domain.ShipmentModeParcel && result.ShipmentMode != domain.ShipmentModeExpress {
		result.ShipmentMode = domain.ShipmentModeExpress
	}

	return result
}

func normalizeCreateBatchMarkCommandForCarrier(command CreateBatchMarkCommand, carrierID string) CreateBatchMarkCommand {
	if !domain.IsManualCarrierID(carrierID) {
		return command
	}
	result := command
	result.Observations = domain.NormalizeCarrierSlug(result.Observations)
	if len(result.Units) == 0 {
		result.Units = []domain.PackageUnit{buildManualPlaceholderUnit()}
	}
	if result.ShipmentMode != domain.ShipmentModeParcel && result.ShipmentMode != domain.ShipmentModeExpress {
		result.ShipmentMode = domain.ShipmentModeExpress
	}

	return result
}

func buildManualPlaceholderUnit() domain.PackageUnit {
	return domain.PackageUnit{
		Description: "Manual tracking entry",
		PackageType: "CAJA",
		Dimensions: domain.Dimensions{
			HeightCM:     1,
			WidthCM:      1,
			DepthCM:      1,
			RealWeightKG: 0.1,
		},
	}
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
	batch, err := s.batchRepository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	if batch.Status == domain.BatchStatusClosed {
		return nil, domain.ErrBatchClosed
	}
	marks, err := s.markRepository.ListByBatchID(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	if domain.IsManualCarrierID(batch.CarrierID) {
		if err := s.ValidateManualDraftsBeforeClose(marks); err != nil {
			return nil, err
		}
	}
	if s.materializer != nil {
		for i := range marks {
			if marks[i].Status != domain.MarkStatusQuoted {
				continue
			}
			if err := s.materializer.Materialize(ctx, &marks[i]); err != nil {
				zap.L().Error("mark materialization failed", zap.String("batch_id", trimmedID), zap.String("mark_id", marks[i].ID), zap.String("order_id", marks[i].OrderID), zap.Error(err))
				var guardrailErr *domain.GuardrailViolationError
				if errors.As(err, &guardrailErr) {
					return nil, err
				}
			}
		}
	}
	if err := s.batchRepository.Close(ctx, trimmedID); err != nil {
		return nil, err
	}
	s.invalidateBatchManifestDocumentCache(ctx, trimmedID)
	batch, err = s.batchRepository.GetByID(ctx, trimmedID)
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

func isActiveMarkStatus(status domain.MarkStatus) bool {
	switch status {
	case domain.MarkStatusPending, domain.MarkStatusQuoted, domain.MarkStatusGenerated, domain.MarkStatusCreated:
		return true
	default:
		return false
	}
}

func markStatusPriority(status domain.MarkStatus) int {
	switch status {
	case domain.MarkStatusCreated:
		return 4
	case domain.MarkStatusGenerated:
		return 3
	case domain.MarkStatusQuoted:
		return 2
	case domain.MarkStatusPending:
		return 1
	default:
		return 0
	}
}

func isMarkInBatch(mark domain.ShippingMark, batchID string) bool {
	if mark.DispatchBatchID == nil {
		return false
	}

	return strings.TrimSpace(*mark.DispatchBatchID) == strings.TrimSpace(batchID)
}

func (s *Service) findExistingActiveMark(ctx context.Context, batchID string, orderID string, quotationID string) (*domain.ShippingMark, error) {
	if s == nil || s.markRepository == nil {
		return nil, nil
	}
	trimmedOrderID := strings.TrimSpace(orderID)
	if trimmedOrderID == "" {
		return nil, nil
	}
	rows, err := s.markRepository.ListByOrderID(ctx, trimmedOrderID)
	if err != nil {
		return nil, err
	}

	var selected *domain.ShippingMark
	trimmedBatchID := strings.TrimSpace(batchID)
	trimmedQuotationID := strings.TrimSpace(quotationID)
	for i := range rows {
		current := rows[i]
		if !isActiveMarkStatus(current.Status) {
			continue
		}
		if trimmedQuotationID != "" {
			currentQuotationID := ""
			if current.QuotationID != nil {
				currentQuotationID = strings.TrimSpace(*current.QuotationID)
			}
			if currentQuotationID != trimmedQuotationID {
				continue
			}
		}
		if selected == nil {
			candidate := current
			selected = &candidate
			continue
		}

		selectedBatchMatch := isMarkInBatch(*selected, trimmedBatchID)
		currentBatchMatch := isMarkInBatch(current, trimmedBatchID)
		if currentBatchMatch && !selectedBatchMatch {
			candidate := current
			selected = &candidate
			continue
		}
		if currentBatchMatch == selectedBatchMatch && markStatusPriority(current.Status) > markStatusPriority(selected.Status) {
			candidate := current
			selected = &candidate
		}
	}

	if selected != nil {
		return selected, nil
	}

	if trimmedQuotationID != "" {
		return nil, nil
	}

	for i := range rows {
		current := rows[i]
		if !isActiveMarkStatus(current.Status) {
			continue
		}
		if selected == nil {
			candidate := current
			selected = &candidate
			continue
		}
		selectedBatchMatch := isMarkInBatch(*selected, trimmedBatchID)
		currentBatchMatch := isMarkInBatch(current, trimmedBatchID)
		if currentBatchMatch && !selectedBatchMatch {
			candidate := current
			selected = &candidate
			continue
		}
		if currentBatchMatch == selectedBatchMatch && markStatusPriority(current.Status) > markStatusPriority(selected.Status) {
			candidate := current
			selected = &candidate
		}
	}

	return selected, nil
}
