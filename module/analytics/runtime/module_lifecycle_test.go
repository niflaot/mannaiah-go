package runtime

import (
	"context"
	"errors"
	"testing"

	affinityapp "mannaiah/module/analytics/application/affinity"
	corecron "mannaiah/module/core/cron"
)

type analyticsCronMock struct {
	addedSpecs []string
	addErr     error
	started    bool
	stopped    bool
	removed    []corecron.EntryID
}

func (m *analyticsCronMock) Add(spec string, job corecron.Job) (corecron.EntryID, error) {
	if m.addErr != nil {
		return 0, m.addErr
	}
	m.addedSpecs = append(m.addedSpecs, spec)
	return corecron.EntryID(len(m.addedSpecs)), nil
}

func (m *analyticsCronMock) AddFunc(spec string, fn func()) (corecron.EntryID, error) {
	if m.addErr != nil {
		return 0, m.addErr
	}
	m.addedSpecs = append(m.addedSpecs, spec)
	return corecron.EntryID(len(m.addedSpecs)), nil
}

func (m *analyticsCronMock) Remove(id corecron.EntryID) {
	m.removed = append(m.removed, id)
}

func (m *analyticsCronMock) Entries() []corecron.Entry {
	return nil
}

func (m *analyticsCronMock) Start() {
	m.started = true
}

func (m *analyticsCronMock) Run() {}

func (m *analyticsCronMock) Stop(ctx context.Context) error {
	m.stopped = true
	return nil
}

// TestConfigureScheduler verifies scheduler dependency injection.
func TestConfigureScheduler(t *testing.T) {
	module := &Module{}
	scheduler := &analyticsCronMock{}
	module.ConfigureScheduler(scheduler)
	if module.scheduler != scheduler {
		t.Fatalf("scheduler not set")
	}
}

// TestStartRegistersAffinityRefreshCron verifies affinity refresh cron registration and startup.
func TestStartRegistersAffinityRefreshCron(t *testing.T) {
	affinitySvc, err := affinityapp.NewService(&noopAffinityStore{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	scheduler := &analyticsCronMock{}
	module := &Module{
		cfg: Config{
			Enabled:                  true,
			AffinityRefreshEnabled:   true,
			AffinityRefreshCron:      "*/30 * * * *",
			AffinityRefreshTimeoutMS: 600000,
		},
		affinityService: affinitySvc,
		scheduler:       scheduler,
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(scheduler.addedSpecs) != 1 {
		t.Fatalf("len(addedSpecs) = %d, want 1", len(scheduler.addedSpecs))
	}
	if scheduler.addedSpecs[0] != "*/30 * * * *" {
		t.Fatalf("addedSpecs[0] = %q, want %q", scheduler.addedSpecs[0], "*/30 * * * *")
	}
	if !scheduler.started {
		t.Fatalf("scheduler should be started")
	}
}

// TestStartSkipsWithoutAffinityRefreshEnabled verifies Start() no-ops when refresh cron is disabled.
func TestStartSkipsWithoutAffinityRefreshEnabled(t *testing.T) {
	affinitySvc, err := affinityapp.NewService(&noopAffinityStore{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	scheduler := &analyticsCronMock{}
	module := &Module{
		cfg: Config{
			Enabled:                true,
			AffinityRefreshEnabled: false,
			AffinityRefreshCron:    "*/30 * * * *",
		},
		affinityService: affinitySvc,
		scheduler:       scheduler,
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(scheduler.addedSpecs) != 0 {
		t.Fatalf("len(addedSpecs) = %d, want 0", len(scheduler.addedSpecs))
	}
	if scheduler.started {
		t.Fatalf("scheduler should not be started")
	}
}

// TestStartSchedulerAddError verifies Start() propagates cron registration failures.
func TestStartSchedulerAddError(t *testing.T) {
	affinitySvc, err := affinityapp.NewService(&noopAffinityStore{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	scheduler := &analyticsCronMock{addErr: errors.New("invalid cron")}
	module := &Module{
		cfg: Config{
			Enabled:                true,
			AffinityRefreshEnabled: true,
			AffinityRefreshCron:    "bad",
		},
		affinityService: affinitySvc,
		scheduler:       scheduler,
	}

	if err := module.Start(context.Background()); err == nil {
		t.Fatalf("Start() error = nil, want non-nil")
	}
}

// TestStopRemovesEntryAndStopsScheduler verifies Stop() removes cron entry and stops scheduler.
func TestStopRemovesEntryAndStopsScheduler(t *testing.T) {
	affinitySvc, err := affinityapp.NewService(&noopAffinityStore{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	scheduler := &analyticsCronMock{}
	module := &Module{
		cfg: Config{
			Enabled:                true,
			AffinityRefreshEnabled: true,
			AffinityRefreshCron:    "*/30 * * * *",
		},
		affinityService: affinitySvc,
		scheduler:       scheduler,
	}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := module.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if len(scheduler.removed) != 1 {
		t.Fatalf("len(removed) = %d, want 1", len(scheduler.removed))
	}
	if !scheduler.stopped {
		t.Fatalf("scheduler should be stopped")
	}
}
