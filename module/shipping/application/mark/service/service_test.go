package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
	var result []domain.ShippingMark
	for _, row := range s.rows {
		if row.OrderID == orderID {
			result = append(result, row)
		}
	}

	return result, nil
}
func (s *markRepositoryStub) ListByBatchID(ctx context.Context, batchID string) ([]domain.ShippingMark, error) {
	var result []domain.ShippingMark
	for _, row := range s.rows {
		if row.DispatchBatchID != nil && strings.TrimSpace(*row.DispatchBatchID) == strings.TrimSpace(batchID) {
			result = append(result, row)
		}
	}

	return result, nil
}
func (s *markRepositoryStub) Update(ctx context.Context, mark *domain.ShippingMark) error {
	s.rows[mark.ID] = *mark

	return nil
}
func (s *markRepositoryStub) Delete(ctx context.Context, id string) error {
	delete(s.rows, id)

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
	if created.DraftSnapshot == "" {
		t.Fatal("created.DraftSnapshot not set")
	}
	if created.ResponseSnapshot == "" {
		t.Fatal("created.ResponseSnapshot not set")
	}
	if _, decodeErr := base64.StdEncoding.DecodeString(created.DraftSnapshot); decodeErr != nil {
		t.Fatalf("created.DraftSnapshot should be base64: %v", decodeErr)
	}
	if _, decodeErr := base64.StdEncoding.DecodeString(created.ResponseSnapshot); decodeErr != nil {
		t.Fatalf("created.ResponseSnapshot should be base64: %v", decodeErr)
	}
	if created.CollectOnDeliveryAmount != 100000 || created.CollectOnDeliveryChargedAmount != 100000 {
		t.Fatalf("created COD values = %#v", created)
	}
	if created.FailureReason != "" {
		t.Fatalf("created.FailureReason = %q, want empty", created.FailureReason)
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

// TestQueryDispatch verifies dispatch provisioning status resolution by order.
func TestQueryDispatch(t *testing.T) {
	repository := newMarkRepositoryStub()
	publisher := &publisherStub{}
	service := NewService(repository, markRegistryStub{}, publisher)

	result, err := service.QueryDispatch(context.Background(), DispatchQuery{OrderID: "order-1"})
	if err != nil {
		t.Fatalf("QueryDispatch() error = %v", err)
	}
	if result.Provisioned {
		t.Fatalf("expected not provisioned for unknown order")
	}

	repository.rows["mark-gen"] = domain.ShippingMark{ID: "mark-gen", OrderID: "order-1", Status: domain.MarkStatusGenerated}
	repository.rows["mark-quoted"] = domain.ShippingMark{ID: "mark-quoted", OrderID: "order-1", Status: domain.MarkStatusQuoted}
	repository.rows["mark-voided"] = domain.ShippingMark{ID: "mark-voided", OrderID: "order-1", Status: domain.MarkStatusVoided}

	result, err = service.QueryDispatch(context.Background(), DispatchQuery{OrderID: "order-1"})
	if err != nil {
		t.Fatalf("QueryDispatch() error = %v", err)
	}
	if !result.Provisioned {
		t.Fatalf("expected provisioned")
	}
	if result.MarkID != "mark-quoted" {
		t.Fatalf("expected QUOTED mark to win; got %q", result.MarkID)
	}
	if result.Status != domain.MarkStatusQuoted {
		t.Fatalf("expected status QUOTED; got %q", result.Status)
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
	if mark.ResponseSnapshot == "" {
		t.Fatal("ResponseSnapshot not set")
	}
	decodedDraftSnapshot, decodeErr := base64.StdEncoding.DecodeString(mark.DraftSnapshot)
	if decodeErr != nil {
		t.Fatalf("DraftSnapshot should be base64: %v", decodeErr)
	}
	decodedResponseSnapshot, responseDecodeErr := base64.StdEncoding.DecodeString(mark.ResponseSnapshot)
	if responseDecodeErr != nil {
		t.Fatalf("ResponseSnapshot should be base64: %v", responseDecodeErr)
	}
	var draftPayload map[string]any
	if unmarshalErr := json.Unmarshal(decodedDraftSnapshot, &draftPayload); unmarshalErr != nil {
		t.Fatalf("decode DraftSnapshot json: %v", unmarshalErr)
	}
	var responsePayload map[string]any
	if unmarshalErr := json.Unmarshal(decodedResponseSnapshot, &responsePayload); unmarshalErr != nil {
		t.Fatalf("decode ResponseSnapshot json: %v", unmarshalErr)
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

// TestRelated verifies related-mark resolution by shared order and batch values.
func TestRelated(t *testing.T) {
	repository := newMarkRepositoryStub()
	service := NewService(repository, markRegistryStub{}, &publisherStub{})

	batchID := "batch-1"
	now := time.Now().UTC()
	repository.rows["mark-target"] = domain.ShippingMark{
		ID:              "mark-target",
		OrderID:         "order-1",
		CarrierID:       "manual",
		Status:          domain.MarkStatusCreated,
		DispatchBatchID: &batchID,
		CreatedAt:       now,
	}
	repository.rows["mark-order"] = domain.ShippingMark{
		ID:        "mark-order",
		OrderID:   "order-1",
		CarrierID: "manual",
		Status:    domain.MarkStatusCreated,
		CreatedAt: now.Add(-1 * time.Minute),
	}
	repository.rows["mark-batch"] = domain.ShippingMark{
		ID:              "mark-batch",
		OrderID:         "order-2",
		CarrierID:       "manual",
		Status:          domain.MarkStatusCreated,
		DispatchBatchID: &batchID,
		CreatedAt:       now.Add(-2 * time.Minute),
	}
	repository.rows["mark-other"] = domain.ShippingMark{
		ID:        "mark-other",
		OrderID:   "order-3",
		CarrierID: "manual",
		Status:    domain.MarkStatusCreated,
		CreatedAt: now.Add(-3 * time.Minute),
	}

	rows, err := service.Related(context.Background(), "mark-target")
	if err != nil {
		t.Fatalf("Related() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want %d", len(rows), 2)
	}
	if rows[0].ID != "mark-order" {
		t.Fatalf("rows[0].ID = %q, want %q", rows[0].ID, "mark-order")
	}
	if rows[1].ID != "mark-batch" {
		t.Fatalf("rows[1].ID = %q, want %q", rows[1].ID, "mark-batch")
	}
}
