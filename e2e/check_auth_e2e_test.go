package e2e_test

import (
	"net/http"
	"testing"
)

// TestCheckAuthE2E verifies authentication-status endpoint behavior.
func TestCheckAuthE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("request check-auth without token")
	status, payload := harness.DoJSONRequest(t, http.MethodGet, "/check-auth", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "unauthorized")
	}

	harness.tracer.Step("request check-auth with malformed token")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/check-auth", "not-a-jwt", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "unauthorized")
	}

	harness.tracer.Step("request check-auth with valid token")
	token := harness.SignToken(t, "contact:view")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/check-auth", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["status"] != "authenticated" {
		t.Fatalf("payload.status = %v, want %q", payload["status"], "authenticated")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(4)
}
