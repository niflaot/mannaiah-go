package shopify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
	shopifyport "mannaiah/module/shopify/port"
)

// TestClientGetOrderRetries429 verifies Shopify 429 retry handling for targeted order fetches.
func TestClientGetOrderRetries429(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/orders/123.json" {
			t.Fatalf("request path = %q, want /orders/123.json", request.URL.Path)
		}
		if request.URL.Query().Get("status") != "any" {
			t.Fatalf("query status = %q, want any", request.URL.Query().Get("status"))
		}
		if atomic.AddInt32(&requests, 1) == 1 {
			writer.WriteHeader(http.StatusTooManyRequests)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"order":{"id":123,"name":"#123","email":"buyer@example.com","line_items":[],"shipping_lines":[],"discount_codes":[],"payment_gateway_names":[],"created_at":"2026-05-05T00:00:00Z"}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, AccessToken: "token", Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	order, err := client.GetOrder(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if order.ID != "123" {
		t.Fatalf("GetOrder().ID = %q, want 123", order.ID)
	}
	if atomic.LoadInt32(&requests) != 2 {
		t.Fatalf("request count = %d, want 2", requests)
	}
}

// TestClientUpdateOrderFromMainstreamAddsCompletedTag verifies outbound note/tag updates.
func TestClientUpdateOrderFromMainstreamAddsCompletedTag(t *testing.T) {
	var putBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"order":{"id":321,"name":"#321","email":"buyer@example.com","note":"existing note","tags":"vip","line_items":[],"shipping_lines":[],"discount_codes":[],"payment_gateway_names":[],"created_at":"2026-05-05T00:00:00Z"}}`))
		case http.MethodPut:
			var err error
			putBody, err = io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("ReadAll(request.Body) error = %v", err)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"order":{"id":321}}`))
		default:
			t.Fatalf("request method = %q, want GET or PUT", request.Method)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, AccessToken: "token", Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := client.UpdateOrderFromMainstream(context.Background(), "321", shopifyport.MainstreamOrderUpdateCommand{Status: ordersdomain.StatusCompleted}); err != nil {
		t.Fatalf("UpdateOrderFromMainstream() error = %v", err)
	}

	var requestBody struct {
		Order struct {
			Note string `json:"note"`
			Tags string `json:"tags"`
		} `json:"order"`
	}
	if err := json.Unmarshal(putBody, &requestBody); err != nil {
		t.Fatalf("json.Unmarshal(putBody) error = %v", err)
	}
	if !strings.Contains(requestBody.Order.Note, "existing note") {
		t.Fatalf("updated note = %q, want existing note retained", requestBody.Order.Note)
	}
	if !strings.Contains(requestBody.Order.Note, "[Mannaiah] Order marked as completed") {
		t.Fatalf("updated note = %q, want completed note", requestBody.Order.Note)
	}
	if !strings.Contains(requestBody.Order.Tags, "vip") {
		t.Fatalf("updated tags = %q, want existing tag retained", requestBody.Order.Tags)
	}
	if !strings.Contains(requestBody.Order.Tags, "mannaiah:completed") {
		t.Fatalf("updated tags = %q, want completion tag", requestBody.Order.Tags)
	}
}
