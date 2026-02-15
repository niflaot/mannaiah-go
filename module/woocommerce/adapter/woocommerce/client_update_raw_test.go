package woocommerce

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mannaiah/module/woocommerce/port"
)

// TestResolveWooProductIDBySKURawToleratesMetadata verifies tolerant product-id lookup behavior for non-scalar metadata.
func TestResolveWooProductIDBySKURawToleratesMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/wp-json/wc/v3/products" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if request.URL.Query().Get("sku") != "SKU-1" {
			t.Fatalf("sku query = %q, want %q", request.URL.Query().Get("sku"), "SKU-1")
		}
		if request.URL.Query().Get("_fields") != "id,sku" {
			t.Fatalf("_fields query = %q, want %q", request.URL.Query().Get("_fields"), "id,sku")
		}
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode([]map[string]any{
			{
				"id": 501,
				"meta_data": []map[string]any{
					{"key": "ts_time_views", "value": []int{1770578958, 1770578928}},
				},
			},
		})
	}))
	defer server.Close()

	client := &Client{
		baseURL:        server.URL,
		consumerKey:    "key",
		consumerSecret: "secret",
		timeout:        time.Second,
		verifySSL:      true,
	}

	productID, err := client.resolveWooProductIDBySKU(context.Background(), "SKU-1")
	if err != nil {
		t.Fatalf("resolveWooProductIDBySKU() error = %v", err)
	}
	if productID != 501 {
		t.Fatalf("resolveWooProductIDBySKU() = %d, want %d", productID, 501)
	}
}

// TestResolveWooProductIDBySKURawMissing verifies missing SKU lookup behavior.
func TestResolveWooProductIDBySKURawMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/wp-json/wc/v3/products" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode([]map[string]any{})
	}))
	defer server.Close()

	client := &Client{
		baseURL:        server.URL,
		consumerKey:    "key",
		consumerSecret: "secret",
		timeout:        time.Second,
		verifySSL:      true,
	}

	productID, err := client.resolveWooProductIDBySKU(context.Background(), "SKU-MISSING")
	if err != nil {
		t.Fatalf("resolveWooProductIDBySKU() error = %v", err)
	}
	if productID != 0 {
		t.Fatalf("resolveWooProductIDBySKU() = %d, want %d", productID, 0)
	}
}

// TestResolveOrderItemsForUpdateUsesRawSKUResolution verifies line-item mapping behavior with raw SKU resolution and fee fallback rows.
func TestResolveOrderItemsForUpdateUsesRawSKUResolution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/wp-json/wc/v3/products" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		sku := strings.TrimSpace(request.URL.Query().Get("sku"))
		writer.WriteHeader(http.StatusOK)
		if sku == "SKU-1" {
			_ = json.NewEncoder(writer).Encode([]map[string]any{{"id": 801}})
			return
		}
		_ = json.NewEncoder(writer).Encode([]map[string]any{})
	}))
	defer server.Close()

	client := &Client{
		baseURL:        server.URL,
		consumerKey:    "key",
		consumerSecret: "secret",
		timeout:        time.Second,
		verifySSL:      true,
	}

	lineItems, feeLines, err := client.resolveOrderItemsForUpdate(context.Background(), []port.OrderSyncItem{
		{SKU: "SKU-1", Quantity: 2, Value: 120000},
		{SKU: "SKU-MISSING", Quantity: 1, Value: 8000},
	})
	if err != nil {
		t.Fatalf("resolveOrderItemsForUpdate() error = %v", err)
	}
	if len(lineItems) != 1 || lineItems[0].ProductId != 801 || lineItems[0].SKU != "SKU-1" {
		t.Fatalf("lineItems = %+v, want one resolved SKU row", lineItems)
	}
	if len(feeLines) != 1 || feeLines[0].Name != "SKU-MISSING" || feeLines[0].Total != 8000 {
		t.Fatalf("feeLines = %+v, want one fallback fee row", feeLines)
	}
}
