package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"mannaiah/module/syncrecord/domain"
	"mannaiah/module/syncrecord/port"
)

// repositoryMock defines in-memory repository behavior for service tests.
type repositoryMock struct {
	createRunFn     func(ctx context.Context, run *domain.SyncRun) error
	completeRunFn   func(ctx context.Context, input port.CompleteInput) error
	addRunErrorsFn  func(ctx context.Context, errors []domain.SyncRunError) error
	getRunByIDFn    func(ctx context.Context, runID string) (*domain.SyncRun, error)
	listRunsFn      func(ctx context.Context, query port.ListQuery) ([]domain.SyncRun, int64, error)
	statsSinceFn    func(ctx context.Context, since time.Time) (*domain.RunStats, error)
	cleanupBeforeFn func(ctx context.Context, cutoff time.Time) (int64, error)
}

// CreateRun persists a new running sync run.
func (m repositoryMock) CreateRun(ctx context.Context, run *domain.SyncRun) error {
	if m.createRunFn == nil {
		return nil
	}
	return m.createRunFn(ctx, run)
}

// CompleteRun updates run status and counters with terminal values.
func (m repositoryMock) CompleteRun(ctx context.Context, input port.CompleteInput) error {
	if m.completeRunFn == nil {
		return nil
	}
	return m.completeRunFn(ctx, input)
}

// AddRunErrors persists child error rows for a run.
func (m repositoryMock) AddRunErrors(ctx context.Context, errors []domain.SyncRunError) error {
	if m.addRunErrorsFn == nil {
		return nil
	}
	return m.addRunErrorsFn(ctx, errors)
}

// GetRunByID retrieves a run with child errors by id.
func (m repositoryMock) GetRunByID(ctx context.Context, runID string) (*domain.SyncRun, error) {
	if m.getRunByIDFn == nil {
		return nil, domain.ErrRunNotFound
	}
	return m.getRunByIDFn(ctx, runID)
}

// ListRuns returns paged run rows and total count for filters.
func (m repositoryMock) ListRuns(ctx context.Context, query port.ListQuery) ([]domain.SyncRun, int64, error) {
	if m.listRunsFn == nil {
		return nil, 0, nil
	}
	return m.listRunsFn(ctx, query)
}

// StatsSince returns aggregate stats from a lower-bound timestamp.
func (m repositoryMock) StatsSince(ctx context.Context, since time.Time) (*domain.RunStats, error) {
	if m.statsSinceFn == nil {
		return &domain.RunStats{}, nil
	}
	return m.statsSinceFn(ctx, since)
}

// CleanupBefore deletes runs older than cutoff and returns deleted rows count.
func (m repositoryMock) CleanupBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	if m.cleanupBeforeFn == nil {
		return 0, nil
	}
	return m.cleanupBeforeFn(ctx, cutoff)
}

// TestStartRun validates creation payload values.
func TestStartRun(t *testing.T) {
	called := false
	svc, err := NewService(repositoryMock{createRunFn: func(ctx context.Context, run *domain.SyncRun) error {
		called = true
		if run.ID == "" {
			t.Fatalf("run.ID is empty")
		}
		if run.Status != domain.RunStatusRunning {
			t.Fatalf("run.Status = %q", run.Status)
		}
		return nil
	}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	runID, runErr := svc.StartRun(context.Background(), port.StartRunInput{Kind: domain.KindWooCommerceContacts, Trigger: domain.TriggerCron})
	if runErr != nil {
		t.Fatalf("StartRun() error = %v", runErr)
	}
	if !called {
		t.Fatalf("createRun was not called")
	}
	if runID == "" {
		t.Fatalf("runID is empty")
	}
}

// TestFailRunPersistsErrors validates failure path behavior.
func TestFailRunPersistsErrors(t *testing.T) {
	errorInserted := false
	completed := false
	svc, err := NewService(repositoryMock{
		addRunErrorsFn: func(ctx context.Context, entries []domain.SyncRunError) error {
			errorInserted = len(entries) == 1 && entries[0].RunID == "run-1"
			return nil
		},
		completeRunFn: func(ctx context.Context, input port.CompleteInput) error {
			completed = input.Status == domain.RunStatusFailed
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	runErr := svc.FailRun(context.Background(), port.FinishRunInput{
		RunID: "run-1",
		Errors: []domain.SyncRunError{{
			Message: "failure",
		}},
	})
	if runErr != nil {
		t.Fatalf("FailRun() error = %v", runErr)
	}
	if !errorInserted {
		t.Fatalf("errors were not inserted")
	}
	if !completed {
		t.Fatalf("run was not marked as failed")
	}
}

// TestCompleteRunRequiresID validates identifier validation behavior.
func TestCompleteRunRequiresID(t *testing.T) {
	svc, err := NewService(repositoryMock{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	runErr := svc.CompleteRun(context.Background(), port.FinishRunInput{RunID: " "})
	if !errors.Is(runErr, domain.ErrInvalidRunID) {
		t.Fatalf("CompleteRun() error = %v, want ErrInvalidRunID", runErr)
	}
}
