package e2e_test

import (
	"net/http"
	"testing"
)

// TestVariationsAuthE2E verifies variation CRUD and auth/permission behavior end-to-end.
func TestVariationsAuthE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("request variation create without authorization header")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/variations", "", []byte(`{"name":"Red","definition":"COLOR","value":"#FF0000"}`))
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "unauthorized")
	}

	readToken := harness.SignToken(t, "variations:read")
	createToken := harness.SignToken(t, "variations:create")
	updateToken := harness.SignToken(t, "variations:update")
	deleteToken := harness.SignToken(t, "variations:delete")

	harness.tracer.Step("request variation create with insufficient permissions")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/variations", readToken, []byte(`{"name":"Red","definition":"COLOR","value":"#FF0000"}`))
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
	}
	if payload["message"] != "forbidden" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "forbidden")
	}

	harness.tracer.Step("create variation with create scope")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/variations", createToken, []byte(`{"name":"Red","definition":"COLOR","value":"#FF0000"}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	variationID, _ := payload["_id"].(string)
	if variationID == "" {
		t.Fatalf("expected variation _id in create response")
	}

	harness.tracer.Step("list variations with read scope")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/variations", readToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	items, ok := payload["data"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("payload.data = %v, want non-empty list", payload["data"])
	}

	harness.tracer.Step("get variation with read scope")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/variations/"+variationID, readToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["definition"] != "COLOR" {
		t.Fatalf("payload.definition = %v, want %q", payload["definition"], "COLOR")
	}

	harness.tracer.Step("update variation with update scope")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/variations/"+variationID, updateToken, []byte(`{"name":"Dark Red","definition":"SIZE","value":"#8B0000"}`))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["name"] != "Dark Red" {
		t.Fatalf("payload.name = %v, want %q", payload["name"], "Dark Red")
	}
	if payload["definition"] != "COLOR" {
		t.Fatalf("payload.definition = %v, want immutable %q", payload["definition"], "COLOR")
	}

	harness.tracer.Step("delete variation with delete scope")
	status, payload = harness.DoJSONRequest(t, http.MethodDelete, "/variations/"+variationID, deleteToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["status"] != "deleted" {
		t.Fatalf("payload.status = %v, want %q", payload["status"], "deleted")
	}

	harness.tracer.Step("request deleted variation")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/variations/"+variationID, readToken, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	if payload["message"] != "variation_not_found" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "variation_not_found")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(9)
}
