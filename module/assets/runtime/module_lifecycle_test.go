package runtime

import (
	"context"
	"errors"
	"testing"

	corecron "mannaiah/module/core/cron"
)

// assetsCronMock defines cron scheduler behavior for lifecycle tests.
type assetsCronMock struct {
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
func (m *assetsCronMock) Add(spec string, job corecron.Job) (corecron.EntryID, error) {
	if m.addErr != nil {
		return 0, m.addErr
	}
	m.addedSpecs = append(m.addedSpecs, spec)
	return corecron.EntryID(len(m.addedSpecs)), nil
}

// AddFunc registers a function job.
func (m *assetsCronMock) AddFunc(spec string, fn func()) (corecron.EntryID, error) {
	if m.addErr != nil {
		return 0, m.addErr
	}
	m.addedSpecs = append(m.addedSpecs, spec)
	return corecron.EntryID(len(m.addedSpecs)), nil
}

// Remove removes a scheduled entry.
func (m *assetsCronMock) Remove(id corecron.EntryID) {
	m.removed = append(m.removed, id)
}

// Entries returns current scheduled entries.
func (m *assetsCronMock) Entries() []corecron.Entry {
	return nil
}

// Start starts scheduling.
func (m *assetsCronMock) Start() {
	m.started = true
}

// Run blocks until stopped.
func (m *assetsCronMock) Run() {}

// Stop stops scheduling.
func (m *assetsCronMock) Stop(ctx context.Context) error {
	m.stopped = true
	return nil
}

// TestStartRegistersAndStartsScheduler verifies cron registration and start behavior.
func TestStartRegistersAndStartsScheduler(t *testing.T) {
	db := newDBForTest(t)
	module, err := NewWithConfig(Config{
		JPGWorkerEnabled: true,
		JPGWorkerCron:    "*/5 * * * *",
		JPGWorkerTags:    "marketplaces",
	}, db, runtimeStorageMock{}, nil)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}

	scheduler := &assetsCronMock{}
	module.ConfigureScheduler(scheduler)
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

// TestStartEnabledWithoutScheduler verifies Start() validation for enabled worker without scheduler.
func TestStartEnabledWithoutScheduler(t *testing.T) {
	db := newDBForTest(t)
	module, err := NewWithConfig(Config{
		JPGWorkerEnabled: true,
		JPGWorkerCron:    "*/5 * * * *",
		JPGWorkerTags:    "marketplaces",
	}, db, runtimeStorageMock{}, nil)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}

	if err := module.Start(context.Background()); !errors.Is(err, ErrNilSchedulerWhenEnabled) {
		t.Fatalf("Start() error = %v, want %v", err, ErrNilSchedulerWhenEnabled)
	}
}

// TestStartSkipsWhenTagsAreMissing verifies Start() no-ops without configured worker tags.
func TestStartSkipsWhenTagsAreMissing(t *testing.T) {
	db := newDBForTest(t)
	module, err := NewWithConfig(Config{
		JPGWorkerEnabled: true,
		JPGWorkerCron:    "*/5 * * * *",
		JPGWorkerTags:    "   ",
	}, db, runtimeStorageMock{}, nil)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}

	scheduler := &assetsCronMock{}
	module.ConfigureScheduler(scheduler)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(scheduler.addedSpecs) != 0 {
		t.Fatalf("len(addedSpecs) = %d, want 0", len(scheduler.addedSpecs))
	}
	if scheduler.started {
		t.Fatalf("scheduler should not be started without worker tags")
	}
}

// TestStopRemovesEntryAndStops verifies Stop() removes scheduled entry and stops scheduler.
func TestStopRemovesEntryAndStops(t *testing.T) {
	db := newDBForTest(t)
	module, err := NewWithConfig(Config{
		JPGWorkerEnabled: true,
		JPGWorkerCron:    "*/5 * * * *",
		JPGWorkerTags:    "marketplaces",
	}, db, runtimeStorageMock{}, nil)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}

	scheduler := &assetsCronMock{}
	module.ConfigureScheduler(scheduler)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
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

// TestStartNilModule verifies nil module error.
func TestStartNilModule(t *testing.T) {
	var module *Module
	if err := module.Start(context.Background()); !errors.Is(err, ErrModuleNotInitialized) {
		t.Fatalf("Start() error = %v, want %v", err, ErrModuleNotInitialized)
	}
}

var _ corecron.Scheduler = (*assetsCronMock)(nil)
