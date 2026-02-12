package e2e_test

import (
	"net/http"
	"testing"
)

// TestProductsAuthE2E verifies products CRUD and auth/permission behavior end-to-end.
func TestProductsAuthE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("request product create without authorization header")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/products", "", []byte(`{"sku":"SKU-1"}`))
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "unauthorized")
	}

	readToken := harness.SignToken(t, "products:read")
	manageToken := harness.SignToken(t, "products:manage")

	harness.tracer.Step("request product create with insufficient permissions")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/products", readToken, []byte(`{"sku":"SKU-1"}`))
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
	}
	if payload["message"] != "forbidden" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "forbidden")
	}

	harness.tracer.Step("create product with manage scope")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/products", manageToken, []byte(`{"sku":"SKU-1","gallery":[{"assetId":"asset-1","isMain":true}],"datasheets":[{"realm":"default","name":"Classic Tee","description":"Base"}],"variations":["var-red"],"variants":[{"variationIds":["var-red"]}]}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	productID, _ := payload["_id"].(string)
	if productID == "" {
		t.Fatalf("expected product _id in create response")
	}

	harness.tracer.Step("create duplicated sku")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/products", manageToken, []byte(`{"sku":"SKU-1"}`))
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want %d", status, http.StatusConflict)
	}
	if payload["message"] != "product_sku_conflict" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "product_sku_conflict")
	}

	harness.tracer.Step("list products with read scope")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/products", readToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	data, ok := payload["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("payload.data = %v, want one product", payload["data"])
	}

	harness.tracer.Step("get product with read scope")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/products/"+productID, readToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["sku"] != "SKU-1" {
		t.Fatalf("payload.sku = %v, want %q", payload["sku"], "SKU-1")
	}

	harness.tracer.Step("update product with manage scope")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/products/"+productID, manageToken, []byte(`{"datasheets":[{"realm":"default","name":"Classic Tee Updated"},{"realm":"b2b","name":"Bulk Tee"}]}`))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	datasheets, ok := payload["datasheets"].([]any)
	if !ok || len(datasheets) != 2 {
		t.Fatalf("payload.datasheets = %v, want merged datasheets", payload["datasheets"])
	}

	harness.tracer.Step("delete product with manage scope")
	status, payload = harness.DoJSONRequest(t, http.MethodDelete, "/products/"+productID, manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["status"] != "deleted" {
		t.Fatalf("payload.status = %v, want %q", payload["status"], "deleted")
	}

	harness.tracer.Step("request deleted product")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/products/"+productID, readToken, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	if payload["message"] != "product_not_found" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "product_not_found")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(10)
}
