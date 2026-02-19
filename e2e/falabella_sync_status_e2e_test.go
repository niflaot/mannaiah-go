package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	corecron "mannaiah/module/core/cron"
	coredatabase "mannaiah/module/core/database"
	corehttp "mannaiah/module/core/http"
	"mannaiah/module/falabella"
)

// newFalabellaFeedStatusServer creates a mock Falabella API server that handles FeedStatus requests.
func newFalabellaFeedStatusServer(t *testing.T, feedResponses map[string]string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("Action")

		switch action {
		case "FeedStatus":
			feedID := r.URL.Query().Get("FeedID")
			xml, ok := feedResponses[feedID]
			if !ok {
				xml = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head><RequestAction>FeedStatus</RequestAction></Head>
  <Body>
    <FeedDetail>
      <Feed>%s</Feed>
      <Status>Queued</Status>
      <Action>ProductCreate</Action>
      <TotalRecords>1</TotalRecords>
      <ProcessedRecords>0</ProcessedRecords>
      <FailedRecords>0</FailedRecords>
    </FeedDetail>
  </Body>
</SuccessResponse>`, feedID)
			}
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(xml))

		case "GetBrands":
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head><RequestAction>GetBrands</RequestAction></Head>
  <Body><Brands></Brands></Body>
</SuccessResponse>`))

		case "ProductCreate":
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId>feed-e2e-001</RequestId>
    <RequestAction>ProductCreate</RequestAction>
  </Head>
  <Body></Body>
</SuccessResponse>`))

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
}

// TestFalabellaSyncStatusEndpointsE2E verifies the sync status HTTP endpoints return correct responses.
func TestFalabellaSyncStatusEndpointsE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("start falabella mock server")
	feedResponses := map[string]string{
		"feed-e2e-resolved": `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head><RequestAction>FeedStatus</RequestAction></Head>
  <Body>
    <FeedDetail>
      <Feed>feed-e2e-resolved</Feed>
      <Status>Finished</Status>
      <Action>ProductCreate</Action>
      <TotalRecords>1</TotalRecords>
      <ProcessedRecords>1</ProcessedRecords>
      <FailedRecords>0</FailedRecords>
    </FeedDetail>
  </Body>
</SuccessResponse>`,
	}
	mockServer := newFalabellaFeedStatusServer(t, feedResponses)
	defer mockServer.Close()

	tracer.Step("open sqlite database")
	db, err := coredatabase.Open(coredatabase.Config{
		Driver:       "sqlite",
		DSN:          "file::memory:?cache=shared",
		MaxOpenConns: 1,
	}, tracer.logger)
	if err != nil {
		t.Fatalf("coredatabase.Open() error = %v", err)
	}

	tracer.Step("initialize falabella module")
	module, err := falabella.New(falabella.Config{
		URL:                 mockServer.URL,
		UserID:              "e2e@test.com",
		APIKey:              "e2e-key-abc",
		RequestTimeoutMS:    2000,
		ValidationTimeoutMS: 1000,
	}, tracer.logger)
	if err != nil {
		t.Fatalf("falabella.New() error = %v", err)
	}

	tracer.Step("configure sync status persistence")
	if syncErr := module.ConfigureSyncStatus(db); syncErr != nil {
		t.Fatalf("ConfigureSyncStatus() error = %v", syncErr)
	}

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8501}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	tracer.Step("verify sync status feed endpoint returns 404 for unknown feed")
	req, _ := http.NewRequest(http.MethodGet, "/falabella/sync/status/feed/nonexistent-feed", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /sync/status/feed/nonexistent status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	tracer.Step("verify sync status product endpoint returns empty for unknown product")
	req, _ = http.NewRequest(http.MethodGet, "/falabella/sync/status/product/nonexistent-product", nil)
	resp, testErr = server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /sync/status/product/nonexistent status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	tracer.Step("verify resolve endpoint returns 404 for non-tracked feed")
	req, _ = http.NewRequest(http.MethodPost, "/falabella/sync/status/feed/feed-e2e-resolved/resolve", nil)
	resp, testErr = server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	// Resolve on a feed that's not tracked locally returns the result from Falabella API
	// The entry-not-found in DB is non-fatal for resolve (it still queries the Falabella API)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /sync/status/feed/feed-e2e-resolved/resolve status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var resolveBody map[string]any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&resolveBody); decodeErr != nil {
		t.Fatalf("decode resolve response error = %v", decodeErr)
	}
	if resolveBody["feedId"] != "feed-e2e-resolved" {
		t.Fatalf("resolve feedId = %v, want %q", resolveBody["feedId"], "feed-e2e-resolved")
	}
	if resolveBody["status"] != "Finished" {
		t.Fatalf("resolve status = %v, want %q", resolveBody["status"], "Finished")
	}

	tracer.AssertStepCount(6)
}

// TestFalabellaSyncStatusLifecycleE2E verifies the scheduler Start/Stop lifecycle.
func TestFalabellaSyncStatusLifecycleE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("start falabella mock server")
	mockServer := newFalabellaFeedStatusServer(t, nil)
	defer mockServer.Close()

	tracer.Step("open sqlite database")
	db, err := coredatabase.Open(coredatabase.Config{
		Driver:       "sqlite",
		DSN:          "file::memory:?cache=shared",
		MaxOpenConns: 1,
	}, tracer.logger)
	if err != nil {
		t.Fatalf("coredatabase.Open() error = %v", err)
	}

	tracer.Step("initialize falabella module")
	module, err := falabella.New(falabella.Config{
		URL:                 mockServer.URL,
		UserID:              "e2e@test.com",
		APIKey:              "e2e-key-abc",
		RequestTimeoutMS:    2000,
		ValidationTimeoutMS: 1000,
		SyncStatusCron:      "*/1 * * * *",
		SyncStatusBatchSize: 10,
	}, tracer.logger)
	if err != nil {
		t.Fatalf("falabella.New() error = %v", err)
	}

	tracer.Step("configure sync status and scheduler")
	if syncErr := module.ConfigureSyncStatus(db); syncErr != nil {
		t.Fatalf("ConfigureSyncStatus() error = %v", syncErr)
	}
	scheduler, err := corecron.NewScheduler(corecron.Config{Location: "UTC"}, tracer.logger)
	if err != nil {
		t.Fatalf("corecron.NewScheduler() error = %v", err)
	}
	module.ConfigureScheduler(scheduler)

	tracer.Step("start module")
	if startErr := module.Start(context.Background()); startErr != nil {
		t.Fatalf("Start() error = %v", startErr)
	}

	tracer.Step("verify scheduler has registered entries")
	entries := scheduler.Entries()
	if len(entries) != 1 {
		t.Fatalf("scheduler entries = %d, want 1", len(entries))
	}

	tracer.Step("stop module")
	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if stopErr := module.Stop(stopCtx); stopErr != nil {
		t.Fatalf("Stop() error = %v", stopErr)
	}

	tracer.Step("verify scheduler stopped (idempotent stop)")
	if stopErr := module.Stop(context.Background()); stopErr != nil {
		t.Fatalf("Stop() second call error = %v", stopErr)
	}

	tracer.AssertStepCount(7)
}

// TestFalabellaSyncStatusWithoutDBE2E verifies sync status endpoints return 503 without DB wiring.
func TestFalabellaSyncStatusWithoutDBE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("start falabella mock server")
	mockServer := newFalabellaFeedStatusServer(t, nil)
	defer mockServer.Close()

	tracer.Step("initialize falabella module without DB")
	module, err := falabella.New(falabella.Config{
		URL:                 mockServer.URL,
		UserID:              "e2e@test.com",
		APIKey:              "e2e-key-abc",
		RequestTimeoutMS:    2000,
		ValidationTimeoutMS: 1000,
	}, tracer.logger)
	if err != nil {
		t.Fatalf("falabella.New() error = %v", err)
	}

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8502}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	tracer.Step("verify sync status feed endpoint returns 503 without DB")
	req, _ := http.NewRequest(http.MethodGet, "/falabella/sync/status/feed/any-feed", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /sync/status/feed status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}

	tracer.Step("verify sync status product endpoint returns 503 without DB")
	req, _ = http.NewRequest(http.MethodGet, "/falabella/sync/status/product/any-product", nil)
	resp, testErr = server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /sync/status/product status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}

	tracer.Step("verify resolve endpoint returns 503 without DB")
	req, _ = http.NewRequest(http.MethodPost, "/falabella/sync/status/feed/any-feed/resolve", nil)
	resp, testErr = server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("POST /sync/status/feed/resolve status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}

	tracer.AssertStepCount(5)
}

// TestFalabellaSyncStatusStartWithoutSchedulerE2E verifies Start gracefully no-ops without scheduler.
func TestFalabellaSyncStatusStartWithoutSchedulerE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("start falabella mock server")
	mockServer := newFalabellaFeedStatusServer(t, nil)
	defer mockServer.Close()

	tracer.Step("initialize falabella module")
	module, err := falabella.New(falabella.Config{
		URL:                 mockServer.URL,
		UserID:              "e2e@test.com",
		APIKey:              "e2e-key-abc",
		RequestTimeoutMS:    2000,
		ValidationTimeoutMS: 1000,
		SyncStatusCron:      "*/5 * * * *",
	}, tracer.logger)
	if err != nil {
		t.Fatalf("falabella.New() error = %v", err)
	}

	tracer.Step("start module without scheduler (no-op)")
	if startErr := module.Start(context.Background()); startErr != nil {
		t.Fatalf("Start() error = %v", startErr)
	}

	tracer.Step("stop module (no-op)")
	if stopErr := module.Stop(context.Background()); stopErr != nil {
		t.Fatalf("Stop() error = %v", stopErr)
	}

	tracer.AssertStepCount(3)
}

// TestFalabellaSyncStatusInvalidConfigE2E verifies module degrades gracefully with invalid config.
func TestFalabellaSyncStatusInvalidConfigE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("initialize falabella module with invalid config")
	module, err := falabella.New(falabella.Config{
		URL:                 "",
		UserID:              "",
		APIKey:              "",
		RequestTimeoutMS:    2000,
		ValidationTimeoutMS: 1000,
	}, tracer.logger)
	if err != nil {
		t.Fatalf("falabella.New() error = %v (should degrade gracefully)", err)
	}

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8503}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	tracer.Step("verify brands endpoint returns 503 with invalid config")
	req, _ := http.NewRequest(http.MethodGet, "/falabella/brands", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /falabella/brands status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}

	tracer.AssertStepCount(3)
}
