package http

import (
	"context"
	stdhttp "net/http"
	"testing"
	"time"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/syncrecord/application"
	"mannaiah/module/syncrecord/domain"
	"mannaiah/module/syncrecord/port"
)

// serviceMock defines sync record service behavior for handler tests.
type serviceMock struct {
	getRunFn     func(ctx context.Context, runID string) (*domain.SyncRun, error)
	listRunsFn   func(ctx context.Context, query port.ListQuery) (*application.ListResult, error)
	statsSinceFn func(ctx context.Context, since time.Time) (*domain.RunStats, error)
}

// GetRun executes configured get behavior.
func (m serviceMock) GetRun(ctx context.Context, runID string) (*domain.SyncRun, error) {
	return m.getRunFn(ctx, runID)
}

// ListRuns executes configured list behavior.
func (m serviceMock) ListRuns(ctx context.Context, query port.ListQuery) (*application.ListResult, error) {
	return m.listRunsFn(ctx, query)
}

// StatsSince executes configured stats behavior.
func (m serviceMock) StatsSince(ctx context.Context, since time.Time) (*domain.RunStats, error) {
	return m.statsSinceFn(ctx, since)
}

// TestHandlerListRuns verifies list endpoint behavior.
func TestHandlerListRuns(t *testing.T) {
	handler, err := NewHandler(serviceMock{
		getRunFn: func(ctx context.Context, runID string) (*domain.SyncRun, error) {
			return &domain.SyncRun{}, nil
		},
		listRunsFn: func(ctx context.Context, query port.ListQuery) (*application.ListResult, error) {
			if query.Kind != "woocommerce.contacts" {
				t.Fatalf("query.Kind = %q", query.Kind)
			}
			return &application.ListResult{Data: []domain.SyncRun{{ID: "run-1"}}, Page: 1, Limit: 10, Total: 1, TotalPages: 1}, nil
		},
		statsSinceFn: func(ctx context.Context, since time.Time) (*domain.RunStats, error) {
			return &domain.RunStats{}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := newHTTPServerForTest(t, handler)
	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/syncrecord/runs?kind=woocommerce.contacts&page=1&limit=10", nil)
	response, requestErr := server.App().Test(request)
	if requestErr != nil {
		t.Fatalf("server.App().Test() error = %v", requestErr)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}

// TestHandlerGetRunNotFound verifies not-found mapping behavior.
func TestHandlerGetRunNotFound(t *testing.T) {
	handler, err := NewHandler(serviceMock{
		getRunFn: func(ctx context.Context, runID string) (*domain.SyncRun, error) {
			return nil, domain.ErrRunNotFound
		},
		listRunsFn: func(ctx context.Context, query port.ListQuery) (*application.ListResult, error) {
			return &application.ListResult{}, nil
		},
		statsSinceFn: func(ctx context.Context, since time.Time) (*domain.RunStats, error) {
			return &domain.RunStats{}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := newHTTPServerForTest(t, handler)
	request, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/syncrecord/runs/run-1", nil)
	response, requestErr := server.App().Test(request)
	if requestErr != nil {
		t.Fatalf("server.App().Test() error = %v", requestErr)
	}
	if response.StatusCode != stdhttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.StatusCode)
	}
}

// newHTTPServerForTest creates test server values for handler tests.
func newHTTPServerForTest(t *testing.T, handler *Handler) *corehttp.Server {
	t.Helper()
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8099}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)
	return server
}
