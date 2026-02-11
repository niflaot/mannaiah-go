package cron

import (
	"context"
	errorspkg "errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestNewValidation verifies scheduler construction validation behavior.
func TestNewValidation(t *testing.T) {
	if _, err := New(Config{Location: "Invalid/Location"}, nil); !errorspkg.Is(err, ErrInvalidLocation) {
		t.Fatalf("New() error = %v, want ErrInvalidLocation", err)
	}

	scheduler, err := New(Config{}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if scheduler == nil {
		t.Fatalf("expected non-nil scheduler")
	}
}

// TestNewScheduler verifies abstract scheduler construction behavior.
func TestNewScheduler(t *testing.T) {
	scheduler, err := NewScheduler(Config{Location: "UTC"}, nil)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}
	if scheduler == nil {
		t.Fatalf("expected non-nil abstract scheduler")
	}
}

// TestAddValidation verifies add-operation validation behavior.
func TestAddValidation(t *testing.T) {
	scheduler, err := New(Config{Location: "UTC"}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, err := scheduler.Add("", JobFunc(func() {})); !errorspkg.Is(err, ErrEmptySpec) {
		t.Fatalf("Add(empty spec) error = %v, want ErrEmptySpec", err)
	}

	if _, err := scheduler.Add("* * * * *", nil); !errorspkg.Is(err, ErrNilJob) {
		t.Fatalf("Add(nil job) error = %v, want ErrNilJob", err)
	}

	if _, err := scheduler.Add("not-a-spec", JobFunc(func() {})); !errorspkg.Is(err, ErrInvalidSpec) {
		t.Fatalf("Add(invalid spec) error = %v, want ErrInvalidSpec", err)
	}

	if _, err := scheduler.AddFunc("* * * * *", nil); !errorspkg.Is(err, ErrNilFunc) {
		t.Fatalf("AddFunc(nil) error = %v, want ErrNilFunc", err)
	}
}

// TestEntriesAndRemove verifies entry metadata and removal behavior.
func TestEntriesAndRemove(t *testing.T) {
	scheduler, err := New(Config{Location: "UTC"}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	id, err := scheduler.AddFunc("@every 1m", func() {})
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	entries := scheduler.Entries()
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 1)
	}
	if entries[0].ID != id {
		t.Fatalf("entries[0].ID = %d, want %d", entries[0].ID, id)
	}
	if entries[0].Spec != "@every 1m" {
		t.Fatalf("entries[0].Spec = %q, want %q", entries[0].Spec, "@every 1m")
	}

	scheduler.Remove(id)
	if len(scheduler.Entries()) != 0 {
		t.Fatalf("expected zero entries after remove")
	}
}

// TestStartAndStop verifies asynchronous scheduler start and stop behavior.
func TestStartAndStop(t *testing.T) {
	scheduler, err := New(Config{Location: "UTC", WithSeconds: true}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	runs := make(chan struct{}, 1)
	if _, err := scheduler.AddFunc("*/1 * * * * *", func() {
		select {
		case runs <- struct{}{}:
		default:
		}
	}); err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	scheduler.Start()
	waitForRun(t, runs, 2*time.Second)

	stopContext, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	if err := scheduler.Stop(stopContext); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

// TestRunAndStopNilContext verifies blocking run and nil-context stop behavior.
func TestRunAndStopNilContext(t *testing.T) {
	scheduler, err := New(Config{Location: "UTC", WithSeconds: true}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	runs := make(chan struct{}, 1)
	if _, err := scheduler.AddFunc("*/1 * * * * *", func() {
		select {
		case runs <- struct{}{}:
		default:
		}
	}); err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	done := make(chan struct{}, 1)
	go func() {
		scheduler.Run()
		done <- struct{}{}
	}()

	waitForRun(t, runs, 2*time.Second)
	if err := scheduler.Stop(nil); err != nil {
		t.Fatalf("Stop(nil) error = %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("scheduler run did not stop")
	}
}

// TestStopContextDeadline verifies stop timeout behavior for long-running jobs.
func TestStopContextDeadline(t *testing.T) {
	scheduler, err := New(Config{Location: "UTC", WithSeconds: true}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	once := sync.Once{}
	if _, err := scheduler.AddFunc("*/1 * * * * *", func() {
		once.Do(func() {
			started <- struct{}{}
		})
		<-release
	}); err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	scheduler.Start()
	waitForRun(t, started, 2*time.Second)

	stopContext, stopCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stopCancel()
	err = scheduler.Stop(stopContext)
	if !errorspkg.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Stop(timeout) error = %v, want context deadline exceeded", err)
	}

	close(release)

	finalContext, finalCancel := context.WithTimeout(context.Background(), time.Second)
	defer finalCancel()
	if err := scheduler.Stop(finalContext); err != nil {
		t.Fatalf("Stop(final) error = %v", err)
	}
}

// TestPanicRecovery verifies panic recovery logging behavior.
func TestPanicRecovery(t *testing.T) {
	observedCore, observed := observer.New(zap.ErrorLevel)
	logger := zap.New(observedCore)

	scheduler, err := New(Config{Location: "UTC"}, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	wrapped := scheduler.wrapJob(JobFunc(func() {
		panic("panic test")
	}))
	wrapped.Run()

	if observed.FilterMessage("cron job panic recovered").Len() == 0 {
		t.Fatalf("expected panic recovery logs")
	}
}

// TestJobFuncRun verifies JobFunc adapter execution behavior.
func TestJobFuncRun(t *testing.T) {
	var called atomic.Bool

	job := JobFunc(func() {
		called.Store(true)
	})
	job.Run()

	if !called.Load() {
		t.Fatalf("expected JobFunc to be executed")
	}
}

// BenchmarkAddAndRemove measures scheduler add/remove hot-path behavior.
func BenchmarkAddAndRemove(b *testing.B) {
	scheduler, err := New(Config{Location: "UTC"}, nil)
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}

	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		id, addErr := scheduler.AddFunc("@every 1m", func() {})
		if addErr != nil {
			b.Fatalf("AddFunc() error = %v", addErr)
		}
		scheduler.Remove(id)
	}
}

// waitForRun waits for scheduled execution signals.
func waitForRun(t *testing.T, signal <-chan struct{}, timeout time.Duration) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(timeout):
		t.Fatalf("timeout waiting for scheduled execution")
	}
}
