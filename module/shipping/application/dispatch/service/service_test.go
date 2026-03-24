package service

import (
	"context"
	"testing"
	"time"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

type dispatchBatchRepositoryStub struct {
	batches map[string]domain.DispatchBatch
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
	return nil, nil
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

type dispatchPublisherStub struct {
	events []port.IntegrationEvent
}

func (s *dispatchPublisherStub) Publish(ctx context.Context, event port.IntegrationEvent) error {
	s.events = append(s.events, event)

	return nil
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

// TestDraftMarkAndClose verifies draft mark creation and batch close materialize flow.
func TestDraftMarkAndClose(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
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

// TestRemoveDraftMark verifies that a QUOTED draft mark can be removed from a batch.
func TestRemoveDraftMark(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
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
	persisted, _ := markRepository.GetByID(context.Background(), mark.ID)
	if persisted.Status != domain.MarkStatusRemoved {
		t.Fatalf("mark status after remove = %q", persisted.Status)
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
