package runtime

import (
	"context"
	"testing"

	corecron "mannaiah/module/core/cron"

	"go.uber.org/zap"
)

// TestStartRegistersShopifySchedulers verifies enabled Shopify cron jobs are registered at startup.
func TestStartRegistersShopifySchedulers(t *testing.T) {
	scheduler := newRecordingScheduler()
	module := &Module{
		cfg: Config{
			SyncContacts:     true,
			SyncContactsCron: "*/15 * * * *",
			SyncOrders:       true,
			SyncOrdersCron:   "*/10 * * * *",
		},
		logger: zap.NewNop(),
	}
	module.ConfigureScheduler(scheduler)

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !scheduler.started {
		t.Fatal("scheduler should be started")
	}
	if len(scheduler.specs) != 2 {
		t.Fatalf("registered specs = %v, want two cron jobs", scheduler.specs)
	}
	if scheduler.specs[1] != "*/15 * * * *" {
		t.Fatalf("contact cron spec = %q", scheduler.specs[1])
	}
	if scheduler.specs[2] != "*/10 * * * *" {
		t.Fatalf("order cron spec = %q", scheduler.specs[2])
	}
}

// TestStopRemovesShopifySchedulers verifies Shopify cron entries are removed during shutdown.
func TestStopRemovesShopifySchedulers(t *testing.T) {
	scheduler := newRecordingScheduler()
	module := &Module{
		cfg: Config{
			SyncContacts:     true,
			SyncContactsCron: "*/15 * * * *",
			SyncOrders:       true,
			SyncOrdersCron:   "*/10 * * * *",
		},
		logger: zap.NewNop(),
	}
	module.ConfigureScheduler(scheduler)

	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if !scheduler.stopped {
		t.Fatal("scheduler should be stopped")
	}
	if len(scheduler.removed) != 2 {
		t.Fatalf("removed entries = %v, want two removed entries", scheduler.removed)
	}
	if module.contactsSchedulerEntryID != 0 || module.ordersSchedulerEntryID != 0 {
		t.Fatalf("scheduler entry ids should be reset, got contacts=%d orders=%d", module.contactsSchedulerEntryID, module.ordersSchedulerEntryID)
	}
}

type recordingScheduler struct {
	nextID  corecron.EntryID
	specs   map[corecron.EntryID]string
	removed []corecron.EntryID
	started bool
	stopped bool
}

func newRecordingScheduler() *recordingScheduler {
	return &recordingScheduler{
		nextID: 1,
		specs:  make(map[corecron.EntryID]string),
	}
}

func (s *recordingScheduler) Add(spec string, job corecron.Job) (corecron.EntryID, error) {
	id := s.nextID
	s.nextID++
	s.specs[id] = spec
	return id, nil
}

func (s *recordingScheduler) AddFunc(spec string, job func()) (corecron.EntryID, error) {
	return s.Add(spec, corecron.JobFunc(job))
}

func (s *recordingScheduler) Remove(id corecron.EntryID) {
	s.removed = append(s.removed, id)
	delete(s.specs, id)
}

func (s *recordingScheduler) Entries() []corecron.Entry {
	entries := make([]corecron.Entry, 0, len(s.specs))
	for id, spec := range s.specs {
		entries = append(entries, corecron.Entry{ID: id, Spec: spec})
	}
	return entries
}

func (s *recordingScheduler) Start() {
	s.started = true
}

func (s *recordingScheduler) Run() {
	s.started = true
}

func (s *recordingScheduler) Stop(ctx context.Context) error {
	s.stopped = true
	return nil
}
