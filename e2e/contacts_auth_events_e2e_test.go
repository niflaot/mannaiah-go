package e2e_test

import (
	"fmt"
	"net/http"
	"testing"

	coremsgbus "mannaiah/module/core/messaging/bus"
)

// TestContactsAuthEventsE2E verifies contacts CRUD, auth/permission checks, and event publication/listening end-to-end.
func TestContactsAuthEventsE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("request create without authorization header")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/contacts", "", []byte(`{"email":"unauth@example.com","legalName":"NoAuth"}`))
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "unauthorized")
	}

	readToken := harness.SignToken(t, "contacts:read")
	manageToken := harness.SignToken(t, "contacts:manage")

	harness.tracer.Step("request create with insufficient permissions")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/contacts", readToken, []byte(`{"email":"forbidden@example.com","legalName":"Forbidden"}`))
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
	}
	if payload["message"] != "forbidden" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "forbidden")
	}

	harness.tracer.Step("create contact with manage scope")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/contacts", manageToken, []byte(`{"email":"john@example.com","legalName":"John Co"}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	contactID, _ := payload["id"].(string)
	if contactID == "" {
		t.Fatalf("expected contact id in create response")
	}

	harness.tracer.Step("assert created integration event")
	createdEvent := harness.AwaitCreatedEvent(t)
	if createdEvent.Payload["id"] != contactID {
		t.Fatalf("created event payload id = %v, want %q", createdEvent.Payload["id"], contactID)
	}
	if createdEvent.Metadata[coremsgbus.MetadataSchemaVersion] != "v1" {
		t.Fatalf("created event schema version = %q, want %q", createdEvent.Metadata[coremsgbus.MetadataSchemaVersion], "v1")
	}

	harness.tracer.Step("request get contact with read scope")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/contacts/"+contactID, readToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["email"] != "john@example.com" {
		t.Fatalf("payload.email = %v, want %q", payload["email"], "john@example.com")
	}

	harness.tracer.Step("update contact with manage scope")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/contacts/"+contactID, manageToken, []byte(`{"email":"john.updated@example.com"}`))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["email"] != "john.updated@example.com" {
		t.Fatalf("payload.email = %v, want %q", payload["email"], "john.updated@example.com")
	}

	harness.tracer.Step("assert updated integration event")
	updatedEvent := harness.AwaitUpdatedEvent(t)
	if updatedEvent.Payload["id"] != contactID {
		t.Fatalf("updated event payload id = %v, want %q", updatedEvent.Payload["id"], contactID)
	}
	if updatedEvent.Metadata[coremsgbus.MetadataSchemaVersion] != "v1" {
		t.Fatalf("updated event schema version = %q, want %q", updatedEvent.Metadata[coremsgbus.MetadataSchemaVersion], "v1")
	}

	harness.tracer.Step("list contacts excluding created id")
	listPath := fmt.Sprintf("/contacts?page=1&limit=10&excludeIds=%s", contactID)
	status, payload = harness.DoJSONRequest(t, http.MethodGet, listPath, readToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected list meta payload")
	}
	if meta["total"] != float64(0) {
		t.Fatalf("meta.total = %v, want %v", meta["total"], float64(0))
	}

	harness.tracer.Step("delete contact with manage scope")
	status, payload = harness.DoJSONRequest(t, http.MethodDelete, "/contacts/"+contactID, manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["status"] != "deleted" {
		t.Fatalf("payload.status = %v, want %q", payload["status"], "deleted")
	}

	harness.tracer.Step("request deleted contact")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/contacts/"+contactID, readToken, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	if payload["message"] != "contact_not_found" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "contact_not_found")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(10)
}
