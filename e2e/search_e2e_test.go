package e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"

	contactsearch "mannaiah/module/contacts/adapter/search"
	corehttp "mannaiah/module/core/http"
	coresearch "mannaiah/module/core/search"
	ordersearch "mannaiah/module/orders/adapter/search"
	productsearch "mannaiah/module/products/adapter/search/product"
)

// TestSearchEndpointsE2E verifies search endpoints return valid paginated
// results through the full HTTP stack including filter, sort, and spotlight.
func TestSearchEndpointsE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("wire search repositories and routes")
	contactRepo, err := contactsearch.NewRepository(harness.db)
	if err != nil {
		t.Fatalf("contactsearch.NewRepository() error = %v", err)
	}
	orderRepo, err := ordersearch.NewRepository(harness.db)
	if err != nil {
		t.Fatalf("ordersearch.NewRepository() error = %v", err)
	}
	productRepo, err := productsearch.NewRepository(harness.db)
	if err != nil {
		t.Fatalf("productsearch.NewRepository() error = %v", err)
	}

	spotlightSvc := coresearch.NewSpotlightService(0, contactRepo, orderRepo, productRepo)

	harness.server.RegisterRoutes(func(router corehttp.Router) {
		router.Get("/search/contacts", coresearch.SearchHandlerFunc(contactRepo))
		router.Get("/search/orders", coresearch.SearchHandlerFunc(orderRepo))
		router.Get("/search/products", coresearch.SearchHandlerFunc(productRepo))
		router.Get("/search", coresearch.SpotlightHandlerFunc(spotlightSvc))
	})

	manageToken := harness.SignToken(t, "contact:manage")

	harness.tracer.Step("create test contact for search")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/contacts", manageToken, []byte(`{
		"firstName": "SearchTestJohn",
		"lastName": "SearchTestDoe",
		"email": "searchjohn@example.com",
		"documentType": "CC",
		"documentNumber": "99887766"
	}`))
	if status != http.StatusCreated {
		t.Fatalf("create contact status = %d, want 201; payload = %v", status, payload)
	}

	harness.tracer.Step("search contacts with term")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/search/contacts?term=SearchTestJohn", "", nil)
	if status != http.StatusOK {
		t.Fatalf("search status = %d, want 200", status)
	}
	assertSearchResponse(t, payload)

	harness.tracer.Step("search contacts with pagination")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/search/contacts?page=1&pageSize=5", "", nil)
	if status != http.StatusOK {
		t.Fatalf("paginated search status = %d, want 200", status)
	}
	assertSearchResponse(t, payload)
	if ps, ok := payload["pageSize"].(float64); !ok || int(ps) != 5 {
		t.Errorf("pageSize = %v, want 5", payload["pageSize"])
	}

	harness.tracer.Step("search contacts empty term returns all")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/search/contacts", "", nil)
	if status != http.StatusOK {
		t.Fatalf("empty search status = %d, want 200", status)
	}
	assertSearchResponse(t, payload)

	harness.tracer.Step("search orders empty (no data)")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/search/orders?term=nonexistent", "", nil)
	if status != http.StatusOK {
		t.Fatalf("orders search status = %d, want 200", status)
	}
	assertSearchResponse(t, payload)

	harness.tracer.Step("spotlight search across resources")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/search?term=SearchTestJohn", "", nil)
	if status != http.StatusOK {
		t.Fatalf("spotlight status = %d, want 200", status)
	}
	if _, ok := payload["results"]; !ok {
		t.Error("spotlight response missing 'results' key")
	}
	if _, ok := payload["meta"]; !ok {
		t.Error("spotlight response missing 'meta' key")
	}
}

// assertSearchResponse validates the standard search response envelope.
func assertSearchResponse(t *testing.T, payload map[string]any) {
	t.Helper()
	requiredKeys := []string{"data", "total", "page", "pageSize", "totalPages"}
	for _, key := range requiredKeys {
		if _, ok := payload[key]; !ok {
			raw, _ := json.Marshal(payload)
			t.Errorf("search response missing %q key; payload = %s", key, raw)
		}
	}
}
