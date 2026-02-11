package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
			writer.Header().Set("X-Wp-Total", "2")
			writer.Header().Set("X-Wp-Totalpages", "1")
			_ = json.NewEncoder(writer).Encode([]map[string]any{
				{
					"id": 1001,
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
					"id": 1002,
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
			writer.Header().Set("X-Wp-Total", "2")
			writer.Header().Set("X-Wp-Totalpages", "1")
			_ = json.NewEncoder(writer).Encode([]map[string]any{})
		}
	}))
}
