package e2e_test

import (
	"net/http"
	"testing"
)

// TestStorefrontRenderableE2E verifies storefront renderable and static-page lifecycle behavior end-to-end.
func TestStorefrontRenderableE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("request renderable create without authorization header")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/storefront/renderable", "", []byte(`{"kind":"static_page","metadata":{"title":"About"},"content":{"body":"hello"}}`))
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "unauthorized")
	}

	harness.tracer.Step("request renderable list with insufficient permissions")
	readToken := harness.SignToken(t, "product:view")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/storefront/renderable", readToken, nil)
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
	}
	if payload["message"] != "forbidden" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "forbidden")
	}

	manageToken := harness.SignToken(t, "storefront:manage")

	harness.tracer.Step("create renderable with storefront manage scope")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/storefront/renderable", manageToken, []byte(`{"kind":"static_page","metadata":{"title":"About"},"content":{"body":"hello"}}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	renderableID, _ := payload["id"].(string)
	if renderableID == "" {
		t.Fatalf("expected renderable id")
	}

	harness.tracer.Step("publish first renderable version")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/storefront/renderable/"+renderableID+"/publish", manageToken, nil)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	firstVersionID, _ := payload["id"].(string)
	if firstVersionID == "" {
		t.Fatalf("expected first version id")
	}

	harness.tracer.Step("update renderable draft")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/storefront/renderable/"+renderableID, manageToken, []byte(`{"metadata":{"title":"About Us"},"content":{"body":"updated"}}`))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["draft"] != true {
		t.Fatalf("payload.draft = %v, want %v", payload["draft"], true)
	}

	harness.tracer.Step("publish second renderable version")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/storefront/renderable/"+renderableID+"/publish", manageToken, nil)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}

	harness.tracer.Step("rollback first renderable version")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/storefront/renderable/"+renderableID+"/versions/"+firstVersionID+"/rollback", manageToken, nil)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	if payload["sourceVersionId"] != firstVersionID {
		t.Fatalf("payload.sourceVersionId = %v, want %q", payload["sourceVersionId"], firstVersionID)
	}

	harness.tracer.Step("get rolled back renderable")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/storefront/renderable/"+renderableID, manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	content, ok := payload["content"].(map[string]any)
	if !ok || content["body"] != "hello" {
		t.Fatalf("payload.content = %v, want rolled back content", payload["content"])
	}

	harness.tracer.Step("create static page bound to renderable")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/storefront/page", manageToken, []byte(`{"renderableId":"`+renderableID+`","title":"About","url":"/about","seoTags":{"robots":"index,follow","priority":0.8}}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	pageID, _ := payload["id"].(string)
	if pageID == "" {
		t.Fatalf("expected page id")
	}

	harness.tracer.Step("list static pages")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/storefront/page", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	data, ok := payload["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("payload.data = %v, want one page", payload["data"])
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(9)
}
