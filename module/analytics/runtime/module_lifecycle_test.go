package runtime

import (
	"context"
	"testing"

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

// TestStartMarksModuleStarted verifies Start() transitions runtime state without scheduling CRM jobs.
func TestStartMarksModuleStarted(t *testing.T) {
	module := &Module{cfg: Config{Enabled: true}, scheduler: &analyticsCronMock{}}

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !module.started {
		t.Fatalf("module should be marked started")
	}
}

// TestStopStopsConfiguredScheduler verifies Stop() still closes optional scheduler dependencies.
func TestStopStopsConfiguredScheduler(t *testing.T) {
	scheduler := &analyticsCronMock{}
	module := &Module{cfg: Config{Enabled: true}, scheduler: scheduler, started: true}

	if err := module.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if !scheduler.stopped {
		t.Fatalf("scheduler should be stopped")
	}
}
