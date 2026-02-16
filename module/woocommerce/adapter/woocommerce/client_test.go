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
				"id":            10,
				"status":        "processing",
				"date_created":  "2024-03-01T08:00:00Z",
				"date_modified": "2024-03-01T09:00:00Z",
				"billing": map[string]any{
					"email":      "john@example.com",
					"first_name": "John",
					"last_name":  "Doe",
					"phone":      "123",
					"address_1":  "Street 1",
					"address_2":  "Suite 1",
					"city":       "Bogota",
				},
				"shipping": map[string]any{
					"first_name": "John",
					"last_name":  "Doe",
					"address_1":  "Street 2",
					"address_2":  "Apt 1",
					"city":       "Medellin",
				},
				"line_items": []map[string]any{
					{
						"name":     "Product One",
						"sku":      "SKU-1",
						"quantity": 2,
						"meta_data": []map[string]any{
							{"key": "color", "value": "red"},
						},
					},
				},
				"fee_lines": []map[string]any{
					{
						"name":  "Cuotas",
						"total": "137000",
					},
				},
				"customer_note": "packed by warehouse",
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
	if orders[0].Status != "processing" {
		t.Fatalf("orders[0].Status = %q, want %q", orders[0].Status, "processing")
	}
	if orders[0].ShippingAddressLine1 != "Street 2" {
		t.Fatalf("orders[0].ShippingAddressLine1 = %q, want %q", orders[0].ShippingAddressLine1, "Street 2")
	}
	if len(orders[0].Items) != 2 || orders[0].Items[0].SKU != "SKU-1" || orders[0].Items[1].Name != "Cuotas" {
		t.Fatalf("orders[0].Items = %+v, want sku row + fee row", orders[0].Items)
	}
	if len(orders[0].Comments) != 1 || orders[0].Comments[0].Description != "packed by warehouse" {
		t.Fatalf("orders[0].Comments = %+v, want customer note comment", orders[0].Comments)
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

// TestSearchOrdersAndGetOrderByID verifies targeted Woo order retrieval behavior.
func TestSearchOrdersAndGetOrderByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/wp-json/wc/v3/orders":
			writer.Header().Set("X-Wp-Total", "1")
			writer.Header().Set("X-Wp-Totalpages", "1")
			writer.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(writer).Encode([]map[string]any{
				{
					"id":           7001,
					"status":       "processing",
					"date_created": "2024-03-01T08:00:00Z",
					"billing": map[string]any{
						"email":      "target@example.com",
						"first_name": "Target",
						"last_name":  "Person",
						"address_1":  "Street 1",
						"city":       "Bogota",
					},
					"line_items": []map[string]any{
						{"name": "Item 1", "sku": "SKU-1", "quantity": 1, "total": "12000"},
					},
				},
			})
		case "/wp-json/wc/v3/orders/7001":
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"id":            7001,
				"status":        "completed",
				"date_created":  "2024-03-01T08:00:00Z",
				"date_modified": "2024-03-01T09:00:00Z",
				"billing": map[string]any{
					"email":      "target@example.com",
					"first_name": "Target",
					"last_name":  "Person",
					"address_1":  "Street 1",
					"city":       "Bogota",
				},
				"line_items": []map[string]any{
					{"name": "Item 1", "sku": "SKU-1", "quantity": 1, "total": "12000"},
				},
			})
		default:
			writer.WriteHeader(http.StatusNotFound)
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

	orders, hasNext, searchErr := client.SearchOrders(context.Background(), "target@example.com", 1, 20)
	if searchErr != nil {
		t.Fatalf("SearchOrders() error = %v", searchErr)
	}
	if hasNext {
		t.Fatalf("expected hasNext=false for one-page search result")
	}
	if len(orders) != 1 || orders[0].ID != 7001 {
		t.Fatalf("SearchOrders() orders = %+v, want one id 7001", orders)
	}

	order, getErr := client.GetOrderByID(context.Background(), 7001)
	if getErr != nil {
		t.Fatalf("GetOrderByID() error = %v", getErr)
	}
	if order.ID != 7001 || order.Status != "completed" {
		t.Fatalf("GetOrderByID() order = %+v, want id 7001 status completed", order)
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
				"status":       "completed",
				"date_created": "2024-04-10T12:30:00Z",
				"billing": map[string]any{
					"email":      "array.meta@example.com",
					"first_name": "Array",
					"last_name":  "Meta",
				},
				"line_items": []map[string]any{
					{
						"name":     "Fallback Product",
						"sku":      "SKU-RAW-1",
						"quantity": "2",
						"total":    "129900.50",
					},
				},
				"shipping_lines": []map[string]any{
					{
						"method_id":    "flat_rate",
						"method_title": "Flat rate",
						"total":        "9900",
					},
				},
				"customer_note": "raw note",
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
	if orders[0].Status != "completed" {
		t.Fatalf("orders[0].Status = %q, want %q", orders[0].Status, "completed")
	}
	if len(orders[0].Items) != 1 || orders[0].Items[0].SKU != "SKU-RAW-1" {
		t.Fatalf("orders[0].Items = %+v, want one raw sku row", orders[0].Items)
	}
	if orders[0].Items[0].Quantity != 2 {
		t.Fatalf("orders[0].Items[0].Quantity = %d, want %d", orders[0].Items[0].Quantity, 2)
	}
	if orders[0].Items[0].Value != 129900.50 {
		t.Fatalf("orders[0].Items[0].Value = %f, want %f", orders[0].Items[0].Value, 129900.50)
	}
	if len(orders[0].ShippingCharges) != 1 || orders[0].ShippingCharges[0].Price != 9900 {
		t.Fatalf("orders[0].ShippingCharges = %+v, want one parsed shipping row", orders[0].ShippingCharges)
	}
	if len(orders[0].Comments) != 1 || orders[0].Comments[0].Description != "raw note" {
		t.Fatalf("orders[0].Comments = %+v, want raw note comment", orders[0].Comments)
	}
	if orders[0].Metadata["_full_payment_orders"] != "1021898" {
		t.Fatalf("metadata[_full_payment_orders] = %q, want %q", orders[0].Metadata["_full_payment_orders"], "1021898")
	}
	if orders[0].CreatedAt.UTC().Format(time.RFC3339) != "2024-04-10T12:30:00Z" {
		t.Fatalf("orders[0].CreatedAt = %v, want %q", orders[0].CreatedAt, "2024-04-10T12:30:00Z")
	}
}

// TestParseFlexibleNumber verifies tolerant JSON number parsing behavior.
func TestParseFlexibleNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{name: "number literal", input: "123.5", want: 123.5},
		{name: "quoted number", input: "\"456\"", want: 456},
		{name: "blank quoted", input: "\"\"", want: 0},
		{name: "null", input: "null", want: 0},
		{name: "invalid", input: "\"abc\"", wantErr: true},
	}

	for _, test := range tests {
		got, err := parseFlexibleNumber(test.input)
		if (err != nil) != test.wantErr {
			t.Fatalf("%s: parseFlexibleNumber(%q) error=%v, wantErr=%t", test.name, test.input, err, test.wantErr)
		}
		if test.wantErr {
			continue
		}
		if got != test.want {
			t.Fatalf("%s: parseFlexibleNumber(%q)=%f, want %f", test.name, test.input, got, test.want)
		}
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
