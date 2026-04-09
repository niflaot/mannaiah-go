package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	markevent "mannaiah/module/shipping/application/mark/event"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// GenerateCommand defines shipping mark generation input values.
type GenerateCommand struct {
	// OrderID defines order identifier values.
	OrderID string
	// CarrierID defines carrier identifier values.
	CarrierID string
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
	// ShipmentMode defines the delivery mode for this mark.
	ShipmentMode domain.ShipmentMode
	// Observations defines optional observation values.
	Observations string
	// TrackingNumber defines optional manual tracking-number values.
	TrackingNumber string
	// DocumentType defines optional manual document-type values.
	DocumentType domain.MarkDocumentType
	// DocumentRef defines optional manual document reference values.
	DocumentRef string
	// ManifestType defines optional manual manifest document-type values.
	ManifestType domain.MarkDocumentType
	// ManifestRef defines optional manual manifest document reference values.
	ManifestRef string
	// CustomTrackingURL defines an optional operator-provided tracking URL override for this mark.
	CustomTrackingURL *string
}

// ListQuery defines mark listing query values.
type ListQuery struct {
	// OrderID filters marks by order identifier.
	OrderID string
	// BatchID filters marks by batch identifier.
	BatchID string
	// Page defines 1-based page values.
	Page int
	// Limit defines page-size values.
	Limit int
}

// Service defines shipping-mark orchestration behavior.
type Service struct {
	// repository defines mark repository dependencies.
	repository port.ShippingMarkRepository
	// registry defines carrier registry dependencies.
	registry port.ProviderRegistry
	// publisher defines integration event publisher dependencies.
	publisher port.IntegrationEventPublisher
	// orderSource defines optional order lookup dependencies used by rotulus rendering.
	orderSource port.OrderQuotationSource
	// rotulusDocuments defines on-demand rotulus document generation dependencies.
	rotulusDocuments *markRotulusDocumentBuilder
}

// NewService creates shipping-mark services.
func NewService(repository port.ShippingMarkRepository, registry port.ProviderRegistry, publisher port.IntegrationEventPublisher) *Service {
	return &Service{
		repository:       repository,
		registry:         registry,
		publisher:        publisher,
		rotulusDocuments: newMarkRotulusDocumentBuilder(),
	}
}

// SetOrderSource configures optional order lookup dependencies used by mark adjunct documents.
func (s *Service) SetOrderSource(source port.OrderQuotationSource) {
	if s == nil {
		return
	}

	s.orderSource = source
}

// Generate creates one shipping mark through the configured carrier provider.
func (s *Service) Generate(ctx context.Context, command GenerateCommand) (*domain.ShippingMark, error) {
	if s == nil || s.repository == nil || s.registry == nil {
		return nil, domain.ErrCarrierNotSupported
	}
	provider, exists := s.registry.CarrierProvider(command.CarrierID)
	if !exists || provider == nil {
		return nil, domain.ErrCarrierNotSupported
	}

	mark := domain.ShippingMark{
		ID:                      uuid.NewString(),
		OrderID:                 strings.TrimSpace(command.OrderID),
		CarrierID:               strings.TrimSpace(command.CarrierID),
		Status:                  domain.MarkStatusPending,
		DocumentType:            command.DocumentType,
		DocumentRef:             strings.TrimSpace(command.DocumentRef),
		ManifestType:            command.ManifestType,
		ManifestRef:             strings.TrimSpace(command.ManifestRef),
		TrackingNumber:          strings.TrimSpace(command.TrackingNumber),
		Sender:                  command.Sender,
		Recipient:               command.Recipient,
		Units:                   command.Units,
		DeclaredValue:           command.DeclaredValue,
		PaymentForm:             strings.TrimSpace(command.PaymentForm),
		CollectOnDeliveryAmount: command.CollectOnDeliveryAmount,
		ShipmentMode:            command.ShipmentMode,
		Observations:            normalizeManualCarrierObservations(command.CarrierID, command.Observations),
		CustomTrackingURL:       command.CustomTrackingURL,
		CreatedAt:               time.Now().UTC(),
		UpdatedAt:               time.Now().UTC(),
	}.Normalize()
	if mark.DraftSnapshot == "" {
		mark.DraftSnapshot = encodeMarkSnapshot(mark)
	}
	if err := mark.Validate(); err != nil {
		return nil, err
	}
	if provider.Carrier().RequiresBalanceCheck {
		if err := provider.CheckBalance(ctx); err != nil {
			return nil, domain.ErrInsufficientBalance
		}
	}

	if err := provider.GenerateMark(ctx, &mark); err != nil {
		mark.Status = domain.MarkStatusFailed
		mark.FailureReason = err.Error()
		if mark.ResponseSnapshot == "" {
			mark.ResponseSnapshot = encodeMarkSnapshot(mark)
		}
		mark.UpdatedAt = time.Now().UTC()
		if createErr := s.repository.Create(ctx, &mark); createErr != nil {
			return nil, createErr
		}
		s.publish(ctx, markevent.BuildMarkFailed(mark, err.Error()))

		return nil, err
	}
	if mark.Status == "" {
		mark.Status = domain.MarkStatusGenerated
	}
	if mark.ResponseSnapshot == "" {
		mark.ResponseSnapshot = encodeMarkSnapshot(mark)
	}
	mark.UpdatedAt = time.Now().UTC()
	if err := s.repository.Create(ctx, &mark); err != nil {
		return nil, err
	}
	s.publish(ctx, markevent.BuildMarkGenerated(mark))

	return &mark, nil
}

// normalizeManualCarrierObservations normalizes manual carrier observations into stable slug values.
func normalizeManualCarrierObservations(carrierID string, value string) string {
	trimmed := strings.TrimSpace(value)
	if !domain.IsManualCarrierID(carrierID) {
		return trimmed
	}

	return domain.NormalizeCarrierSlug(trimmed)
}

// Get resolves one shipping mark by identifier.
func (s *Service) Get(ctx context.Context, id string) (*domain.ShippingMark, error) {
	if s == nil || s.repository == nil {
		return nil, domain.ErrNotFound
	}

	return s.repository.GetByID(ctx, strings.TrimSpace(id))
}

// List resolves shipping marks with pagination and optional filters.
func (s *Service) List(ctx context.Context, query ListQuery) ([]domain.ShippingMark, int64, error) {
	if s == nil || s.repository == nil {
		return []domain.ShippingMark{}, 0, nil
	}

	return s.repository.List(ctx, port.MarkListQuery{
		OrderID: strings.TrimSpace(query.OrderID),
		BatchID: strings.TrimSpace(query.BatchID),
		Page:    query.Page,
		Limit:   query.Limit,
	})
}

// Related resolves shipping marks related to one mark identifier.
// Related marks are those that share the same order id or dispatch batch id.
// The target mark itself is excluded from the output.
func (s *Service) Related(ctx context.Context, id string) ([]domain.ShippingMark, error) {
	if s == nil || s.repository == nil {
		return []domain.ShippingMark{}, nil
	}

	target, err := s.repository.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	related := map[string]domain.ShippingMark{}
	appendRows := func(rows []domain.ShippingMark) {
		for _, row := range rows {
			trimmedRowID := strings.TrimSpace(row.ID)
			if trimmedRowID == "" || trimmedRowID == strings.TrimSpace(target.ID) {
				continue
			}
			related[trimmedRowID] = row
		}
	}

	orderRows, orderErr := s.repository.ListByOrderID(ctx, strings.TrimSpace(target.OrderID))
	if orderErr != nil {
		return nil, orderErr
	}
	appendRows(orderRows)

	if target.DispatchBatchID != nil && strings.TrimSpace(*target.DispatchBatchID) != "" {
		batchRows, batchErr := s.repository.ListByBatchID(ctx, strings.TrimSpace(*target.DispatchBatchID))
		if batchErr != nil {
			return nil, batchErr
		}
		appendRows(batchRows)
	}

	result := make([]domain.ShippingMark, 0, len(related))
	for _, row := range related {
		result = append(result, row)
	}
	sort.SliceStable(result, func(i int, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return strings.ToLower(strings.TrimSpace(result[i].ID)) > strings.ToLower(strings.TrimSpace(result[j].ID))
		}

		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

// DispatchQuery defines order dispatch provisioning check query values.
type DispatchQuery struct {
	// OrderID defines the order identifier to check.
	OrderID string
}

// DispatchResult defines the dispatch provisioning status of one order.
type DispatchResult struct {
	// OrderID defines the queried order identifier.
	OrderID string
	// Provisioned reports whether the order has an active mark in the dispatch workflow.
	Provisioned bool
	// MarkID defines the active mark identifier when provisioned.
	MarkID string
	// BatchID defines the associated batch identifier when provisioned.
	BatchID *string
	// Status defines the active mark status when provisioned.
	Status domain.MarkStatus
}

// dispatchPriority returns the selection priority of one mark status.
// Higher values win. Zero means the status is not considered active.
func dispatchPriority(s domain.MarkStatus) int {
	switch s {
	case domain.MarkStatusQuoted:
		return 3
	case domain.MarkStatusCreated:
		return 2
	case domain.MarkStatusGenerated:
		return 1
	}

	return 0
}

// QueryDispatch resolves the dispatch provisioning status for one order.
func (s *Service) QueryDispatch(ctx context.Context, query DispatchQuery) (*DispatchResult, error) {
	if s == nil || s.repository == nil {
		return &DispatchResult{OrderID: strings.TrimSpace(query.OrderID)}, nil
	}
	trimmed := strings.TrimSpace(query.OrderID)
	marks, err := s.repository.ListByOrderID(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	var best *domain.ShippingMark
	for i := range marks {
		m := &marks[i]
		if dispatchPriority(m.Status) == 0 {
			continue
		}
		if best == nil || dispatchPriority(m.Status) > dispatchPriority(best.Status) {
			best = m
		}
	}
	result := &DispatchResult{OrderID: trimmed}
	if best != nil {
		result.Provisioned = true
		result.MarkID = best.ID
		result.BatchID = best.DispatchBatchID
		result.Status = best.Status
	}

	return result, nil
}

// Void voids one shipping mark with a local status change only.
// No carrier API call is made; carrier-level void must be handled out-of-band when the carrier supports it.
func (s *Service) Void(ctx context.Context, id string, reason string) (*domain.ShippingMark, error) {
	if s == nil || s.repository == nil {
		return nil, domain.ErrNotFound
	}
	mark, err := s.repository.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	mark.Status = domain.MarkStatusVoided
	mark.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, mark); err != nil {
		return nil, err
	}
	s.publish(ctx, markevent.BuildMarkVoided(*mark, strings.TrimSpace(reason)))

	return mark, nil
}

// Materialize submits one QUOTED draft mark to the carrier and updates its status to CREATED or FAILED.
// Base64-encoded JSON snapshots are captured before submission (DraftSnapshot) and after response handling (ResponseSnapshot).
func (s *Service) Materialize(ctx context.Context, mark *domain.ShippingMark) error {
	if s == nil || s.repository == nil || s.registry == nil {
		return domain.ErrCarrierNotSupported
	}
	provider, exists := s.registry.CarrierProvider(mark.CarrierID)
	if !exists || provider == nil {
		return domain.ErrCarrierNotSupported
	}
	if strings.TrimSpace(mark.DraftSnapshot) == "" {
		mark.DraftSnapshot = encodeMarkSnapshot(*mark)
	}
	if provider.Carrier().RequiresBalanceCheck {
		if err := provider.CheckBalance(ctx); err != nil {
			mark.Status = domain.MarkStatusFailed
			mark.FailureReason = domain.ErrInsufficientBalance.Error()
			if strings.TrimSpace(mark.ResponseSnapshot) == "" {
				mark.ResponseSnapshot = encodeMarkSnapshot(*mark)
			}
			mark.UpdatedAt = time.Now().UTC()
			_ = s.repository.Update(ctx, mark)
			s.publish(ctx, markevent.BuildMarkFailed(*mark, domain.ErrInsufficientBalance.Error()))

			return domain.ErrInsufficientBalance
		}
	}
	if err := provider.GenerateMark(ctx, mark); err != nil {
		mark.Status = domain.MarkStatusFailed
		mark.FailureReason = err.Error()
		if strings.TrimSpace(mark.ResponseSnapshot) == "" {
			mark.ResponseSnapshot = encodeMarkSnapshot(*mark)
		}
		mark.UpdatedAt = time.Now().UTC()
		_ = s.repository.Update(ctx, mark)
		s.publish(ctx, markevent.BuildMarkFailed(*mark, err.Error()))

		return err
	}
	mark.Status = domain.MarkStatusCreated
	if strings.TrimSpace(mark.ResponseSnapshot) == "" {
		mark.ResponseSnapshot = encodeMarkSnapshot(*mark)
	}
	mark.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, mark); err != nil {
		return err
	}
	s.publish(ctx, markevent.BuildMarkGenerated(*mark))

	return nil
}

// publish publishes one integration event and suppresses publication errors.
func (s *Service) publish(ctx context.Context, event port.IntegrationEvent) {
	if s == nil || s.publisher == nil {
		return
	}
	_ = s.publisher.Publish(ctx, event)
}

func encodeMarkSnapshot(mark domain.ShippingMark) string {
	snapshot := mark.Normalize()
	snapshot.DraftSnapshot = ""
	snapshot.RequestSnapshot = ""
	snapshot.ResponseSnapshot = ""
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(payload)
}
