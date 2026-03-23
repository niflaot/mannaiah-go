package service

import (
	"context"
	"testing"

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

func (s *dispatchMarkRepositoryStub) Create(ctx context.Context, mark *domain.ShippingMark) error {
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
	return nil, nil
}
func (s *dispatchMarkRepositoryStub) Update(ctx context.Context, mark *domain.ShippingMark) error {
	return nil
}
func (s *dispatchMarkRepositoryStub) List(ctx context.Context, query port.MarkListQuery) ([]domain.ShippingMark, int64, error) {
	return nil, 0, nil
}

type dispatchPublisherStub struct {
	events []port.IntegrationEvent
}

func (s *dispatchPublisherStub) Publish(ctx context.Context, event port.IntegrationEvent) error {
	s.events = append(s.events, event)

	return nil
}

// TestCreateAddClose verifies batch creation, mark assignment, and batch close behaviors.
func TestCreateAddClose(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := &dispatchMarkRepositoryStub{marks: map[string]domain.ShippingMark{
		"mark-1": {ID: "mark-1", CarrierID: "manual", Status: domain.MarkStatusGenerated},
	}}
	publisher := &dispatchPublisherStub{}
	service := NewService(batchRepository, markRepository, publisher)

	batch, err := service.Create(context.Background(), CreateBatchCommand{Name: "Batch A", CarrierID: "manual"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if batch == nil || batch.ID == "" {
		t.Fatalf("invalid batch = %#v", batch)
	}
	if len(publisher.events) == 0 || publisher.events[0].Topic != port.TopicBatchCreated {
		t.Fatalf("missing batch created event")
	}

	updated, err := service.AddMarks(context.Background(), AddMarksCommand{BatchID: batch.ID, MarkIDs: []string{"mark-1"}})
	if err != nil {
		t.Fatalf("AddMarks() error = %v", err)
	}
	if len(updated.MarkIDs) != 1 {
		t.Fatalf("updated mark ids = %#v", updated.MarkIDs)
	}

	closed, err := service.Close(context.Background(), batch.ID)
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if closed.Status != domain.BatchStatusClosed {
		t.Fatalf("closed status = %q", closed.Status)
	}
	if len(publisher.events) < 2 || publisher.events[len(publisher.events)-1].Topic != port.TopicBatchClosed {
		t.Fatalf("missing batch closed event")
	}
}
