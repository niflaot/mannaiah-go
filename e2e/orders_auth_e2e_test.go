package e2e_test

import (
	"net/http"
	"testing"
)

// TestOrdersAuthE2E verifies orders endpoints, auth checks, and normalized flow behavior end-to-end.
func TestOrdersAuthE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("create contact required by order flow")
	contactsManageToken := harness.SignToken(t, "contacts:manage")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/contacts", contactsManageToken, []byte(`{"email":"order-contact@example.com","legalName":"Order Contact","address":"Billing Street 9","addressExtra":"Apt 101","phone":"+573001112233","cityCode":"11001"}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	contactID, _ := payload["id"].(string)
	if contactID == "" {
		t.Fatalf("expected contact id")
	}

	harness.tracer.Step("request create order without authorization header")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/orders", "", []byte(`{"identifier":"woo-201","realm":"woocommerce","contactId":"`+contactID+`","items":[{"sku":"SKU-1","quantity":2}]}`))
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "unauthorized")
	}

	harness.tracer.Step("request create order with insufficient permissions")
	ordersReadToken := harness.SignToken(t, "orders:read")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/orders", ordersReadToken, []byte(`{"identifier":"woo-201","realm":"woocommerce","contactId":"`+contactID+`","items":[{"sku":"SKU-1","quantity":2}]}`))
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
	}
	if payload["message"] != "forbidden" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "forbidden")
	}

	harness.tracer.Step("create order with manage scope")
	ordersManageToken := harness.SignToken(t, "orders:manage")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/orders", ordersManageToken, []byte(`{"identifier":"woo-201","realm":"woocommerce","contactId":"`+contactID+`","items":[{"sku":"SKU-1","quantity":2},{"sku":"SKU-2","alternateName":"Fallback Name","quantity":1}],"author":"integration","description":"imported from woo"}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	orderID, _ := payload["id"].(string)
	if orderID == "" {
		t.Fatalf("expected order id")
	}
	if payload["hasCustomShippingAddress"] != false {
		t.Fatalf("payload.hasCustomShippingAddress = %v, want %v", payload["hasCustomShippingAddress"], false)
	}
	shipping, ok := payload["shippingAddress"].(map[string]any)
	if !ok {
		t.Fatalf("expected shippingAddress payload")
	}
	if shipping["address"] != "Billing Street 9" {
		t.Fatalf("shipping.address = %v, want %q", shipping["address"], "Billing Street 9")
	}

	harness.tracer.Step("request get order with read scope")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/orders/"+orderID, ordersReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["identifier"] != "woo-201" {
		t.Fatalf("payload.identifier = %v, want %q", payload["identifier"], "woo-201")
	}

	harness.tracer.Step("request list orders with filters")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/orders?page=1&limit=10&realm=woocommerce&status=CREATED", ordersReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	data, ok := payload["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("payload.data = %v, want one row", payload["data"])
	}

	harness.tracer.Step("update order status")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/orders/"+orderID+"/status", ordersManageToken, []byte(`{"status":"COMPLETED","author":"ops","description":"finalized"}`))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["currentStatus"] != "COMPLETED" {
		t.Fatalf("payload.currentStatus = %v, want %q", payload["currentStatus"], "COMPLETED")
	}

	harness.tracer.Step("request updated order by id")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/orders/"+orderID, ordersReadToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["currentStatus"] != "COMPLETED" {
		t.Fatalf("payload.currentStatus = %v, want %q", payload["currentStatus"], "COMPLETED")
	}

	harness.tracer.Step("append order comment")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/orders/"+orderID+"/comments", ordersManageToken, []byte(`{"author":"ops","comment":"first","internal":true}`))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	comments, ok := payload["comments"].([]any)
	if !ok || len(comments) == 0 {
		t.Fatalf("expected comments payload")
	}
	firstComment, ok := comments[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first comment payload")
	}
	commentID, _ := firstComment["id"].(string)
	if commentID == "" {
		t.Fatalf("expected comment id")
	}

	harness.tracer.Step("patch order comment")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/orders/"+orderID+"/comments/"+commentID, ordersManageToken, []byte(`{"comment":"updated","internal":false}`))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	comments, ok = payload["comments"].([]any)
	if !ok || len(comments) != 1 {
		t.Fatalf("expected one updated comment")
	}
	firstComment, ok = comments[0].(map[string]any)
	if !ok {
		t.Fatalf("expected updated comment payload")
	}
	if firstComment["comment"] != "updated" || firstComment["internal"] != false {
		t.Fatalf("updated comment payload = %v, want comment/internal updated", firstComment)
	}

	harness.tracer.Step("delete order comment")
	status, payload = harness.DoJSONRequest(t, http.MethodDelete, "/orders/"+orderID+"/comments/"+commentID, ordersManageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	comments, ok = payload["comments"].([]any)
	if !ok || len(comments) != 0 {
		t.Fatalf("expected empty comments after delete")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(12)
}
