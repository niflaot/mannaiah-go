package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
)

// TestCategoriesLifecycleE2E verifies full category CRUD lifecycle end-to-end.
func TestCategoriesLifecycleE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	viewToken := harness.SignToken(t, "products:read")
	manageToken := harness.SignToken(t, "products:manage")

	harness.tracer.Step("create category without auth")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/categories", "", []byte(`{"slug":"electronics","name":"Electronics"}`))
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if payload["message"] != "unauthorized" {
		t.Fatalf("message = %v, want unauthorized", payload["message"])
	}

	harness.tracer.Step("create category with view scope (insufficient)")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/categories", viewToken, []byte(`{"slug":"electronics","name":"Electronics"}`))
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
	}
	if payload["message"] != "forbidden" {
		t.Fatalf("message = %v, want forbidden", payload["message"])
	}

	harness.tracer.Step("create category with manage scope")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/categories", manageToken, []byte(`{"slug":"electronics","name":"Electronics","description":"Electronic goods"}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	categoryID, _ := payload["id"].(string)
	if categoryID == "" {
		t.Fatalf("expected category id in create response, payload = %v", payload)
	}

	harness.tracer.Step("create duplicate slug")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/categories", manageToken, []byte(`{"slug":"electronics","name":"Electronics 2"}`))
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want %d", status, http.StatusConflict)
	}
	if payload["message"] != "category_slug_conflict" {
		t.Fatalf("message = %v, want category_slug_conflict", payload["message"])
	}

	harness.tracer.Step("get category tree with view scope")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/categories", viewToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	data, ok := payload["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data = %v, want one category", payload["data"])
	}

	harness.tracer.Step("get category by id")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/categories/"+categoryID, viewToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["slug"] != "electronics" {
		t.Fatalf("slug = %v, want electronics", payload["slug"])
	}

	harness.tracer.Step("create child category")
	childBody := fmt.Sprintf(`{"slug":"laptops","name":"Laptops","parentId":%q}`, categoryID)
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/categories", manageToken, []byte(childBody))
	if status != http.StatusCreated {
		t.Fatalf("create child status = %d, want %d", status, http.StatusCreated)
	}
	childID, _ := payload["id"].(string)
	if childID == "" {
		t.Fatalf("expected child category id")
	}

	harness.tracer.Step("list children of parent category")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/categories/"+categoryID+"/children", viewToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	children, ok := payload["data"].([]any)
	if !ok || len(children) != 1 {
		t.Fatalf("children = %v, want one child", payload["data"])
	}

	harness.tracer.Step("update category")
	status, payload = harness.DoJSONRequest(t, http.MethodPatch, "/categories/"+categoryID, manageToken, []byte(`{"name":"Electronics Updated","filterTags":["tech","gadget"]}`))
	if status != http.StatusOK {
		t.Fatalf("update status = %d, want %d", status, http.StatusOK)
	}
	if payload["name"] != "Electronics Updated" {
		t.Fatalf("name = %v, want Electronics Updated", payload["name"])
	}

	harness.tracer.Step("list products in empty category")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/categories/"+categoryID+"/products?page=1&pageSize=10", viewToken, nil)
	if status != http.StatusOK {
		t.Fatalf("products status = %d, want %d", status, http.StatusOK)
	}
	total, _ := payload["total"].(float64)
	if total != 0 {
		t.Fatalf("total = %v, want 0", total)
	}

	harness.tracer.Step("delete child first, then parent")
	status, _ = harness.DoJSONRequest(t, http.MethodDelete, "/categories/"+categoryID, manageToken, nil)
	if status != http.StatusConflict {
		t.Fatalf("delete parent with child status = %d, want %d", status, http.StatusConflict)
	}

	status, payload = harness.DoJSONRequest(t, http.MethodDelete, "/categories/"+childID, manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("delete child status = %d, want %d", status, http.StatusOK)
	}
	if payload["status"] != "deleted" {
		t.Fatalf("payload.status = %v, want deleted", payload["status"])
	}

	status, payload = harness.DoJSONRequest(t, http.MethodDelete, "/categories/"+categoryID, manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("delete parent status = %d, want %d", status, http.StatusOK)
	}
	if payload["status"] != "deleted" {
		t.Fatalf("payload.status = %v, want deleted", payload["status"])
	}

	harness.tracer.Step("get deleted category returns not found")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/categories/"+categoryID, viewToken, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	if payload["message"] != "category_not_found" {
		t.Fatalf("message = %v, want category_not_found", payload["message"])
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(13)
}
