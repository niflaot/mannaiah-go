package woocommerce

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewClientValidation verifies constructor validation behavior.
func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient(Config{}); err != ErrInvalidURL {
		t.Fatalf("NewClient(empty) error = %v, want ErrInvalidURL", err)
	}
	if _, err := NewClient(Config{URL: "https://example.com"}); err != ErrInvalidConsumerKey {
		t.Fatalf("NewClient(missing key) error = %v, want ErrInvalidConsumerKey", err)
	}
	if _, err := NewClient(Config{URL: "https://example.com", ConsumerKey: "key"}); err != ErrInvalidConsumerSecret {
		t.Fatalf("NewClient(missing secret) error = %v, want ErrInvalidConsumerSecret", err)
	}
}

// TestValidateAndListOrders verifies WooCommerce SDK mapping behavior.
func TestValidateAndListOrders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/wp-json/wc/v3/orders" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.Header().Set("X-Wp-Total", "2")
		writer.Header().Set("X-Wp-Totalpages", "2")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode([]map[string]any{
			{
				"id": 10,
				"billing": map[string]any{
					"email":      "john@example.com",
					"first_name": "John",
					"last_name":  "Doe",
					"phone":      "123",
					"address_1":  "Street 1",
					"address_2":  "Suite 1",
					"city":       "Bogota",
				},
				"meta_data": []map[string]any{
					{"key": "_billing_document", "value": "998877"},
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(Config{
		URL:            server.URL,
		ConsumerKey:    "key",
		ConsumerSecret: "secret",
		Timeout:        time.Second,
		VerifySSL:      true,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := client.Validate(context.Background()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	orders, hasNext, err := client.ListOrders(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListOrders() error = %v", err)
	}
	if !hasNext {
		t.Fatalf("expected hasNext on first page")
	}
	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want %d", len(orders), 1)
	}
	if orders[0].BillingEmail != "john@example.com" {
		t.Fatalf("orders[0].BillingEmail = %q, want %q", orders[0].BillingEmail, "john@example.com")
	}
	if orders[0].Metadata["_billing_document"] != "998877" {
		t.Fatalf("orders[0].Metadata[_billing_document] = %q, want %q", orders[0].Metadata["_billing_document"], "998877")
	}
}

// TestListOrdersContextCancel verifies cancellation behavior.
func TestListOrdersContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode([]map[string]any{})
	}))
	defer server.Close()

	client, err := NewClient(Config{
		URL:            server.URL,
		ConsumerKey:    "key",
		ConsumerSecret: "secret",
		Timeout:        time.Second,
		VerifySSL:      true,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, _, listErr := client.ListOrders(ctx, 1, 10); listErr == nil {
		t.Fatalf("expected ListOrders() cancellation error")
	}
}

// TestValidateContextCancel verifies validation cancellation behavior.
func TestValidateContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode([]map[string]any{})
	}))
	defer server.Close()

	client, err := NewClient(Config{
		URL:            server.URL,
		ConsumerKey:    "key",
		ConsumerSecret: "secret",
		Timeout:        time.Second,
		VerifySSL:      true,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if validateErr := client.Validate(ctx); validateErr == nil {
		t.Fatalf("expected Validate() cancellation error")
	}
}
