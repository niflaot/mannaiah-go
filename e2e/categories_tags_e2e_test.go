package e2e_test

import (
	"net/http"
	"testing"
)

// TestCategoriesTagsAndPriceE2E verifies category tag filtering and price range behavior end-to-end.
func TestCategoriesTagsAndPriceE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	viewToken := harness.SignToken(t, "product:view")
	manageToken := harness.SignToken(t, "product:manage")
	assetCreateToken := harness.SignToken(t, "assets:create")
	productsManageToken := harness.SignToken(t, "products:manage")

	harness.tracer.Step("upload asset for product")
	assetStatus, assetPayload := doAssetUploadRequest(t, harness, assetCreateToken, "tag-test.png", []byte("image"), map[string]string{"name": "Tag Test"})
	if assetStatus != http.StatusCreated {
		t.Fatalf("asset status = %d, want %d", assetStatus, http.StatusCreated)
	}
	assetID, _ := assetPayload["_id"].(string)
	if assetID == "" {
		t.Fatalf("expected asset id")
	}

	harness.tracer.Step("create tagged product with price")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/products", productsManageToken, []byte(`{"sku":"TAGGED-1","price":49.99,"tags":["tech","sale"],"gallery":[{"assetId":"`+assetID+`","isMain":true}]}`))
	if status != http.StatusCreated {
		t.Fatalf("create product status = %d, want %d", status, http.StatusCreated)
	}
	productID, _ := payload["_id"].(string)
	if productID == "" {
		t.Fatalf("expected product id")
	}

	harness.tracer.Step("create category with pinned product")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/categories", manageToken, []byte(`{"slug":"sale-items","name":"Sale Items","productIds":["`+productID+`"]}`))
	if status != http.StatusCreated {
		t.Fatalf("create category status = %d, want %d", status, http.StatusCreated)
	}
	catID, _ := payload["id"].(string)
	if catID == "" {
		t.Fatalf("expected category id")
	}

	harness.tracer.Step("list products in category with pinned product")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/categories/"+catID+"/products?page=1&pageSize=10", viewToken, nil)
	if status != http.StatusOK {
		t.Fatalf("list products status = %d, want %d", status, http.StatusOK)
	}
	items, ok := payload["data"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %v, want one product", payload["data"])
	}

	harness.tracer.Step("create category with tag filter")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/categories", manageToken, []byte(`{"slug":"tech-items","name":"Tech Items","filterTags":["tech"]}`))
	if status != http.StatusCreated {
		t.Fatalf("create tag-filter category status = %d, want %d", status, http.StatusCreated)
	}
	tagCatID, _ := payload["id"].(string)
	if tagCatID == "" {
		t.Fatalf("expected tag category id")
	}

	harness.tracer.Step("list products in tag-filter category")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/categories/"+tagCatID+"/products?page=1&pageSize=10", viewToken, nil)
	if status != http.StatusOK {
		t.Fatalf("list products in tag-filter status = %d, want %d", status, http.StatusOK)
	}
	tagItems, ok := payload["data"].([]any)
	if !ok || len(tagItems) != 1 {
		t.Fatalf("tag items = %v, want one product", payload["data"])
	}

	harness.tracer.Step("create category with price range filter")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/categories", manageToken, []byte(`{"slug":"budget-items","name":"Budget Items","filterMinPrice":10.0,"filterMaxPrice":60.0}`))
	if status != http.StatusCreated {
		t.Fatalf("create price-filter category status = %d, want %d", status, http.StatusCreated)
	}
	priceCatID, _ := payload["id"].(string)
	if priceCatID == "" {
		t.Fatalf("expected price category id")
	}

	harness.tracer.Step("list products in price-range category")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/categories/"+priceCatID+"/products?page=1&pageSize=10", viewToken, nil)
	if status != http.StatusOK {
		t.Fatalf("list products in price-filter status = %d, want %d", status, http.StatusOK)
	}
	priceItems, ok := payload["data"].([]any)
	if !ok || len(priceItems) != 1 {
		t.Fatalf("price items = %v, want one product", payload["data"])
	}

	harness.tracer.Step("create parent category with includeChildren")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/categories", manageToken, []byte(`{"slug":"all-tech","name":"All Tech","includeChildren":true}`))
	if status != http.StatusCreated {
		t.Fatalf("create parent status = %d, want %d", status, http.StatusCreated)
	}
	parentCatID, _ := payload["id"].(string)
	if parentCatID == "" {
		t.Fatalf("expected parent category id")
	}

	harness.tracer.Step("update child category to use parent")
	updateBody := `{"parentId":"` + parentCatID + `"}`
	status, _ = harness.DoJSONRequest(t, http.MethodPatch, "/categories/"+tagCatID, manageToken, []byte(updateBody))
	if status != http.StatusOK {
		t.Fatalf("update child status = %d, want %d", status, http.StatusOK)
	}

	harness.tracer.Step("list children of parent")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/categories/"+parentCatID+"/children", viewToken, nil)
	if status != http.StatusOK {
		t.Fatalf("list children status = %d, want %d", status, http.StatusOK)
	}
	children, ok := payload["data"].([]any)
	if !ok || len(children) < 1 {
		t.Fatalf("children = %v, want at least one", payload["data"])
	}

	harness.tracer.Step("invalid category returns not found")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/categories/nonexistent-id", viewToken, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	if payload["message"] != "category_not_found" {
		t.Fatalf("message = %v, want category_not_found", payload["message"])
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(13)
}
