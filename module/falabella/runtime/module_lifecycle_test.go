package runtime

import (
	"context"
	"errors"
	"testing"

	corecron "mannaiah/module/core/cron"
	syncstatusservice "mannaiah/module/falabella/application/syncstatus/service"
	syncdomain "mannaiah/module/falabella/domain/sync"

	"go.uber.org/zap"
)

// cronMock defines cron scheduler behavior for lifecycle tests.
type cronMock struct {
	// addedSpecs captures registered cron specs.
	addedSpecs []string
	// addErr defines AddFunc() errors.
	addErr error
	// started reports whether Start() was called.
	started bool
	// stopped reports whether Stop() was called.
	stopped bool
	// removed captures removed entry IDs.
	removed []corecron.EntryID
}

// Add registers a job.
func (m *cronMock) Add(spec string, job corecron.Job) (corecron.EntryID, error) {
	if m.addErr != nil {
		return 0, m.addErr
	}
	m.addedSpecs = append(m.addedSpecs, spec)
	return corecron.EntryID(len(m.addedSpecs)), nil
}

// AddFunc registers a function job.
func (m *cronMock) AddFunc(spec string, fn func()) (corecron.EntryID, error) {
	if m.addErr != nil {
		return 0, m.addErr
	}
	m.addedSpecs = append(m.addedSpecs, spec)
	return corecron.EntryID(len(m.addedSpecs)), nil
}

// Remove removes a scheduled entry.
func (m *cronMock) Remove(id corecron.EntryID) {
	m.removed = append(m.removed, id)
}

// Entries returns current scheduled entries.
func (m *cronMock) Entries() []corecron.Entry {
	return nil
}

// Start starts scheduling.
func (m *cronMock) Start() {
	m.started = true
}

// Run blocks until stopped.
func (m *cronMock) Run() {}

// Stop stops scheduling.
func (m *cronMock) Stop(ctx context.Context) error {
	m.stopped = true
	return nil
}

// syncStatusSvcMock defines sync status service behavior for lifecycle tests.
type syncStatusSvcMock struct{}

// RecordEntry returns nil.
func (m *syncStatusSvcMock) RecordEntry(ctx context.Context, entry *syncdomain.SyncEntry) error {
	return nil
}

// GetExecutionByID returns nil.
func (m *syncStatusSvcMock) GetExecutionByID(ctx context.Context, executionID string) (*syncdomain.SyncExecution, error) {
	return nil, nil
}

// GetByFeedID returns nil.
func (m *syncStatusSvcMock) GetByFeedID(ctx context.Context, feedID string) (*syncdomain.SyncEntry, error) {
	return nil, nil
}

// GetByExecutionID returns nil.
func (m *syncStatusSvcMock) GetByExecutionID(ctx context.Context, executionID string) ([]syncdomain.SyncEntry, error) {
	return nil, nil
}

// GetByProductID returns nil.
func (m *syncStatusSvcMock) GetByProductID(ctx context.Context, productID string) ([]syncdomain.SyncEntry, error) {
	return nil, nil
}

// ResolveFeedStatus returns nil.
func (m *syncStatusSvcMock) ResolveFeedStatus(ctx context.Context, feedID string) (*syncstatusservice.ResolveResult, error) {
	return nil, nil
}

// ResolvePendingFeeds returns a zero result.
func (m *syncStatusSvcMock) ResolvePendingFeeds(ctx context.Context, limit int) (*syncstatusservice.ResolvePendingResult, error) {
	return &syncstatusservice.ResolvePendingResult{}, nil
}

// TestStartRegistersAndStartsScheduler verifies cron registration and start behavior.
func TestStartRegistersAndStartsScheduler(t *testing.T) {
	scheduler := &cronMock{}
	module := &Module{
		cfg:               Config{SyncStatusCron: "*/5 * * * *", SyncStatusBatchSize: 10, RequestTimeoutMS: 5000},
		syncStatusService: &syncStatusSvcMock{},
		scheduler:         scheduler,
		logger:            zap.NewNop(),
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !scheduler.started {
		t.Fatalf("scheduler should be started")
	}
	if len(scheduler.addedSpecs) != 1 {
		t.Fatalf("len(addedSpecs) = %d, want 1", len(scheduler.addedSpecs))
	}
	if scheduler.addedSpecs[0] != "*/5 * * * *" {
		t.Fatalf("spec = %q, want %q", scheduler.addedSpecs[0], "*/5 * * * *")
	}
}

// TestStartIdempotent verifies repeated Start() calls are idempotent.
func TestStartIdempotent(t *testing.T) {
	scheduler := &cronMock{}
	module := &Module{
		cfg:               Config{SyncStatusCron: "*/5 * * * *"},
		syncStatusService: &syncStatusSvcMock{},
		scheduler:         scheduler,
		logger:            zap.NewNop(),
	}

	_ = module.Start(context.Background())
	_ = module.Start(context.Background())

	if len(scheduler.addedSpecs) != 1 {
		t.Fatalf("len(addedSpecs) = %d, want 1 (idempotent)", len(scheduler.addedSpecs))
	}
}

// TestStartSkipsWithoutSyncStatus verifies Start() no-ops without sync status service.
func TestStartSkipsWithoutSyncStatus(t *testing.T) {
	scheduler := &cronMock{}
	module := &Module{
		cfg:       Config{SyncStatusCron: "*/5 * * * *"},
		scheduler: scheduler,
		logger:    zap.NewNop(),
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if scheduler.started {
		t.Fatalf("scheduler should not be started without sync status service")
	}
}

// TestStartSkipsWithoutScheduler verifies Start() no-ops without scheduler.
func TestStartSkipsWithoutScheduler(t *testing.T) {
	module := &Module{
		cfg:               Config{SyncStatusCron: "*/5 * * * *"},
		syncStatusService: &syncStatusSvcMock{},
		logger:            zap.NewNop(),
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

// TestStartSkipsEmptyCron verifies Start() no-ops with empty cron spec.
func TestStartSkipsEmptyCron(t *testing.T) {
	scheduler := &cronMock{}
	module := &Module{
		cfg:               Config{SyncStatusCron: ""},
		syncStatusService: &syncStatusSvcMock{},
		scheduler:         scheduler,
		logger:            zap.NewNop(),
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if scheduler.started {
		t.Fatalf("scheduler should not be started with empty cron spec")
	}
}

// TestStartSchedulerAddError verifies Start() error propagation from scheduler.AddFunc.
func TestStartSchedulerAddError(t *testing.T) {
	scheduler := &cronMock{addErr: errors.New("bad cron")}
	module := &Module{
		cfg:               Config{SyncStatusCron: "bad"},
		syncStatusService: &syncStatusSvcMock{},
		scheduler:         scheduler,
		logger:            zap.NewNop(),
	}

	if err := module.Start(context.Background()); err == nil {
		t.Fatalf("Start() expected error for bad cron spec")
	}
}

// TestStopRemovesEntryAndStops verifies Stop() removes scheduled entry and stops scheduler.
func TestStopRemovesEntryAndStops(t *testing.T) {
	scheduler := &cronMock{}
	module := &Module{
		cfg:               Config{SyncStatusCron: "*/5 * * * *", RequestTimeoutMS: 5000},
		syncStatusService: &syncStatusSvcMock{},
		scheduler:         scheduler,
		logger:            zap.NewNop(),
	}

	_ = module.Start(context.Background())
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if !scheduler.stopped {
		t.Fatalf("scheduler should be stopped")
	}
	if len(scheduler.removed) != 1 {
		t.Fatalf("len(removed) = %d, want 1", len(scheduler.removed))
	}
}

// TestStopIdempotent verifies repeated Stop() calls are safe.
func TestStopIdempotent(t *testing.T) {
	module := &Module{logger: zap.NewNop()}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop(not-started) error = %v", err)
	}
}

// TestStopNilModule verifies nil module safety.
func TestStopNilModule(t *testing.T) {
	var m *Module
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop(nil) error = %v", err)
	}
}

// TestStartNilModule verifies nil module error.
func TestStartNilModule(t *testing.T) {
	var m *Module
	if err := m.Start(context.Background()); err == nil {
		t.Fatalf("Start(nil) expected error")
	}
}

// TestConfigureScheduler verifies scheduler dependency injection.
func TestConfigureScheduler(t *testing.T) {
	module := &Module{logger: zap.NewNop()}
	scheduler := &cronMock{}
	module.ConfigureScheduler(scheduler)
	if module.scheduler != scheduler {
		t.Fatalf("scheduler not set")
	}
}

// TestConfigureSchedulerNilModule verifies nil module safety.
func TestConfigureSchedulerNilModule(t *testing.T) {
	var m *Module
	m.ConfigureScheduler(&cronMock{})
}

// TestStopWithNilContext verifies Stop() handles nil context.
func TestStopWithNilContext(t *testing.T) {
	scheduler := &cronMock{}
	module := &Module{
		cfg:               Config{SyncStatusCron: "*/5 * * * *", RequestTimeoutMS: 5000},
		syncStatusService: &syncStatusSvcMock{},
		scheduler:         scheduler,
		logger:            zap.NewNop(),
	}

	_ = module.Start(context.Background())
	var nilCtx context.Context
	if err := module.Stop(nilCtx); err != nil {
		t.Fatalf("Stop(nil ctx) error = %v", err)
	}
	if !scheduler.stopped {
		t.Fatalf("scheduler should be stopped")
	}
}

// Verify schedulerMock satisfies interface at compile time.
var _ corecron.Scheduler = (*cronMock)(nil)

// Verify syncStatusSvcMock satisfies interface at compile time.
var _ syncstatusservice.Service = (*syncStatusSvcMock)(nil)
