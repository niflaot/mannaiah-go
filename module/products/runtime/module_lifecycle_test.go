package runtime

import (
	"context"
	"errors"
	"testing"

	corecron "mannaiah/module/core/cron"
	storefrontdomain "mannaiah/module/products/domain/storefront"

	"go.uber.org/zap"
)

// storefrontCronMock defines scheduler behavior for lifecycle tests.
type storefrontCronMock struct {
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
func (m *storefrontCronMock) Add(spec string, job corecron.Job) (corecron.EntryID, error) {
	if m.addErr != nil {
		return 0, m.addErr
	}
	m.addedSpecs = append(m.addedSpecs, spec)
	return corecron.EntryID(len(m.addedSpecs)), nil
}

// AddFunc registers a function job.
func (m *storefrontCronMock) AddFunc(spec string, fn func()) (corecron.EntryID, error) {
	if m.addErr != nil {
		return 0, m.addErr
	}
	m.addedSpecs = append(m.addedSpecs, spec)
	return corecron.EntryID(len(m.addedSpecs)), nil
}

// Remove removes a scheduled entry.
func (m *storefrontCronMock) Remove(id corecron.EntryID) {
	m.removed = append(m.removed, id)
}

// Entries returns current scheduled entries.
func (m *storefrontCronMock) Entries() []corecron.Entry {
	return nil
}

// Start starts scheduling.
func (m *storefrontCronMock) Start() {
	m.started = true
}

// Run blocks until stopped.
func (m *storefrontCronMock) Run() {}

// Stop stops scheduling.
func (m *storefrontCronMock) Stop(ctx context.Context) error {
	m.stopped = true
	return nil
}

// storefrontServiceMock defines storefront service behavior for lifecycle tests.
type storefrontServiceMock struct{}

// Get returns nil navigation values for lifecycle tests.
func (m storefrontServiceMock) Get(ctx context.Context) (*storefrontdomain.Navigation, error) {
	return nil, nil
}

// Regenerate returns nil navigation values for lifecycle tests.
func (m storefrontServiceMock) Regenerate(ctx context.Context) (*storefrontdomain.Navigation, error) {
	return &storefrontdomain.Navigation{}, nil
}

// TriggerRefresh is unused in lifecycle tests.
func (m storefrontServiceMock) TriggerRefresh(ctx context.Context) {}

// TestStartRegistersAndStartsStorefrontScheduler verifies cron registration and start behavior.
func TestStartRegistersAndStartsStorefrontScheduler(t *testing.T) {
	scheduler := &storefrontCronMock{}
	module := &Module{
		cfg:               Config{StorefrontNavigationEnabled: true, StorefrontNavigationRefreshHours: 12, StorefrontNavigationRegenerationTimeoutSeconds: 30},
		storefrontService: storefrontServiceMock{},
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
	if scheduler.addedSpecs[0] != "@every 12h0m0s" {
		t.Fatalf("spec = %q, want %q", scheduler.addedSpecs[0], "@every 12h0m0s")
	}
}

// TestStartSkipsWithoutScheduler verifies Start() no-ops without scheduler dependencies.
func TestStartSkipsWithoutScheduler(t *testing.T) {
	module := &Module{
		cfg:               Config{StorefrontNavigationEnabled: true, StorefrontNavigationRefreshHours: 12},
		storefrontService: storefrontServiceMock{},
		logger:            zap.NewNop(),
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

// TestStartReturnsRegistrationErrors verifies cron registration failures are propagated.
func TestStartReturnsRegistrationErrors(t *testing.T) {
	module := &Module{
		cfg:               Config{StorefrontNavigationEnabled: true, StorefrontNavigationRefreshHours: 12},
		storefrontService: storefrontServiceMock{},
		scheduler:         &storefrontCronMock{addErr: errors.New("boom")},
		logger:            zap.NewNop(),
	}

	if err := module.Start(context.Background()); err == nil {
		t.Fatalf("Start() error = nil, want non-nil")
	}
}

// TestStopRemovesEntryAndStopsScheduler verifies Stop() removes registered cron entries.
func TestStopRemovesEntryAndStopsScheduler(t *testing.T) {
	scheduler := &storefrontCronMock{}
	module := &Module{
		cfg:               Config{StorefrontNavigationEnabled: true, StorefrontNavigationRefreshHours: 12},
		storefrontService: storefrontServiceMock{},
		scheduler:         scheduler,
		logger:            zap.NewNop(),
	}

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
