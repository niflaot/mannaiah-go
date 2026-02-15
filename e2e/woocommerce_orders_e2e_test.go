package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	corecron "mannaiah/module/core/cron"
	"mannaiah/module/woocommerce"
)

// TestWooCommerceOrdersSyncE2E verifies WooCommerce order sync behavior including contact fallback and status updates.
func TestWooCommerceOrdersSyncE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("start woocommerce orders mock server")
	wooServer, setState := newWooOrdersSyncServer(t)
	defer wooServer.Close()

	harness.tracer.Step("initialize woocommerce scheduler")
	scheduler, err := corecron.NewScheduler(corecron.Config{Location: "UTC"}, harness.tracer.logger)
	if err != nil {
		t.Fatalf("corecron.NewScheduler() error = %v", err)
	}

	harness.tracer.Step("initialize woocommerce module with orders sync enabled")
	module, err := woocommerce.New(woocommerce.Config{
		URL:                 wooServer.URL,
		ConsumerKey:         "key",
		ConsumerSecret:      "secret",
		SyncContacts:        false,
		SyncOrders:          true,
		SyncOrdersCron:      "0 0 * * *",
		SyncPageSize:        10,
		SyncWorkers:         4,
		RequestTimeoutMS:    2000,
		ValidationTimeoutMS: 1000,
		VerifySSL:           true,
	}, harness.contactsModule.Service(), harness.ordersModule.Service(), scheduler, harness.tracer.logger)
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

	ordersManageToken := harness.SignToken(t, "orders:manage")
	ordersReadToken := harness.SignToken(t, "orders:read")
	contactsReadToken := harness.SignToken(t, "contacts:read")

	harness.tracer.Step("trigger manual woocommerce orders sync")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/woo/sync/orders", ordersManageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["created"] != float64(1) {
		t.Fatalf("payload.created = %v, want %v", payload["created"], float64(1))
	}

	harness.tracer.Step("verify contact was created by order sync fallback")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/contacts?page=1&limit=10&email=woo.order@example.com", contactsReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok || meta["total"] != float64(1) {
		t.Fatalf("contacts meta.total = %v, want %v", payload["meta"], float64(1))
	}

	harness.tracer.Step("verify order was created with mapped status and custom shipping")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/orders?page=1&limit=10&realm=woocommerce&identifier=1001", ordersReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	rows, ok := payload["data"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("orders payload.data = %v, want one row", payload["data"])
	}
	orderRow, _ := rows[0].(map[string]any)
	if orderRow["currentStatus"] != "CREATED" {
		t.Fatalf("order currentStatus = %v, want %q", orderRow["currentStatus"], "CREATED")
	}
	if orderRow["hasCustomShippingAddress"] != true {
		t.Fatalf("order hasCustomShippingAddress = %v, want %v", orderRow["hasCustomShippingAddress"], true)
	}
	orderID, _ := orderRow["id"].(string)
	if orderID == "" {
		t.Fatalf("expected order id")
	}

	harness.tracer.Step("update mock order source to completed with note comment")
	setState("completed", "Delivered by carrier")

	harness.tracer.Step("trigger manual woocommerce orders sync for update")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/woo/sync/orders", ordersManageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["updated"] != float64(1) {
		t.Fatalf("payload.updated = %v, want %v", payload["updated"], float64(1))
	}

	harness.tracer.Step("verify order status and synchronized order comments")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/orders/"+orderID, ordersReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["currentStatus"] != "COMPLETED" {
		t.Fatalf("order currentStatus = %v, want %q", payload["currentStatus"], "COMPLETED")
	}
	history, ok := payload["statusHistory"].([]any)
	if !ok || len(history) < 2 {
		t.Fatalf("statusHistory = %v, want at least 2 entries", payload["statusHistory"])
	}
	comments, ok := payload["comments"].([]any)
	if !ok || len(comments) < 1 {
		t.Fatalf("comments = %v, want at least one comment entry", payload["comments"])
	}
	if !containsOrderComment(comments, "system", "Delivered by carrier", false) {
		t.Fatalf("expected synchronized order comment entry")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(9)
}

// newWooOrdersSyncServer creates WooCommerce orders mock servers with mutable status and comment payload values.
func newWooOrdersSyncServer(t *testing.T) (*httptest.Server, func(status string, note string)) {
	t.Helper()

	stateMutex := sync.RWMutex{}
	currentStatus := "processing"
	currentNote := ""

	setState := func(status string, note string) {
		stateMutex.Lock()
		currentStatus = status
		currentNote = note
		stateMutex.Unlock()
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/wp-json/wc/v3/orders" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		stateMutex.RLock()
		status := currentStatus
		note := currentNote
		stateMutex.RUnlock()

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Wp-Total", "1")
		writer.Header().Set("X-Wp-Totalpages", "1")
		_ = json.NewEncoder(writer).Encode([]map[string]any{
			{
				"id":            1001,
				"status":        status,
				"date_created":  "2026-02-10T12:00:00Z",
				"date_modified": "2026-02-10T13:00:00Z",
				"customer_note": note,
				"billing": map[string]any{
					"email":      "woo.order@example.com",
					"first_name": "Woo",
					"last_name":  "Order",
					"phone":      "3001112233",
					"address_1":  "Billing 1",
					"address_2":  "Billing 2",
					"city":       "11001",
				},
				"shipping": map[string]any{
					"first_name": "Woo",
					"last_name":  "Order",
					"address_1":  "Shipping 1",
					"address_2":  "Shipping 2",
					"city":       "05001",
				},
				"line_items": []map[string]any{
					{
						"name":     "Woo Product",
						"sku":      "SKU-WOO-1",
						"quantity": 1,
						"meta_data": []map[string]any{
							{"key": "source", "value": "woocommerce"},
						},
					},
				},
			},
		})
	}))

	return server, setState
}

// containsOrderComment reports whether order-comment payload values contain matching values.
func containsOrderComment(values []any, author string, comment string, internal bool) bool {
	for _, raw := range values {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		authorValue, _ := row["author"].(string)
		commentValue, _ := row["comment"].(string)
		internalValue, _ := row["internal"].(bool)
		if authorValue == author && commentValue == comment && internalValue == internal {
			return true
		}
	}

	return false
}
