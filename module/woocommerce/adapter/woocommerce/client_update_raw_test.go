package woocommerce

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	messagingplatform "mannaiah/module/core/messaging/platform"
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

// TestUpdateOrderFromMainstreamUsesRawPayload verifies WooCommerce raw update payload mapping behavior.
func TestUpdateOrderFromMainstreamUsesRawPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/wp-json/wc/v3/products":
			sku := strings.TrimSpace(request.URL.Query().Get("sku"))
			writer.WriteHeader(http.StatusOK)
			if sku == "SKU-1" {
				_ = json.NewEncoder(writer).Encode([]map[string]any{{"id": 901}})
				return
			}
			_ = json.NewEncoder(writer).Encode([]map[string]any{})
		case "/wp-json/wc/v3/orders/1023650":
			if request.Method == http.MethodGet {
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte(`{"line_items":[{"id":11,"sku":"SKU-1","product_id":901}],"fee_lines":[{"id":22,"name":"Quota"}],"shipping_lines":[{"id":33,"method_id":"flat_rate","method_title":"Flat Rate"}]}`))
				return
			}
			if request.Method == http.MethodPut {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Fatalf("ReadAll(request.Body) error = %v", err)
				}

				payload := map[string]any{}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("json.Unmarshal() error = %v", err)
				}

				lineItems, _ := payload["line_items"].([]any)
				if len(lineItems) != 1 {
					t.Fatalf("line_items length = %d, want %d", len(lineItems), 1)
				}
				firstLineItem, _ := lineItems[0].(map[string]any)
				if firstLineItem["id"].(float64) != 11 {
					t.Fatalf("line_items[0].id = %v, want %d", firstLineItem["id"], 11)
				}
				if firstLineItem["total"] != "120000.00" {
					t.Fatalf("line_items[0].total = %v, want %q", firstLineItem["total"], "120000.00")
				}

				feeLines, _ := payload["fee_lines"].([]any)
				if len(feeLines) != 1 {
					t.Fatalf("fee_lines length = %d, want %d", len(feeLines), 1)
				}
				firstFeeLine, _ := feeLines[0].(map[string]any)
				if firstFeeLine["id"].(float64) != 22 {
					t.Fatalf("fee_lines[0].id = %v, want %d", firstFeeLine["id"], 22)
				}

				shippingLines, _ := payload["shipping_lines"].([]any)
				if len(shippingLines) != 1 {
					t.Fatalf("shipping_lines length = %d, want %d", len(shippingLines), 1)
				}
				firstShippingLine, _ := shippingLines[0].(map[string]any)
				if firstShippingLine["id"].(float64) != 33 {
					t.Fatalf("shipping_lines[0].id = %v, want %d", firstShippingLine["id"], 33)
				}
				if firstShippingLine["total"] != "9000.00" {
					t.Fatalf("shipping_lines[0].total = %v, want %q", firstShippingLine["total"], "9000.00")
				}

				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte(`{"id":1023650}`))
				return
			}

			writer.WriteHeader(http.StatusMethodNotAllowed)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:        server.URL,
		consumerKey:    "key",
		consumerSecret: "secret",
		timeout:        time.Second,
		verifySSL:      true,
	}

	err := client.UpdateOrderFromMainstream(context.Background(), port.MainstreamOrderUpdateCommand{
		Identifier: "1023650",
		Items: []port.OrderSyncItem{
			{SKU: "SKU-1", Quantity: 2, Value: 120000},
			{Name: "Quota", Quantity: 1, Value: 15000},
		},
		ShippingCharges: []port.OrderSyncShippingCharge{
			{MethodID: "flat_rate", MethodTitle: "Flat Rate", Price: 9000},
		},
		ShippingAddress: &port.OrderSyncShippingAddress{
			Address:  "Street 1",
			Address2: "Apt 2",
			Phone:    "3001234567",
			CityCode: "11001",
		},
	})
	if err != nil {
		t.Fatalf("UpdateOrderFromMainstream() error = %v", err)
	}
}

// TestUpdateOrderFromMainstreamAvoidsLineItemDuplication verifies full-state fetch behavior prevents duplicate line-item appends.
func TestUpdateOrderFromMainstreamAvoidsLineItemDuplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/wp-json/wc/v3/products":
			sku := strings.TrimSpace(request.URL.Query().Get("sku"))
			writer.WriteHeader(http.StatusOK)
			if sku == "SKU-1" {
				_ = json.NewEncoder(writer).Encode([]map[string]any{{"id": 901}})
				return
			}
			_ = json.NewEncoder(writer).Encode([]map[string]any{})
		case "/wp-json/wc/v3/orders/1023650":
			if request.Method == http.MethodGet {
				if request.URL.Query().Get("_fields") != "" {
					writer.WriteHeader(http.StatusOK)
					_, _ = writer.Write([]byte(`{"line_items":[{"id":11}],"fee_lines":[],"shipping_lines":[]}`))
					return
				}
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte(`{"line_items":[{"id":11,"sku":"SKU-1","product_id":901}],"fee_lines":[],"shipping_lines":[]}`))
				return
			}
			if request.Method == http.MethodPut {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Fatalf("ReadAll(request.Body) error = %v", err)
				}

				payload := map[string]any{}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("json.Unmarshal() error = %v", err)
				}
				lineItems, _ := payload["line_items"].([]any)
				if len(lineItems) != 1 {
					t.Fatalf("line_items length = %d, want %d", len(lineItems), 1)
				}
				firstLineItem, _ := lineItems[0].(map[string]any)
				if firstLineItem["id"].(float64) != 11 {
					t.Fatalf("line_items[0].id = %v, want %d", firstLineItem["id"], 11)
				}
				if _, hasProductID := firstLineItem["product_id"]; hasProductID {
					t.Fatalf("line_items[0].product_id should be omitted when matching existing line id")
				}

				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte(`{"id":1023650}`))
				return
			}

			writer.WriteHeader(http.StatusMethodNotAllowed)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:        server.URL,
		consumerKey:    "key",
		consumerSecret: "secret",
		timeout:        time.Second,
		verifySSL:      true,
	}

	err := client.UpdateOrderFromMainstream(context.Background(), port.MainstreamOrderUpdateCommand{
		Identifier: "1023650",
		Items: []port.OrderSyncItem{
			{SKU: "SKU-1", Quantity: 1, Value: 139000},
		},
	})
	if err != nil {
		t.Fatalf("UpdateOrderFromMainstream() error = %v", err)
	}
}

// TestUpdateOrderFromMainstreamMarks4xxNonRetriable verifies non-retriable error classification for WooCommerce 4xx failures.
func TestUpdateOrderFromMainstreamMarks4xxNonRetriable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/wp-json/wc/v3/orders/1023650" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if request.Method == http.MethodGet {
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{"line_items":[],"fee_lines":[],"shipping_lines":[]}`))
			return
		}
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"code":"rest_invalid_param","message":"Parámetro(s) no válido(s): line_items, shipping_lines"}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:        server.URL,
		consumerKey:    "key",
		consumerSecret: "secret",
		timeout:        time.Second,
		verifySSL:      true,
	}

	err := client.UpdateOrderFromMainstream(context.Background(), port.MainstreamOrderUpdateCommand{
		Identifier: "1023650",
		ShippingCharges: []port.OrderSyncShippingCharge{
			{MethodID: "flat_rate", MethodTitle: "Flat Rate", Price: 9000},
		},
	})
	if err == nil {
		t.Fatalf("UpdateOrderFromMainstream() error = nil, want non-nil")
	}
	if !messagingplatform.IsNonRetriable(err) {
		t.Fatalf("expected non-retriable error, got %v", err)
	}
}
