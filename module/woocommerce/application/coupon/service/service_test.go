package service

import (
	"context"
	errorspkg "errors"
	"strings"
	"testing"

	"mannaiah/module/woocommerce/application/coupon/event"
	"mannaiah/module/woocommerce/port"

	"go.uber.org/zap"
)

// couponSourceMock defines WooCommerce coupon source behavior for sync tests.
type couponSourceMock struct {
	// validateErr defines validation errors.
	validateErr error
	// pages defines paginated coupon responses.
	pages [][]port.WooCoupon
	// listErrAtPage defines page numbers that should return list errors.
	listErrAtPage map[int]error
}

// Validate verifies source connectivity.
func (m *couponSourceMock) Validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return m.validateErr
}

// ListCoupons retrieves paginated coupon values.
func (m *couponSourceMock) ListCoupons(ctx context.Context, page int, pageSize int) (coupons []port.WooCoupon, hasNext bool, err error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if listErr, hasError := m.listErrAtPage[page]; hasError {
		return nil, false, listErr
	}
	if page <= 0 || page > len(m.pages) {
		return nil, false, nil
	}

	items := m.pages[page-1]
	return items, page < len(m.pages), nil
}

// GetCouponByID retrieves one coupon by identifier.
func (m *couponSourceMock) GetCouponByID(ctx context.Context, id int) (port.WooCoupon, error) {
	if err := ctx.Err(); err != nil {
		return port.WooCoupon{}, err
	}

	for _, page := range m.pages {
		for _, coupon := range page {
			if coupon.ID == id {
				return coupon, nil
			}
		}
	}

	return port.WooCoupon{}, nil
}

// couponTargetMock defines coupon upsert behavior for sync tests.
type couponTargetMock struct {
	// outcomes defines upsert outcomes keyed by WooCommerce coupon identifier.
	outcomes map[int]port.UpsertOutcome
	// errors defines upsert errors keyed by WooCommerce coupon identifier.
	errors map[int]error
	// coupons stores received WooCommerce coupons.
	coupons []port.WooCoupon
}

// UpsertByWooID creates or updates coupons by WooCommerce identifier.
func (m *couponTargetMock) UpsertByWooID(ctx context.Context, coupon port.WooCoupon) (port.UpsertOutcome, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	m.coupons = append(m.coupons, coupon)
	if err := m.errors[coupon.ID]; err != nil {
		return "", err
	}
	if outcome, ok := m.outcomes[coupon.ID]; ok {
		return outcome, nil
	}

	return port.UpsertOutcomeUpdated, nil
}

// couponPublisherMock defines integration event publication behavior for sync tests.
type couponPublisherMock struct {
	// events stores published integration events.
	events []port.IntegrationEvent
}

// Publish captures integration events.
func (m *couponPublisherMock) Publish(ctx context.Context, integrationEvent port.IntegrationEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.events = append(m.events, integrationEvent)
	return nil
}

// couponSyncRecorderMock defines sync recorder behavior for coupon sync tests.
type couponSyncRecorderMock struct {
	// runID defines returned run identifiers.
	runID string
	// startErr defines run-start errors.
	startErr error
	// completeCalled reports completion calls.
	completeCalled bool
	// failCalled reports failure calls.
	failCalled bool
	// processed defines latest processed counters.
	processed int
	// succeeded defines latest succeeded counters.
	succeeded int
	// failed defines latest failed counters.
	failed int
	// skipped defines latest skipped counters.
	skipped int
	// syncErrors stores the latest failure payload.
	syncErrors []port.SyncError
}

// StartRun captures sync start calls.
func (m *couponSyncRecorderMock) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if m.startErr != nil {
		return "", m.startErr
	}
	if strings.TrimSpace(m.runID) == "" {
		return "run-1", nil
	}

	return m.runID, nil
}

// CompleteRun captures sync completion calls.
func (m *couponSyncRecorderMock) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.completeCalled = true
	m.processed = processed
	m.succeeded = succeeded
	m.failed = failed
	m.skipped = skipped
	return nil
}

// FailRun captures sync failure calls.
func (m *couponSyncRecorderMock) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []port.SyncError) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.failCalled = true
	m.processed = processed
	m.succeeded = succeeded
	m.failed = failed
	m.skipped = skipped
	m.syncErrors = append([]port.SyncError(nil), syncErrors...)
	return nil
}

// TestSyncCouponsSuccessCompletesRun verifies successful coupon sync completion behavior.
func TestSyncCouponsSuccessCompletesRun(t *testing.T) {
	source := &couponSourceMock{
		pages: [][]port.WooCoupon{{
			{ID: 101, Code: "WELCOME10"},
			{ID: 102, Code: "SPRING15"},
		}},
	}
	target := &couponTargetMock{
		outcomes: map[int]port.UpsertOutcome{
			101: port.UpsertOutcomeCreated,
			102: port.UpsertOutcomeUnchanged,
		},
		errors: map[int]error{},
	}
	publisher := &couponPublisherMock{}
	recorder := &couponSyncRecorderMock{runID: "run-success"}

	svc, err := NewService(SyncConfig{Enabled: true, PageSize: 100}, source, target, publisher, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	svc.SetSyncRecorder(recorder)

	summary, syncErr := svc.SyncCoupons(context.Background(), "manual")
	if syncErr != nil {
		t.Fatalf("SyncCoupons() error = %v", syncErr)
	}
	if summary.Processed != 2 || summary.Created != 1 || summary.Unchanged != 1 || summary.Failed != 0 {
		t.Fatalf("summary = %+v, want processed=2 created=1 unchanged=1 failed=0", summary)
	}
	if !recorder.completeCalled {
		t.Fatalf("expected CompleteRun to be called")
	}
	if recorder.failCalled {
		t.Fatalf("expected FailRun not to be called")
	}
	if recorder.processed != 2 || recorder.succeeded != 2 || recorder.failed != 0 || recorder.skipped != 0 {
		t.Fatalf("recorder counters = processed=%d succeeded=%d failed=%d skipped=%d, want 2/2/0/0", recorder.processed, recorder.succeeded, recorder.failed, recorder.skipped)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("len(events) = %d, want %d", len(publisher.events), 2)
	}
	if publisher.events[0].Topic != event.TopicCouponsSyncStarted {
		t.Fatalf("events[0].Topic = %q, want %q", publisher.events[0].Topic, event.TopicCouponsSyncStarted)
	}
	if publisher.events[1].Topic != event.TopicCouponsSyncCompleted {
		t.Fatalf("events[1].Topic = %q, want %q", publisher.events[1].Topic, event.TopicCouponsSyncCompleted)
	}
}

// TestSyncCouponsPartialFailureFailsRun verifies failed upserts mark the run as failed and return an error.
func TestSyncCouponsPartialFailureFailsRun(t *testing.T) {
	source := &couponSourceMock{
		pages: [][]port.WooCoupon{{
			{ID: 201, Code: "GOOD10"},
			{ID: 202, Code: "BROKEN10"},
		}},
	}
	target := &couponTargetMock{
		outcomes: map[int]port.UpsertOutcome{
			201: port.UpsertOutcomeCreated,
		},
		errors: map[int]error{
			202: errorspkg.New("persist coupon: unknown column woocommerce_id"),
		},
	}
	publisher := &couponPublisherMock{}
	recorder := &couponSyncRecorderMock{runID: "run-failed"}

	svc, err := NewService(SyncConfig{Enabled: true, PageSize: 100}, source, target, publisher, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	svc.SetSyncRecorder(recorder)

	summary, syncErr := svc.SyncCoupons(context.Background(), "manual")
	if !errorspkg.Is(syncErr, ErrPartialSyncFailure) {
		t.Fatalf("SyncCoupons() error = %v, want ErrPartialSyncFailure", syncErr)
	}
	if summary == nil {
		t.Fatalf("summary = nil, want summary")
	}
	if summary.Processed != 2 || summary.Created != 1 || summary.Failed != 1 {
		t.Fatalf("summary = %+v, want processed=2 created=1 failed=1", summary)
	}
	if recorder.completeCalled {
		t.Fatalf("expected CompleteRun not to be called")
	}
	if !recorder.failCalled {
		t.Fatalf("expected FailRun to be called")
	}
	if recorder.processed != 2 || recorder.succeeded != 1 || recorder.failed != 1 || recorder.skipped != 0 {
		t.Fatalf("recorder counters = processed=%d succeeded=%d failed=%d skipped=%d, want 2/1/1/0", recorder.processed, recorder.succeeded, recorder.failed, recorder.skipped)
	}
	if len(recorder.syncErrors) != 1 {
		t.Fatalf("len(syncErrors) = %d, want %d", len(recorder.syncErrors), 1)
	}
	if recorder.syncErrors[0].Code != "coupon_upsert_failed" {
		t.Fatalf("syncErrors[0].Code = %q, want %q", recorder.syncErrors[0].Code, "coupon_upsert_failed")
	}
	if !strings.Contains(recorder.syncErrors[0].Message, "BROKEN10") {
		t.Fatalf("syncErrors[0].Message = %q, want message containing coupon code", recorder.syncErrors[0].Message)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("len(events) = %d, want %d", len(publisher.events), 2)
	}
	if publisher.events[0].Topic != event.TopicCouponsSyncStarted {
		t.Fatalf("events[0].Topic = %q, want %q", publisher.events[0].Topic, event.TopicCouponsSyncStarted)
	}
	if publisher.events[1].Topic != event.TopicCouponsSyncFailed {
		t.Fatalf("events[1].Topic = %q, want %q", publisher.events[1].Topic, event.TopicCouponsSyncFailed)
	}
}
