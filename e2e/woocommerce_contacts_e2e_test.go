package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	corecron "mannaiah/module/core/cron"
	"mannaiah/module/woocommerce"
	wooevent "mannaiah/module/woocommerce/adapter/event"
)

// TestWooCommerceContactsSyncE2E verifies manual WooCommerce contact sync endpoint behavior.
func TestWooCommerceContactsSyncE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("start woocommerce mock server")
	wooServer := newWooOrdersServer(t)
	defer wooServer.Close()

	harness.tracer.Step("initialize woocommerce scheduler")
	scheduler, err := corecron.NewScheduler(corecron.Config{Location: "UTC"}, harness.tracer.logger)
	if err != nil {
		t.Fatalf("corecron.NewScheduler() error = %v", err)
	}

	harness.tracer.Step("initialize woocommerce event publisher")
	publisher, err := wooevent.NewPublisher(harness.messaging.Publisher())
	if err != nil {
		t.Fatalf("wooevent.NewPublisher() error = %v", err)
	}

	harness.tracer.Step("initialize woocommerce module")
	module, err := woocommerce.New(woocommerce.Config{
		URL:                 wooServer.URL,
		ConsumerKey:         "key",
		ConsumerSecret:      "secret",
		SyncContacts:        true,
		SyncContactsCron:    "0 0 * * *",
		SyncPageSize:        2,
		SyncWorkers:         4,
		RequestTimeoutMS:    2000,
		ValidationTimeoutMS: 1000,
		VerifySSL:           true,
	}, harness.contactsModule.Service(), scheduler, harness.tracer.logger, publisher)
	if err != nil {
		t.Fatalf("woocommerce.New() error = %v", err)
	}
	module.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(module.RegisterRoutes)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("module.Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = module.Stop(stopCtx)
	}()

	manageToken := harness.SignToken(t, "contacts:manage")
	readToken := harness.SignToken(t, "contacts:read")

	harness.tracer.Step("trigger manual woocommerce contacts sync")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/woo/sync/contacts", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["processed"] != float64(2) {
		t.Fatalf("payload.processed = %v, want %v", payload["processed"], float64(2))
	}

	harness.tracer.Step("assert contacts created events from sync")
	harness.AwaitCreatedEvent(t)
	harness.AwaitCreatedEvent(t)

	harness.tracer.Step("verify contacts were persisted")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/contacts?page=1&limit=10", readToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected contacts meta payload")
	}
	if meta["total"] != float64(2) {
		t.Fatalf("meta.total = %v, want %v", meta["total"], float64(2))
	}
	data, ok := payload["data"].([]any)
	if !ok {
		t.Fatalf("expected contacts data payload")
	}
	contact := findContactByEmail(t, data, "woo.one@example.com")
	createdAt, _ := contact["createdAt"].(string)
	if createdAt != "2024-03-01T08:00:00Z" {
		t.Fatalf("woo.one createdAt = %q, want %q", createdAt, "2024-03-01T08:00:00Z")
	}
	metadata, ok := contact["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected woo.one metadata payload")
	}
	if metadata["integration.source"] != "woocommerce" {
		t.Fatalf("woo.one metadata integration.source = %v, want %q", metadata["integration.source"], "woocommerce")
	}
	if metadata["integration.woocommerce.oldest_order_id"] != "1001" {
		t.Fatalf("woo.one metadata oldest_order_id = %v, want %q", metadata["integration.woocommerce.oldest_order_id"], "1001")
	}
	if metadata["integration.woocommerce.oldest_order_created_at"] != "2024-03-01T08:00:00Z" {
		t.Fatalf("woo.one metadata oldest_order_created_at = %v, want %q", metadata["integration.woocommerce.oldest_order_created_at"], "2024-03-01T08:00:00Z")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(8)
}

// TestWooCommerceInvalidIntegrationE2E verifies disabled endpoint behavior for invalid integrations.
func TestWooCommerceInvalidIntegrationE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("initialize woocommerce scheduler")
	scheduler, err := corecron.NewScheduler(corecron.Config{Location: "UTC"}, harness.tracer.logger)
	if err != nil {
		t.Fatalf("corecron.NewScheduler() error = %v", err)
	}

	harness.tracer.Step("initialize woocommerce module with invalid integration config")
	module, err := woocommerce.New(woocommerce.Config{
		SyncContacts:     true,
		SyncContactsCron: "0 0 * * *",
	}, harness.contactsModule.Service(), scheduler, harness.tracer.logger)
	if err != nil {
		t.Fatalf("woocommerce.New() error = %v", err)
	}
	module.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(module.RegisterRoutes)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("module.Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = module.Stop(stopCtx)
	}()

	manageToken := harness.SignToken(t, "contacts:manage")

	harness.tracer.Step("trigger manual woocommerce contacts sync with invalid integration")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/woo/sync/contacts", manageToken, nil)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", status, http.StatusServiceUnavailable)
	}
	if payload["message"] != "woocommerce_integration_unavailable" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "woocommerce_integration_unavailable")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(4)
}

// TestWooCommerceSyncDisabledE2E verifies controlled disabled behavior when sync is turned off.
func TestWooCommerceSyncDisabledE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("initialize woocommerce module with sync disabled")
	module, err := woocommerce.New(woocommerce.Config{
		SyncContacts: false,
	}, harness.contactsModule.Service(), nil, harness.tracer.logger)
	if err != nil {
		t.Fatalf("woocommerce.New() error = %v", err)
	}
	module.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(module.RegisterRoutes)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("module.Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = module.Stop(stopCtx)
	}()

	manageToken := harness.SignToken(t, "contacts:manage")

	harness.tracer.Step("trigger manual woocommerce contacts sync while disabled")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/woo/sync/contacts", manageToken, nil)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", status, http.StatusServiceUnavailable)
	}
	if payload["message"] != "woocommerce_contacts_sync_disabled" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "woocommerce_contacts_sync_disabled")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(3)
}

// TestWooCommerceOutageCircuitBreakerE2E verifies fail-fast behavior under repeated WooCommerce outages.
func TestWooCommerceOutageCircuitBreakerE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("start failing woocommerce mock server")
	wooServer, requestCount := newFailingWooOrdersServer(t)
	defer wooServer.Close()

	harness.tracer.Step("initialize woocommerce scheduler")
	scheduler, err := corecron.NewScheduler(corecron.Config{Location: "UTC"}, harness.tracer.logger)
	if err != nil {
		t.Fatalf("corecron.NewScheduler() error = %v", err)
	}

	harness.tracer.Step("initialize woocommerce module with source circuit breaker")
	module, err := woocommerce.New(woocommerce.Config{
		URL:                            wooServer.URL,
		ConsumerKey:                    "key",
		ConsumerSecret:                 "secret",
		SyncContacts:                   true,
		SyncContactsCron:               "0 0 * * *",
		SyncPageSize:                   2,
		SyncWorkers:                    4,
		RequestTimeoutMS:               1500,
		ValidationTimeoutMS:            1000,
		VerifySSL:                      true,
		CircuitBreakerEnabled:          true,
		CircuitBreakerFailureThreshold: 1,
		CircuitBreakerTimeoutMS:        120000,
		CircuitBreakerIntervalMS:       120000,
	}, harness.contactsModule.Service(), scheduler, harness.tracer.logger)
	if err != nil {
		t.Fatalf("woocommerce.New() error = %v", err)
	}
	module.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(module.RegisterRoutes)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("module.Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = module.Stop(stopCtx)
	}()

	manageToken := harness.SignToken(t, "contacts:manage")

	harness.tracer.Step("trigger repeated manual sync requests during outage")
	for attempt := 0; attempt < 3; attempt++ {
		status, payload := harness.DoJSONRequest(t, http.MethodPost, "/woo/sync/contacts", manageToken, nil)
		if status != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", status, http.StatusServiceUnavailable)
		}
		if payload["message"] != "woocommerce_integration_unavailable" {
			t.Fatalf("payload.message = %v, want %q", payload["message"], "woocommerce_integration_unavailable")
		}
	}

	if requestCount.Load() != 1 {
		t.Fatalf("woocommerce request count = %d, want %d after breaker opens", requestCount.Load(), 1)
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(5)
}

// TestWooCommerceSyncPageFailureNoPartialWritesE2E verifies that listing failures do not persist partial sync writes.
func TestWooCommerceSyncPageFailureNoPartialWritesE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("start woocommerce page-failure mock server")
	wooServer := newWooOrdersPageFailureServer(t)
	defer wooServer.Close()

	harness.tracer.Step("initialize woocommerce scheduler")
	scheduler, err := corecron.NewScheduler(corecron.Config{Location: "UTC"}, harness.tracer.logger)
	if err != nil {
		t.Fatalf("corecron.NewScheduler() error = %v", err)
	}

	harness.tracer.Step("initialize woocommerce module")
	module, err := woocommerce.New(woocommerce.Config{
		URL:                 wooServer.URL,
		ConsumerKey:         "key",
		ConsumerSecret:      "secret",
		SyncContacts:        true,
		SyncContactsCron:    "0 0 * * *",
		SyncPageSize:        2,
		SyncWorkers:         4,
		RequestTimeoutMS:    2000,
		ValidationTimeoutMS: 1000,
		VerifySSL:           true,
	}, harness.contactsModule.Service(), scheduler, harness.tracer.logger)
	if err != nil {
		t.Fatalf("woocommerce.New() error = %v", err)
	}
	module.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(module.RegisterRoutes)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("module.Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = module.Stop(stopCtx)
	}()

	manageToken := harness.SignToken(t, "contacts:manage")
	readToken := harness.SignToken(t, "contacts:read")

	harness.tracer.Step("trigger manual woocommerce contacts sync expecting page failure")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/woo/sync/contacts", manageToken, nil)
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", status, http.StatusInternalServerError)
	}
	if payload["message"] != "internal_server_error" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "internal_server_error")
	}

	harness.tracer.Step("verify contacts were not partially persisted")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/contacts?page=1&limit=10", readToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected contacts meta payload")
	}
	if meta["total"] != float64(0) {
		t.Fatalf("meta.total = %v, want %v", meta["total"], float64(0))
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(6)
}

// newWooOrdersServer creates a WooCommerce-compatible orders listing test server.
func newWooOrdersServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.HasPrefix(request.URL.Path, "/wp-json/wc/v3/orders") {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		page := request.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}

		writer.Header().Set("Content-Type", "application/json")
		switch page {
		case "1":
			writer.Header().Set("X-Wp-Total", "3")
			writer.Header().Set("X-Wp-Totalpages", "2")
			_ = json.NewEncoder(writer).Encode([]map[string]any{
				{
					"id":           1002,
					"date_created": "2024-03-03T10:00:00Z",
					"billing": map[string]any{
						"email":      "woo.one@example.com",
						"first_name": "Woo",
						"last_name":  "One",
						"phone":      "111",
						"address_1":  "Street 1",
						"address_2":  "Suite 1",
						"city":       "Bogota",
					},
					"meta_data": []map[string]any{
						{"key": "_billing_document", "value": "12345"},
					},
				},
				{
					"id":           1001,
					"date_created": "2024-03-01T08:00:00Z",
					"billing": map[string]any{
						"email":      "woo.one@example.com",
						"first_name": "Woo",
						"last_name":  "One",
						"phone":      "111",
						"address_1":  "Street 1",
						"address_2":  "Suite 1",
						"city":       "Bogota",
					},
					"meta_data": []map[string]any{
						{"key": "_billing_document", "value": "12345"},
					},
				},
			})
		case "2":
			writer.Header().Set("X-Wp-Total", "3")
			writer.Header().Set("X-Wp-Totalpages", "2")
			_ = json.NewEncoder(writer).Encode([]map[string]any{
				{
					"id":           1003,
					"date_created": "2024-03-02T09:30:00Z",
					"billing": map[string]any{
						"email":      "woo.two@example.com",
						"first_name": "Woo",
						"last_name":  "Two",
						"phone":      "222",
						"address_1":  "Street 2",
						"address_2":  "Suite 2",
						"city":       "Medellin",
					},
				},
			})
		default:
			writer.Header().Set("X-Wp-Total", "3")
			writer.Header().Set("X-Wp-Totalpages", "2")
			_ = json.NewEncoder(writer).Encode([]map[string]any{})
		}
	}))
}

// newFailingWooOrdersServer creates a WooCommerce-compatible server that always returns 500 for orders.
func newFailingWooOrdersServer(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()

	requestCount := &atomic.Int64{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.HasPrefix(request.URL.Path, "/wp-json/wc/v3/orders") {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		requestCount.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"code":    "woocommerce_error",
			"message": "upstream unavailable",
		})
	}))

	return server, requestCount
}

// newWooOrdersPageFailureServer creates a WooCommerce-compatible server that fails when listing a later page.
func newWooOrdersPageFailureServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.HasPrefix(request.URL.Path, "/wp-json/wc/v3/orders") {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		page := request.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Wp-Total", "4")
		writer.Header().Set("X-Wp-Totalpages", "2")

		switch page {
		case "1":
			writer.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(writer).Encode([]map[string]any{
				{
					"id": 2001,
					"billing": map[string]any{
						"email":      "staged.one@example.com",
						"first_name": "Staged",
						"last_name":  "One",
					},
				},
				{
					"id": 2002,
					"billing": map[string]any{
						"email":      "staged.two@example.com",
						"first_name": "Staged",
						"last_name":  "Two",
					},
				},
			})
		default:
			writer.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"code":    "woocommerce_error",
				"message": "page failure",
			})
		}
	}))
}

// findContactByEmail resolves a contact row from list payload values by email.
func findContactByEmail(t *testing.T, rows []any, email string) map[string]any {
	t.Helper()

	for _, row := range rows {
		typed, ok := row.(map[string]any)
		if !ok {
			continue
		}
		if typedEmail, _ := typed["email"].(string); typedEmail == email {
			return typed
		}
	}

	t.Fatalf("contact with email %q not found in payload", email)
	return nil
}
