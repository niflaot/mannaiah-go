package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

type dispatchBatchRepositoryStub struct {
	batches   map[string]domain.DispatchBatch
	markStore *dispatchMarkRepositoryStub
}

func newDispatchBatchRepositoryStub() *dispatchBatchRepositoryStub {
	return &dispatchBatchRepositoryStub{batches: map[string]domain.DispatchBatch{}}
}

func (s *dispatchBatchRepositoryStub) Create(ctx context.Context, batch *domain.DispatchBatch) error {
	s.batches[batch.ID] = *batch

	return nil
}
func (s *dispatchBatchRepositoryStub) GetByID(ctx context.Context, id string) (*domain.DispatchBatch, error) {
	row, exists := s.batches[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	copy := row
	if s.markStore != nil {
		markIDs := make([]string, 0)
		for _, m := range s.markStore.marks {
			if m.DispatchBatchID != nil && *m.DispatchBatchID == id {
				markIDs = append(markIDs, m.ID)
			}
		}
		copy.MarkIDs = markIDs
	}

	return &copy, nil
}
func (s *dispatchBatchRepositoryStub) Close(ctx context.Context, id string) error {
	row, exists := s.batches[id]
	if !exists {
		return domain.ErrNotFound
	}
	row.Status = domain.BatchStatusClosed
	s.batches[id] = row

	return nil
}
func (s *dispatchBatchRepositoryStub) AddMark(ctx context.Context, batchID string, markID string) error {
	row := s.batches[batchID]
	row.MarkIDs = append(row.MarkIDs, markID)
	s.batches[batchID] = row
	if s.markStore != nil {
		if mark, exists := s.markStore.marks[markID]; exists {
			mark.DispatchBatchID = &batchID
			s.markStore.marks[markID] = mark
		}
	}

	return nil
}
func (s *dispatchBatchRepositoryStub) RemoveMark(ctx context.Context, batchID string, markID string) error {
	row := s.batches[batchID]
	updated := make([]string, 0, len(row.MarkIDs))
	for _, current := range row.MarkIDs {
		if current != markID {
			updated = append(updated, current)
		}
	}
	row.MarkIDs = updated
	s.batches[batchID] = row

	return nil
}
func (s *dispatchBatchRepositoryStub) List(ctx context.Context, query port.BatchListQuery) ([]domain.DispatchBatch, int64, error) {
	rows := make([]domain.DispatchBatch, 0, len(s.batches))
	for _, row := range s.batches {
		if query.CarrierID != "" && !strings.EqualFold(row.CarrierID, query.CarrierID) {
			continue
		}
		if query.Status != "" && row.Status != query.Status {
			continue
		}
		rows = append(rows, row)
	}

	return rows, int64(len(rows)), nil
}

type dispatchMarkRepositoryStub struct {
	marks map[string]domain.ShippingMark
}

func newDispatchMarkRepositoryStub() *dispatchMarkRepositoryStub {
	return &dispatchMarkRepositoryStub{marks: map[string]domain.ShippingMark{}}
}

func (s *dispatchMarkRepositoryStub) Create(ctx context.Context, mark *domain.ShippingMark) error {
	s.marks[mark.ID] = *mark

	return nil
}
func (s *dispatchMarkRepositoryStub) GetByID(ctx context.Context, id string) (*domain.ShippingMark, error) {
	row, exists := s.marks[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	copy := row

	return &copy, nil
}
func (s *dispatchMarkRepositoryStub) GetByTrackingNumber(ctx context.Context, trackingNumber string) (*domain.ShippingMark, error) {
	return nil, domain.ErrNotFound
}
func (s *dispatchMarkRepositoryStub) ListByOrderID(ctx context.Context, orderID string) ([]domain.ShippingMark, error) {
	result := make([]domain.ShippingMark, 0)
	for _, m := range s.marks {
		if m.OrderID == orderID {
			result = append(result, m)
		}
	}

	return result, nil
}
func (s *dispatchMarkRepositoryStub) ListByBatchID(ctx context.Context, batchID string) ([]domain.ShippingMark, error) {
	result := make([]domain.ShippingMark, 0)
	for _, m := range s.marks {
		if m.DispatchBatchID != nil && *m.DispatchBatchID == batchID {
			result = append(result, m)
		}
	}

	return result, nil
}
func (s *dispatchMarkRepositoryStub) Update(ctx context.Context, mark *domain.ShippingMark) error {
	s.marks[mark.ID] = *mark

	return nil
}
func (s *dispatchMarkRepositoryStub) Delete(ctx context.Context, id string) error {
	delete(s.marks, id)

	return nil
}
func (s *dispatchMarkRepositoryStub) List(ctx context.Context, query port.MarkListQuery) ([]domain.ShippingMark, int64, error) {
	return nil, 0, nil
}

type materializerStub struct {
	calls      int
	repository port.ShippingMarkRepository
}

func (s *materializerStub) Materialize(ctx context.Context, mark *domain.ShippingMark) error {
	s.calls++
	mark.Status = domain.MarkStatusCreated
	mark.TrackingNumber = "TRACK-" + mark.ID
	if s.repository != nil {
		_ = s.repository.Update(ctx, mark)
	}

	return nil
}

type materializerErrorStub struct {
	calls int
	err   error
}

func (s *materializerErrorStub) Materialize(ctx context.Context, mark *domain.ShippingMark) error {
	s.calls++
	return s.err
}

type dispatchPublisherStub struct {
	events []port.IntegrationEvent
}

func (s *dispatchPublisherStub) Publish(ctx context.Context, event port.IntegrationEvent) error {
	s.events = append(s.events, event)

	return nil
}

type dispatchQuotationRepositoryStub struct {
	rows map[string]port.QuotationRecord
}

func newDispatchQuotationRepositoryStub() *dispatchQuotationRepositoryStub {
	return &dispatchQuotationRepositoryStub{rows: map[string]port.QuotationRecord{}}
}

func (s *dispatchQuotationRepositoryStub) Create(ctx context.Context, record port.QuotationRecord) (string, error) {
	s.rows[record.ID] = record
	return record.ID, nil
}

func (s *dispatchQuotationRepositoryStub) GetByID(ctx context.Context, id string) (*port.QuotationRecord, error) {
	row, exists := s.rows[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	copy := row
	return &copy, nil
}

func (s *dispatchQuotationRepositoryStub) ListByOrderID(ctx context.Context, orderID string) ([]port.QuotationRecord, error) {
	return nil, nil
}

func (s *dispatchQuotationRepositoryStub) GetLatestByOrderAndCarrier(ctx context.Context, orderID string, carrierID string) (*port.QuotationRecord, error) {
	return nil, nil
}

func (s *dispatchQuotationRepositoryStub) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

type dispatchOrderQuotationSourceStub struct {
	row *port.OrderQuotationData
}

func (s dispatchOrderQuotationSourceStub) GetByIDOrIdentifier(ctx context.Context, identifier string) (*port.OrderQuotationData, error) {
	return s.row, nil
}

// TestCreateBatch verifies batch creation publishes the batch-created event.
func TestCreateBatch(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	publisher := &dispatchPublisherStub{}
	service := NewService(batchRepository, markRepository, publisher)

	batch, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if batch == nil || batch.ID == "" {
		t.Fatalf("invalid batch = %#v", batch)
	}
	if len(publisher.events) == 0 || publisher.events[0].Topic != port.TopicBatchCreated {
		t.Fatalf("missing batch created event")
	}
}

// TestCreateBatchRejectsSecondOpenBatchForSameCarrier verifies one carrier cannot keep multiple open batches.
func TestCreateBatchRejectsSecondOpenBatchForSameCarrier(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	service := NewService(batchRepository, markRepository, &dispatchPublisherStub{})

	if _, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-1"}); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	if _, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-2"}); !errors.Is(err, domain.ErrBatchOpenForCarrier) {
		t.Fatalf("Create(second) error = %v, want ErrBatchOpenForCarrier", err)
	}
	if _, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "tcc", CreatedBy: "user-3"}); err != nil {
		t.Fatalf("Create(other carrier) error = %v", err)
	}
}

// TestDraftMarkAndClose verifies draft mark creation and batch close materialize flow.
func TestDraftMarkAndClose(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	publisher := &dispatchPublisherStub{}
	materializer := &materializerStub{repository: markRepository}
	service := NewService(batchRepository, markRepository, publisher, materializer)

	batch, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	mark, err := service.DraftMark(context.Background(), DraftMarkCommand{
		BatchID:           batch.ID,
		QuotationID:       "quote-1",
		QuotedFreightCost: 15000,
		OrderID:           "order-1",
		Sender:            domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"},
		Recipient:         domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"},
		Units:             []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode:      domain.ShipmentModeParcel,
	})
	if err != nil {
		t.Fatalf("DraftMark() error = %v", err)
	}
	if mark.Status != domain.MarkStatusQuoted {
		t.Fatalf("draft mark status = %q", mark.Status)
	}
	if mark.QuotedFreightCost != 15000 {
		t.Fatalf("quoted freight cost = %v", mark.QuotedFreightCost)
	}

	closed, err := service.Close(context.Background(), batch.ID)
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if closed.Status != domain.BatchStatusClosed {
		t.Fatalf("closed status = %q", closed.Status)
	}
	if materializer.calls != 1 {
		t.Fatalf("materializer calls = %d", materializer.calls)
	}
	persisted, _ := markRepository.GetByID(context.Background(), mark.ID)
	if persisted.Status != domain.MarkStatusCreated {
		t.Fatalf("mark status after close = %q", persisted.Status)
	}
	if len(publisher.events) < 2 || publisher.events[len(publisher.events)-1].Topic != port.TopicBatchClosed {
		t.Fatalf("missing batch closed event")
	}
}

// TestCloseReturnsGuardrailViolation verifies batch close stops and returns guardrail violations from mark materialization.
func TestCloseReturnsGuardrailViolation(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	guardrailErr := &domain.GuardrailViolationError{
		CarrierID:      "tcc",
		MarkID:         "mark-guardrail",
		OrderID:        "order-guardrail",
		Rule:           "tcc_non_cod_formapago_must_be_1",
		RequestPreview: "{\"formapago\":\"2\"}",
	}
	materializer := &materializerErrorStub{err: guardrailErr}
	service := NewService(batchRepository, markRepository, nil, materializer)

	batch, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "tcc", CreatedBy: "user-123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	_, err = service.DraftMark(context.Background(), DraftMarkCommand{
		BatchID:      batch.ID,
		OrderID:      "order-guardrail",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	})
	if err != nil {
		t.Fatalf("DraftMark() error = %v", err)
	}

	closed, err := service.Close(context.Background(), batch.ID)
	if err == nil {
		t.Fatalf("Close() error = nil, want guardrail violation")
	}
	if closed != nil {
		t.Fatalf("Close() batch = %#v, want nil on guardrail error", closed)
	}
	if err != guardrailErr {
		t.Fatalf("Close() error = %v, want same guardrail error", err)
	}
	if materializer.calls != 1 {
		t.Fatalf("materializer calls = %d, want 1", materializer.calls)
	}
	reloaded, getErr := batchRepository.GetByID(context.Background(), batch.ID)
	if getErr != nil {
		t.Fatalf("GetByID() error = %v", getErr)
	}
	if reloaded.Status != domain.BatchStatusOpen {
		t.Fatalf("batch status = %q, want OPEN", reloaded.Status)
	}
}

// TestCreateBatchMarkDraft verifies CreateBatchMark delegates to draft behavior when direct is disabled.
func TestCreateBatchMarkDraft(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	service := NewService(batchRepository, markRepository, nil)

	batch, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	mark, err := service.CreateBatchMark(context.Background(), CreateBatchMarkCommand{
		BatchID:      batch.ID,
		Direct:       false,
		OrderID:      "order-1",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	})
	if err != nil {
		t.Fatalf("CreateBatchMark() error = %v", err)
	}
	if mark.Status != domain.MarkStatusQuoted {
		t.Fatalf("status = %q", mark.Status)
	}
	if mark.CollectOnDeliveryAmount != 0 {
		t.Fatalf("collectOnDeliveryAmount = %v, want 0", mark.CollectOnDeliveryAmount)
	}
	if mark.DispatchBatchID == nil || *mark.DispatchBatchID != batch.ID {
		t.Fatalf("dispatch batch id = %v", mark.DispatchBatchID)
	}
}

// TestDraftMarkEnrichesFromOrderData verifies draft-mark creation can enrich sender/recipient/COD defaults from order data.
func TestDraftMarkEnrichesFromOrderData(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	service := NewService(batchRepository, markRepository, nil)
	service.SetDefaultSender(domain.Address{
		Name:        "Flock",
		ID:          "901599500",
		IDType:      "NIT",
		AddressLine: "Calle 18 Sur 24D 46 P2",
		CityCode:    "11001000",
		Phone:       "3057901484",
		Email:       "coccostoreco@gmail.com",
	})
	service.SetOrderSource(dispatchOrderQuotationSourceStub{row: &port.OrderQuotationData{
		OrderID:                 "order-internal-1",
		DestCityCode:            "76001000",
		TotalValue:              162000,
		CollectOnDeliveryAmount: 162000,
		RecipientName:           "Marylu",
		RecipientID:             "83395cf06d6837104f19a7c9a99a2517",
		RecipientIDType:         "CC",
		RecipientAddressLine:    "Recipient street 456",
		RecipientPhone:          "3110000000",
		RecipientEmail:          "coccostoreco@gmail.com",
	}})

	batch, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	customTrackingURL := "https://rastreo.flockstore.co/guide/manual-001"
	mark, err := service.DraftMark(context.Background(), DraftMarkCommand{
		BatchID:           batch.ID,
		OrderID:           "1024554",
		Sender:            domain.Address{},
		Recipient:         domain.Address{},
		Units:             []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode:      domain.ShipmentModeParcel,
		TrackingNumber:    "MANUAL-001",
		CustomTrackingURL: &customTrackingURL,
	})
	if err != nil {
		t.Fatalf("DraftMark() error = %v", err)
	}
	if mark.OrderID != "order-internal-1" {
		t.Fatalf("mark.OrderID = %q, want %q", mark.OrderID, "order-internal-1")
	}
	if mark.Sender.Name != "Flock" {
		t.Fatalf("mark.Sender.Name = %q, want %q", mark.Sender.Name, "Flock")
	}
	if mark.Recipient.Name != "Marylu" {
		t.Fatalf("mark.Recipient.Name = %q, want %q", mark.Recipient.Name, "Marylu")
	}
	if mark.Recipient.CityCode != "76001000" {
		t.Fatalf("mark.Recipient.CityCode = %q, want %q", mark.Recipient.CityCode, "76001000")
	}
	if mark.CollectOnDeliveryAmount != 162000 {
		t.Fatalf("mark.CollectOnDeliveryAmount = %v, want %v", mark.CollectOnDeliveryAmount, 162000)
	}
	if mark.DeclaredValue != 162000 {
		t.Fatalf("mark.DeclaredValue = %v, want %v", mark.DeclaredValue, 162000)
	}
	if mark.TrackingNumber != "MANUAL-001" {
		t.Fatalf("mark.TrackingNumber = %q, want %q", mark.TrackingNumber, "MANUAL-001")
	}
	if mark.CustomTrackingURL == nil || *mark.CustomTrackingURL != customTrackingURL {
		t.Fatalf("mark.CustomTrackingURL = %v, want %q", mark.CustomTrackingURL, customTrackingURL)
	}
}

// TestDraftMarkManualDefaultsAllowSparsePayload verifies manual draft marks can be created without quotation-derived units or shipment mode.
func TestDraftMarkManualDefaultsAllowSparsePayload(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	service := NewService(batchRepository, markRepository, nil)

	batch, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	mark, err := service.DraftMark(context.Background(), DraftMarkCommand{
		BatchID:        batch.ID,
		OrderID:        "order-manual-1",
		TrackingNumber: "MANUAL-TRACK-001",
	})
	if err != nil {
		t.Fatalf("DraftMark() error = %v", err)
	}
	if len(mark.Units) != 1 {
		t.Fatalf("len(mark.Units) = %d, want 1", len(mark.Units))
	}
	if mark.Units[0].PackageType != "CAJA" {
		t.Fatalf("mark.Units[0].PackageType = %q, want %q", mark.Units[0].PackageType, "CAJA")
	}
	if mark.ShipmentMode != domain.ShipmentModeExpress {
		t.Fatalf("mark.ShipmentMode = %q, want %q", mark.ShipmentMode, domain.ShipmentModeExpress)
	}
	if mark.TrackingNumber != "MANUAL-TRACK-001" {
		t.Fatalf("mark.TrackingNumber = %q, want %q", mark.TrackingNumber, "MANUAL-TRACK-001")
	}
}

// TestCreateBatchMarkDirectAllowsClosedBatch verifies direct creation materializes marks even when the batch is closed.
func TestCreateBatchMarkDirectAllowsClosedBatch(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	materializer := &materializerStub{repository: markRepository}
	service := NewService(batchRepository, markRepository, nil, materializer)

	batch, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := batchRepository.Close(context.Background(), batch.ID); err != nil {
		t.Fatalf("Close() stub error = %v", err)
	}

	mark, err := service.CreateBatchMark(context.Background(), CreateBatchMarkCommand{
		BatchID:      batch.ID,
		Direct:       true,
		OrderID:      "order-2",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	})
	if err != nil {
		t.Fatalf("CreateBatchMark() error = %v", err)
	}
	if materializer.calls != 1 {
		t.Fatalf("materializer calls = %d", materializer.calls)
	}
	if mark.Status != domain.MarkStatusCreated {
		t.Fatalf("status = %q", mark.Status)
	}
	if mark.CollectOnDeliveryAmount != 0 {
		t.Fatalf("collectOnDeliveryAmount = %v, want 0", mark.CollectOnDeliveryAmount)
	}
	if mark.DispatchBatchID == nil || *mark.DispatchBatchID != batch.ID {
		t.Fatalf("dispatch batch id = %v", mark.DispatchBatchID)
	}
}

// TestCreateBatchMarkDirectManualDefaultsAllowSparsePayload verifies direct manual marks can materialize with sparse operator input.
func TestCreateBatchMarkDirectManualDefaultsAllowSparsePayload(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	materializer := &materializerStub{repository: markRepository}
	service := NewService(batchRepository, markRepository, nil, materializer)

	batch, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	customTrackingURL := "https://rastreo.flockstore.co/manual-track-2"
	mark, err := service.CreateBatchMark(context.Background(), CreateBatchMarkCommand{
		BatchID:           batch.ID,
		Direct:            true,
		OrderID:           "order-manual-2",
		TrackingNumber:    "MANUAL-TRACK-002",
		Observations:      "Servientrega",
		CustomTrackingURL: &customTrackingURL,
	})
	if err != nil {
		t.Fatalf("CreateBatchMark() error = %v", err)
	}
	if materializer.calls != 1 {
		t.Fatalf("materializer calls = %d, want 1", materializer.calls)
	}
	if mark.Status != domain.MarkStatusCreated {
		t.Fatalf("mark.Status = %q, want %q", mark.Status, domain.MarkStatusCreated)
	}
	if len(mark.Units) != 1 {
		t.Fatalf("len(mark.Units) = %d, want 1", len(mark.Units))
	}
	if mark.ShipmentMode != domain.ShipmentModeExpress {
		t.Fatalf("mark.ShipmentMode = %q, want %q", mark.ShipmentMode, domain.ShipmentModeExpress)
	}
	if mark.CustomTrackingURL == nil || *mark.CustomTrackingURL != customTrackingURL {
		t.Fatalf("mark.CustomTrackingURL = %v, want %q", mark.CustomTrackingURL, customTrackingURL)
	}
}

// TestCreateBatchMarkRequiresBatchID verifies batch id is mandatory for both draft and direct flows.
func TestCreateBatchMarkRequiresBatchID(t *testing.T) {
	service := NewService(newDispatchBatchRepositoryStub(), newDispatchMarkRepositoryStub(), nil)

	_, err := service.CreateBatchMark(context.Background(), CreateBatchMarkCommand{
		Direct:       false,
		OrderID:      "order-1",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("CreateBatchMark() error = %v", err)
	}
}

// TestCreateBatchMarkFromQuotation verifies quotation-seeded batch mark creation uses quotation snapshot/order enrichment values.
func TestCreateBatchMarkFromQuotation(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	quotationRepository := newDispatchQuotationRepositoryStub()
	service := NewService(batchRepository, markRepository, nil)
	service.SetQuotationRepository(quotationRepository)
	service.SetDefaultSender(domain.Address{
		Name:        "(FALKON)-GRUPO COCCO",
		ID:          "901599500",
		IDType:      "NIT",
		AddressLine: "Calle 18 Sur 24D 46 P2",
		CityCode:    "11001",
		Phone:       "3057901484",
		Email:       "coccostoreco@gmail.com",
	})
	service.SetOrderSource(dispatchOrderQuotationSourceStub{row: &port.OrderQuotationData{
		OrderID:                 "order-1",
		DestCityCode:            "76001000",
		CollectOnDeliveryAmount: 0,
		RecipientName:           "Cliente",
		RecipientAddressLine:    "Calle 1 # 2-3",
		RecipientPhone:          "3001234567",
	}})

	batch, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	snapshot, marshalErr := json.Marshal(domain.QuotationRequest{
		OrderID:                 "order-1",
		CarrierID:               "manual",
		OriginCityCode:          "11001000",
		DestCityCode:            "76001000",
		Units:                   []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		DeclaredValue:           50000,
		CollectOnDeliveryAmount: 0,
		ShipmentMode:            domain.ShipmentModeExpress,
	})
	if marshalErr != nil {
		t.Fatalf("marshal snapshot: %v", marshalErr)
	}
	quotationRepository.rows["quote-1"] = port.QuotationRecord{
		ID:              "quote-1",
		OrderID:         "order-1",
		OrderIdentifier: "1024554",
		CarrierID:       "manual",
		OriginCityCode:  "11001000",
		DestCityCode:    "76001000",
		FreightCost:     15000,
		Units:           []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		RequestSnapshot: base64.StdEncoding.EncodeToString(snapshot),
		CreatedAt:       time.Now().UTC(),
	}

	mark, err := service.CreateBatchMarkFromQuotation(context.Background(), CreateBatchMarkFromQuotationCommand{
		BatchID:     batch.ID,
		QuotationID: "quote-1",
		Direct:      false,
	})
	if err != nil {
		t.Fatalf("CreateBatchMarkFromQuotation() error = %v", err)
	}
	if mark == nil || mark.QuotationID == nil || *mark.QuotationID != "quote-1" {
		t.Fatalf("mark quotation id = %#v", mark)
	}
	if mark.CollectOnDeliveryAmount != 0 {
		t.Fatalf("mark.CollectOnDeliveryAmount = %v, want 0", mark.CollectOnDeliveryAmount)
	}
	if mark.Recipient.CityCode != "76001000" {
		t.Fatalf("mark recipient city = %q, want 76001000", mark.Recipient.CityCode)
	}
	if mark.Sender.Name != "(FALKON)-GRUPO COCCO" {
		t.Fatalf("mark sender name = %q, want configured default sender name", mark.Sender.Name)
	}
	if mark.Sender.CityCode != "11001" {
		t.Fatalf("mark sender city = %q, want 11001", mark.Sender.CityCode)
	}

	duplicate, err := service.CreateBatchMarkFromQuotation(context.Background(), CreateBatchMarkFromQuotationCommand{
		BatchID:     batch.ID,
		QuotationID: "quote-1",
		Direct:      false,
	})
	if err != nil {
		t.Fatalf("CreateBatchMarkFromQuotation(duplicate) error = %v", err)
	}
	if duplicate == nil || duplicate.ID != mark.ID {
		t.Fatalf("duplicate mark id = %v, want existing mark id %q", duplicate, mark.ID)
	}
	if len(markRepository.marks) != 1 {
		t.Fatalf("mark repository size = %d, want 1", len(markRepository.marks))
	}
}

// TestCreateBatchMarkDirectOnExistingQuotedMaterializes verifies direct mode materializes the existing quoted mark instead of creating duplicates.
func TestCreateBatchMarkDirectOnExistingQuotedMaterializes(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	quotationRepository := newDispatchQuotationRepositoryStub()
	materializer := &materializerStub{repository: markRepository}
	service := NewService(batchRepository, markRepository, nil, materializer)
	service.SetQuotationRepository(quotationRepository)
	service.SetDefaultSender(domain.Address{
		Name:        "(FALKON)-GRUPO COCCO",
		ID:          "901599500",
		IDType:      "NIT",
		AddressLine: "Calle 18 Sur 24D 46 P2",
		CityCode:    "11001",
		Phone:       "3057901484",
		Email:       "coccostoreco@gmail.com",
	})
	service.SetOrderSource(dispatchOrderQuotationSourceStub{row: &port.OrderQuotationData{
		OrderID:                 "order-2",
		DestCityCode:            "76001000",
		CollectOnDeliveryAmount: 0,
		RecipientName:           "Cliente",
	}})

	batch, err := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	snapshot, marshalErr := json.Marshal(domain.QuotationRequest{
		OrderID:                 "order-2",
		CarrierID:               "manual",
		OriginCityCode:          "11001000",
		DestCityCode:            "76001000",
		Units:                   []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		DeclaredValue:           50000,
		CollectOnDeliveryAmount: 0,
		ShipmentMode:            domain.ShipmentModeExpress,
	})
	if marshalErr != nil {
		t.Fatalf("marshal snapshot: %v", marshalErr)
	}
	quotationRepository.rows["quote-2"] = port.QuotationRecord{
		ID:              "quote-2",
		OrderID:         "order-2",
		OrderIdentifier: "1024555",
		CarrierID:       "manual",
		OriginCityCode:  "11001000",
		DestCityCode:    "76001000",
		FreightCost:     15000,
		Units:           []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		RequestSnapshot: base64.StdEncoding.EncodeToString(snapshot),
		CreatedAt:       time.Now().UTC(),
	}

	quotedMark, err := service.CreateBatchMarkFromQuotation(context.Background(), CreateBatchMarkFromQuotationCommand{
		BatchID:     batch.ID,
		QuotationID: "quote-2",
		Direct:      false,
	})
	if err != nil {
		t.Fatalf("CreateBatchMarkFromQuotation(quoted) error = %v", err)
	}
	if quotedMark.Status != domain.MarkStatusQuoted {
		t.Fatalf("quoted mark status = %q, want QUOTED", quotedMark.Status)
	}

	materializedMark, err := service.CreateBatchMarkFromQuotation(context.Background(), CreateBatchMarkFromQuotationCommand{
		BatchID:     batch.ID,
		QuotationID: "quote-2",
		Direct:      true,
	})
	if err != nil {
		t.Fatalf("CreateBatchMarkFromQuotation(direct) error = %v", err)
	}
	if materializedMark.ID != quotedMark.ID {
		t.Fatalf("materialized mark id = %q, want existing mark id %q", materializedMark.ID, quotedMark.ID)
	}
	if materializedMark.Status != domain.MarkStatusCreated {
		t.Fatalf("materialized mark status = %q, want CREATED", materializedMark.Status)
	}
	if materializer.calls != 1 {
		t.Fatalf("materializer calls = %d, want 1", materializer.calls)
	}
	if len(markRepository.marks) != 1 {
		t.Fatalf("mark repository size = %d, want 1", len(markRepository.marks))
	}
}

// TestRemoveDraftMark verifies that a QUOTED draft mark is permanently deleted from the store.
func TestRemoveDraftMark(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	publisher := &dispatchPublisherStub{}
	service := NewService(batchRepository, markRepository, publisher)

	batch, _ := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	mark, _ := service.DraftMark(context.Background(), DraftMarkCommand{
		BatchID:      batch.ID,
		OrderID:      "order-1",
		Sender:       domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"},
		Recipient:    domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"},
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		ShipmentMode: domain.ShipmentModeParcel,
	})

	updated, err := service.RemoveDraftMark(context.Background(), batch.ID, mark.ID)
	if err != nil {
		t.Fatalf("RemoveDraftMark() error = %v", err)
	}
	if len(updated.MarkIDs) != 0 {
		t.Fatalf("batch mark ids after remove = %v", updated.MarkIDs)
	}
	_, err = markRepository.GetByID(context.Background(), mark.ID)
	if err == nil {
		t.Fatal("expected mark to be permanently deleted but GetByID returned no error")
	}
}

// TestRemoveDraftMarkRejectsNonDraft verifies that only QUOTED marks can be removed.
func TestRemoveDraftMarkRejectsNonDraft(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	service := NewService(batchRepository, markRepository, nil)

	batch, _ := service.Create(context.Background(), CreateBatchCommand{CarrierID: "manual", CreatedBy: "user-123"})
	batchID := batch.ID
	markRepository.marks["mark-created"] = domain.ShippingMark{
		ID:              "mark-created",
		Status:          domain.MarkStatusCreated,
		CarrierID:       "manual",
		DispatchBatchID: &batchID,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	batchRepository.batches[batch.ID] = domain.DispatchBatch{
		ID:        batch.ID,
		CarrierID: "manual",
		Status:    domain.BatchStatusOpen,
		MarkIDs:   []string{"mark-created"},
	}

	_, err := service.RemoveDraftMark(context.Background(), batch.ID, "mark-created")
	if err == nil {
		t.Fatal("RemoveDraftMark() expected error for non-QUOTED mark")
	}
}
