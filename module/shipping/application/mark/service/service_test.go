package service

import (
	"context"
	"testing"

	markevent "mannaiah/module/shipping/application/mark/event"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

type markRepositoryStub struct {
	rows map[string]domain.ShippingMark
}

func newMarkRepositoryStub() *markRepositoryStub {
	return &markRepositoryStub{rows: map[string]domain.ShippingMark{}}
}

func (s *markRepositoryStub) Create(ctx context.Context, mark *domain.ShippingMark) error {
	s.rows[mark.ID] = *mark

	return nil
}
func (s *markRepositoryStub) GetByID(ctx context.Context, id string) (*domain.ShippingMark, error) {
	row, exists := s.rows[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	copy := row

	return &copy, nil
}
func (s *markRepositoryStub) GetByTrackingNumber(ctx context.Context, trackingNumber string) (*domain.ShippingMark, error) {
	for _, row := range s.rows {
		if row.TrackingNumber == trackingNumber {
			copy := row

			return &copy, nil
		}
	}

	return nil, domain.ErrNotFound
}
func (s *markRepositoryStub) ListByOrderID(ctx context.Context, orderID string) ([]domain.ShippingMark, error) {
	return nil, nil
}
func (s *markRepositoryStub) ListByBatchID(ctx context.Context, batchID string) ([]domain.ShippingMark, error) {
	return nil, nil
}
func (s *markRepositoryStub) Update(ctx context.Context, mark *domain.ShippingMark) error {
	s.rows[mark.ID] = *mark

	return nil
}
func (s *markRepositoryStub) List(ctx context.Context, query port.MarkListQuery) ([]domain.ShippingMark, int64, error) {
	rows := make([]domain.ShippingMark, 0, len(s.rows))
	for _, row := range s.rows {
		rows = append(rows, row)
	}

	return rows, int64(len(rows)), nil
}

type markProviderStub struct{}

func (markProviderStub) CarrierID() string { return "manual" }
func (markProviderStub) Carrier() domain.Carrier {
	return domain.Carrier{ID: "manual", Name: "Manual", Type: domain.CarrierTypeManual, Active: true}
}
func (markProviderStub) Quote(ctx context.Context, request domain.QuotationRequest) (*domain.QuotationResult, error) {
	return nil, domain.ErrQuotationNotSupported
}
func (markProviderStub) GenerateMark(ctx context.Context, mark *domain.ShippingMark) error {
	mark.Status = domain.MarkStatusGenerated
	if mark.TrackingNumber == "" {
		mark.TrackingNumber = "MANUAL-TRACK"
	}

	return nil
}
func (markProviderStub) VoidMark(ctx context.Context, trackingNumber string) error { return nil }
func (markProviderStub) CheckBalance(ctx context.Context) error                    { return nil }
func (markProviderStub) SupportsQuotation() bool                                   { return false }

type markRegistryStub struct {
	provider port.CarrierProvider
}

func (s markRegistryStub) CarrierProvider(carrierID string) (port.CarrierProvider, bool) {
	return s.provider, s.provider != nil
}
func (markRegistryStub) TrackingProvider(carrierID string) (port.TrackingProvider, bool) {
	return nil, false
}
func (markRegistryStub) Carriers() []domain.Carrier { return nil }

type publisherStub struct {
	events []port.IntegrationEvent
}

func (s *publisherStub) Publish(ctx context.Context, event port.IntegrationEvent) error {
	s.events = append(s.events, event)

	return nil
}

// TestGenerateAndVoid verifies mark generation and void flows.
func TestGenerateAndVoid(t *testing.T) {
	repository := newMarkRepositoryStub()
	publisher := &publisherStub{}
	service := NewService(repository, markRegistryStub{provider: markProviderStub{}}, publisher)

	created, err := service.Generate(context.Background(), GenerateCommand{
		OrderID:                 "order-1",
		CarrierID:               "manual",
		Sender:                  domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"},
		Recipient:               domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"},
		Units:                   []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		CollectOnDeliveryAmount: 100000,
		ShipmentMode:            domain.ShipmentModeParcel,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if created == nil || created.Status != domain.MarkStatusGenerated {
		t.Fatalf("created status = %#v", created)
	}
	if created.CollectOnDeliveryAmount != 100000 || created.CollectOnDeliveryChargedAmount != 100000 {
		t.Fatalf("created COD values = %#v", created)
	}
	if len(publisher.events) == 0 || publisher.events[0].Topic != port.TopicMarkGenerated {
		t.Fatalf("unexpected generated event = %#v", publisher.events)
	}

	voided, err := service.Void(context.Background(), created.ID, "cancel")
	if err != nil {
		t.Fatalf("Void() error = %v", err)
	}
	if voided.Status != domain.MarkStatusVoided {
		t.Fatalf("voided status = %q", voided.Status)
	}
	if len(publisher.events) < 2 || publisher.events[1].Topic != port.TopicMarkVoided {
		t.Fatalf("unexpected void event topic")
	}
	if _, ok := publisher.events[0].Payload.(markevent.MarkGeneratedPayload); !ok {
		t.Fatalf("generated payload type mismatch")
	}
}

// TestVoidNoRegistry verifies that Void does not require a carrier registry.
func TestVoidNoRegistry(t *testing.T) {
	repository := newMarkRepositoryStub()
	publisher := &publisherStub{}
	service := NewService(repository, markRegistryStub{}, publisher)

	repository.rows["mark-1"] = domain.ShippingMark{
		ID:        "mark-1",
		OrderID:   "order-1",
		CarrierID: "manual",
		Status:    domain.MarkStatusGenerated,
	}

	voided, err := service.Void(context.Background(), "mark-1", "cancel")
	if err != nil {
		t.Fatalf("Void() error = %v", err)
	}
	if voided.Status != domain.MarkStatusVoided {
		t.Fatalf("voided status = %q", voided.Status)
	}
	if len(publisher.events) == 0 || publisher.events[0].Topic != port.TopicMarkVoided {
		t.Fatalf("missing void event")
	}
}

// TestMaterialize verifies that Materialize captures a snapshot, calls the provider, updates status to CREATED, and publishes the generated event.
func TestMaterialize(t *testing.T) {
	repository := newMarkRepositoryStub()
	publisher := &publisherStub{}
	service := NewService(repository, markRegistryStub{provider: markProviderStub{}}, publisher)

	mark := domain.ShippingMark{
		ID:        "mark-draft-1",
		OrderID:   "order-1",
		CarrierID: "manual",
		Status:    domain.MarkStatusQuoted,
		Sender:    domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"},
		Recipient: domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"},
		Units:     []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
	}
	repository.rows[mark.ID] = mark

	if err := service.Materialize(context.Background(), &mark); err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}
	if mark.Status != domain.MarkStatusCreated {
		t.Fatalf("mark status after materialize = %q", mark.Status)
	}
	if mark.DraftSnapshot == "" {
		t.Fatal("DraftSnapshot not set")
	}
	if mark.TrackingNumber == "" {
		t.Fatal("TrackingNumber not set by provider")
	}
	persisted, _ := repository.GetByID(context.Background(), mark.ID)
	if persisted.Status != domain.MarkStatusCreated {
		t.Fatalf("persisted status = %q", persisted.Status)
	}
	if len(publisher.events) == 0 || publisher.events[0].Topic != port.TopicMarkGenerated {
		t.Fatalf("missing mark generated event")
	}
}
