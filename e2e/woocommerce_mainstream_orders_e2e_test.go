package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"mannaiah/module/core/messaging/bus"
	ordersport "mannaiah/module/orders/port"
	"mannaiah/module/woocommerce"
)

// TestWooCommerceMainstreamOrderUpdateE2E verifies event-driven mainstream order updates to WooCommerce and loop prevention behavior.
func TestWooCommerceMainstreamOrderUpdateE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)
	orderIdentifier := strconv.FormatInt(time.Now().UTC().UnixNano()%1000000+1000, 10)
	contactEmail := "mainstream.woo." + orderIdentifier + "@example.com"

	harness.tracer.Step("start woocommerce mainstream update mock server")
	updatePayloads := make(chan map[string]any, 4)
	updateCount := int32(0)
	wooServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/wp-json/wc/v3/orders":
			writer.Header().Set("Content-Type", "application/json")
			writer.Header().Set("X-Wp-Total", "0")
			writer.Header().Set("X-Wp-Totalpages", "0")
			_ = json.NewEncoder(writer).Encode([]map[string]any{})
			return
		case request.Method == http.MethodGet && request.URL.Path == "/wp-json/wc/v3/products":
			writer.Header().Set("Content-Type", "application/json")
			if strings.TrimSpace(request.URL.Query().Get("sku")) == "SKU-1" {
				_ = json.NewEncoder(writer).Encode([]map[string]any{{"id": 501}})
				return
			}
			_ = json.NewEncoder(writer).Encode([]map[string]any{})
			return
		case request.Method == http.MethodPut && strings.HasPrefix(request.URL.Path, "/wp-json/wc/v3/orders/"):
			atomic.AddInt32(&updateCount, 1)
			payload := map[string]any{}
			_ = json.NewDecoder(request.Body).Decode(&payload)
			updatePayloads <- payload
			writer.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(writer).Encode(map[string]any{"id": 1001})
			return
		default:
			writer.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer wooServer.Close()

	harness.tracer.Step("initialize woocommerce module with messaging registrar")
	wooModule, err := woocommerce.NewWithMessaging(
		woocommerce.Config{
			URL:            wooServer.URL,
			ConsumerKey:    "key",
			ConsumerSecret: "secret",
			SyncContacts:   false,
			SyncOrders:     false,
			VerifySSL:      true,
			RequestTimeoutMS: 300,
		},
		harness.contactsModule.Service(),
		harness.ordersModule.Service(),
		nil,
		harness.tracer.logger,
		harness.messaging.Registrar(),
	)
	if err != nil {
		t.Fatalf("woocommerce.NewWithMessaging() error = %v", err)
	}
	wooModule.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(wooModule.RegisterRoutes)
	if err := wooModule.Start(context.Background()); err != nil {
		t.Fatalf("wooModule.Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = wooModule.Stop(stopCtx)
	}()

	contactsManageToken := harness.SignToken(t, "contacts:manage")
	ordersManageToken := harness.SignToken(t, "orders:manage")

	harness.tracer.Step("create contact for order creation")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/contacts", contactsManageToken, []byte(`{
		"firstName":"Main",
		"lastName":"Stream",
		"email":"`+contactEmail+`"
	}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	contactID, _ := payload["id"].(string)
	if contactID == "" {
		t.Fatalf("expected created contact id")
	}

	harness.tracer.Step("create woo-realm order and trigger order created integration event")
	status, _ = harness.DoJSONRequest(t, http.MethodPost, "/orders", ordersManageToken, []byte(`{
		"identifier":"`+orderIdentifier+`",
		"realm":"woocommerce",
		"contactId":"`+contactID+`",
		"items":[{"sku":"SKU-1","quantity":2,"value":12000}],
		"shippingAddress":{"address":"Street 1","address2":"Apt 2","phone":"3001112233","cityCode":"11001"},
		"shippingCharges":[{"methodId":"flat_rate","methodTitle":"Flat Rate","price":8000}]
	}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}

	harness.tracer.Step("assert woo update payload was received")
	select {
	case updatePayload := <-updatePayloads:
		lineItems, _ := updatePayload["line_items"].([]any)
		if len(lineItems) != 1 {
			t.Fatalf("line_items = %v, want one row", updatePayload["line_items"])
		}
		shippingLines, _ := updatePayload["shipping_lines"].([]any)
		if len(shippingLines) != 1 {
			t.Fatalf("shipping_lines = %v, want one row", updatePayload["shipping_lines"])
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected woo update request")
	}

	harness.tracer.Step("publish woocommerce-origin order event and assert loop prevention")
	loopPayload, _ := json.Marshal(map[string]any{
		"id":         "order-1",
		"identifier": orderIdentifier,
		"realm":      "woocommerce",
		"source":     "woocommerce_sync",
		"items":      []map[string]any{{"sku": "SKU-1", "quantity": 1, "value": 1}},
	})
	if err := harness.messaging.Publisher().Publish(context.Background(), bus.Message{
		ID:      "loop-event-1",
		Topic:   ordersport.TopicOrderUpdated,
		Payload: loopPayload,
	}); err != nil {
		t.Fatalf("messaging.Publisher().Publish() error = %v", err)
	}
	time.Sleep(250 * time.Millisecond)
	if atomic.LoadInt32(&updateCount) != 1 {
		t.Fatalf("updateCount = %d, want %d", atomic.LoadInt32(&updateCount), 1)
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(6)
}
