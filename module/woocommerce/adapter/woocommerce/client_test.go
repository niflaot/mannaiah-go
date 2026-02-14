package woocommerce

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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
				"id":           10,
				"date_created": "2024-03-01T08:00:00Z",
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
	if orders[0].CreatedAt.UTC().Format(time.RFC3339) != "2024-03-01T08:00:00Z" {
		t.Fatalf("orders[0].CreatedAt = %v, want %q", orders[0].CreatedAt, "2024-03-01T08:00:00Z")
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

// TestListOrdersHasNextFallback verifies pagination continuation fallback when headers under-report total pages.
func TestListOrdersHasNextFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/wp-json/wc/v3/orders" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		page, _ := strconv.Atoi(request.URL.Query().Get("page"))
		if page == 0 {
			page = 1
		}
		perPage, _ := strconv.Atoi(request.URL.Query().Get("per_page"))
		if perPage == 0 {
			perPage = 100
		}

		writer.Header().Set("X-Wp-Total", "400")
		writer.Header().Set("X-Wp-Totalpages", "2")
		writer.WriteHeader(http.StatusOK)

		switch page {
		case 1, 2, 3, 4:
			items := make([]map[string]any, 0, perPage)
			for index := 0; index < perPage; index++ {
				items = append(items, map[string]any{
					"id": page*1000 + index,
					"billing": map[string]any{
						"email":      "user@example.com",
						"first_name": "First",
						"last_name":  "Last",
					},
				})
			}
			_ = json.NewEncoder(writer).Encode(items)
		default:
			_ = json.NewEncoder(writer).Encode([]map[string]any{})
		}
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

	if _, hasNext, listErr := client.ListOrders(context.Background(), 2, 100); listErr != nil {
		t.Fatalf("ListOrders(page=2) error = %v", listErr)
	} else if !hasNext {
		t.Fatalf("expected hasNext=true due full-page fallback")
	}

	if _, hasNext, listErr := client.ListOrders(context.Background(), 5, 100); listErr != nil {
		t.Fatalf("ListOrders(page=5) error = %v", listErr)
	} else if hasNext {
		t.Fatalf("expected hasNext=false for empty page")
	}
}

// TestResolveHasNextPage verifies helper pagination continuation behavior.
func TestResolveHasNextPage(t *testing.T) {
	if !resolveHasNextPage(1, 100, 20, 4, false) {
		t.Fatalf("expected hasNext=true when current page is below total pages")
	}
	if !resolveHasNextPage(2, 100, 100, 2, true) {
		t.Fatalf("expected hasNext=true for full-page fallback even on reported last page")
	}
	if resolveHasNextPage(3, 100, 0, 2, true) {
		t.Fatalf("expected hasNext=false on empty final page")
	}
}

// TestListOrdersMetadataArrayFallback verifies tolerant metadata decoding for non-scalar metadata values.
func TestListOrdersMetadataArrayFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/wp-json/wc/v3/orders" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		writer.Header().Set("X-Wp-Total", "1")
		writer.Header().Set("X-Wp-Totalpages", "1")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode([]map[string]any{
			{
				"id":           42,
				"date_created": "2024-04-10T12:30:00Z",
				"billing": map[string]any{
					"email":      "array.meta@example.com",
					"first_name": "Array",
					"last_name":  "Meta",
				},
				"meta_data": []map[string]any{
					{"key": "_full_payment_orders", "value": []int{1021898}},
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

	orders, hasNext, listErr := client.ListOrders(context.Background(), 1, 100)
	if listErr != nil {
		t.Fatalf("ListOrders() error = %v", listErr)
	}
	if hasNext {
		t.Fatalf("expected hasNext=false for single-page payload")
	}
	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want %d", len(orders), 1)
	}
	if orders[0].Metadata["_full_payment_orders"] != "1021898" {
		t.Fatalf("metadata[_full_payment_orders] = %q, want %q", orders[0].Metadata["_full_payment_orders"], "1021898")
	}
	if orders[0].CreatedAt.UTC().Format(time.RFC3339) != "2024-04-10T12:30:00Z" {
		t.Fatalf("orders[0].CreatedAt = %v, want %q", orders[0].CreatedAt, "2024-04-10T12:30:00Z")
	}
}

// TestShouldUseRawOrderFallback verifies fallback detection markers.
func TestShouldUseRawOrderFallback(t *testing.T) {
	err := errors.New("entity.Order.MetaData: []entity.Meta: entity.Meta.Value: fuzzyStringDecoder: not number or string")
	if !shouldUseRawOrderFallback(err) {
		t.Fatalf("expected shouldUseRawOrderFallback=true for fuzzy decoder failure")
	}
	if shouldUseRawOrderFallback(errors.New("401 unauthorized")) {
		t.Fatalf("expected shouldUseRawOrderFallback=false for auth errors")
	}
	if shouldUseRawOrderFallback(nil) {
		t.Fatalf("expected shouldUseRawOrderFallback=false for nil errors")
	}
}

// TestCompactError verifies error normalization and truncation behavior.
func TestCompactError(t *testing.T) {
	if compactError(nil, 10) != "" {
		t.Fatalf("expected empty value for nil error")
	}

	value := compactError(errors.New("  too   many \n spaces\t"), 0)
	if value != "too many spaces" {
		t.Fatalf("compactError() = %q, want %q", value, "too many spaces")
	}

	longErr := errors.New(strings.Repeat("x", 40))
	truncated := compactError(longErr, 10)
	if !strings.HasSuffix(truncated, "...") {
		t.Fatalf("expected truncated value to end with ellipsis, got %q", truncated)
	}
}

// TestParseWooOrderTime verifies WooCommerce order date parsing behavior.
func TestParseWooOrderTime(t *testing.T) {
	if !parseWooOrderTime("2024-03-01T08:00:00Z").Equal(time.Date(2024, time.March, 1, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("parseWooOrderTime(rfc3339) should parse UTC values")
	}
	if !parseWooOrderTime("2024-03-01 08:00:00").Equal(time.Date(2024, time.March, 1, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("parseWooOrderTime(sql-like) should parse fallback layouts")
	}
	if !parseWooOrderTime("2024-03-01T08:00:00-05:00").Equal(time.Date(2024, time.March, 1, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("parseWooOrderTime(offset) should normalize to UTC")
	}
	if !parseWooOrderTime(" ").IsZero() {
		t.Fatalf("parseWooOrderTime(blank) should return zero time")
	}
	if !parseWooOrderTime("invalid").IsZero() {
		t.Fatalf("parseWooOrderTime(invalid) should return zero time")
	}
}
