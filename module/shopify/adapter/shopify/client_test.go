package shopify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	shopifyport "mannaiah/module/shopify/port"
)

type staticInstallationResolver struct {
	installation *shopifyport.Installation
}

func (r staticInstallationResolver) ResolveInstallation(ctx context.Context, shopDomain string) (*shopifyport.Installation, error) {
	_ = ctx
	if r.installation == nil {
		return nil, shopifyport.ErrInstallationNotFound
	}
	resolved := *r.installation
	if shopDomain != "" {
		resolved.ShopDomain = shopDomain
	}
	return &resolved, nil
}

func (r staticInstallationResolver) Refresh(ctx context.Context) error {
	_ = ctx
	return nil
}

// TestClientGetOrderRetries429 verifies Shopify 429 retry handling for targeted order fetches.
func TestClientGetOrderRetries429(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/orders/123.json" {
			t.Fatalf("request path = %q, want /orders/123.json", request.URL.Path)
		}
		if request.URL.Query().Get("status") != "any" {
			t.Fatalf("query status = %q, want any", request.URL.Query().Get("status"))
		}
		if atomic.AddInt32(&requests, 1) == 1 {
			writer.WriteHeader(http.StatusTooManyRequests)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"order":{"id":123,"name":"#123","email":"buyer@example.com","line_items":[],"shipping_lines":[],"discount_codes":[],"payment_gateway_names":[],"created_at":"2026-05-05T00:00:00Z"}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:                   server.URL,
		ClientID:                  "client-id",
		ClientSecret:              "client-secret",
		TooManyRequestsRetryDelay: 5 * time.Millisecond,
		TokenResolver: staticInstallationResolver{installation: &shopifyport.Installation{
			ShopDomain:  "flock-6591.myshopify.com",
			AccessToken: "token",
		}},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	order, err := client.GetOrder(shopifyport.WithShopDomain(context.Background(), "flock-6591.myshopify.com"), "123")
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if order.ID != "123" {
		t.Fatalf("GetOrder().ID = %q, want 123", order.ID)
	}
	if atomic.LoadInt32(&requests) != 2 {
		t.Fatalf("request count = %d, want 2", requests)
	}
}

// TestClientAdminRequestsAreRateLimited verifies shared Shopify Admin API pacing.
func TestClientAdminRequestsAreRateLimited(t *testing.T) {
	var mu sync.Mutex
	requestTimes := make([]time.Time, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()

		if request.URL.Path != "/customers.json" {
			t.Fatalf("request path = %q, want /customers.json", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"customers":[]}`))
	}))
	defer server.Close()

	interval := 25 * time.Millisecond
	client, err := NewClient(Config{
		BaseURL:                server.URL,
		ClientID:               "client-id",
		ClientSecret:           "client-secret",
		AdminRateLimitInterval: interval,
		TokenResolver: staticInstallationResolver{installation: &shopifyport.Installation{
			ShopDomain:  "flock-6591.myshopify.com",
			AccessToken: "token",
		}},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := shopifyport.WithShopDomain(context.Background(), "flock-6591.myshopify.com")
	if _, _, err := client.ListCustomers(ctx, "", 1); err != nil {
		t.Fatalf("ListCustomers() first error = %v", err)
	}
	if _, _, err := client.ListCustomers(ctx, "", 1); err != nil {
		t.Fatalf("ListCustomers() second error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requestTimes) != 2 {
		t.Fatalf("request count = %d, want 2", len(requestTimes))
	}
	if elapsed := requestTimes[1].Sub(requestTimes[0]); elapsed < interval-5*time.Millisecond {
		t.Fatalf("request interval = %s, want close to at least %s", elapsed, interval)
	}
}

// TestClientAdminRateLimitSerializesConcurrentRequests verifies concurrent callers share one paced lane.
func TestClientAdminRateLimitSerializesConcurrentRequests(t *testing.T) {
	var mu sync.Mutex
	requestTimes := make([]time.Time, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"customers":[]}`))
	}))
	defer server.Close()

	interval := 20 * time.Millisecond
	client, err := NewClient(Config{
		BaseURL:                server.URL,
		ClientID:               "client-id",
		ClientSecret:           "client-secret",
		AdminRateLimitInterval: interval,
		TokenResolver: staticInstallationResolver{installation: &shopifyport.Installation{
			ShopDomain:  "flock-6591.myshopify.com",
			AccessToken: "token",
		}},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := shopifyport.WithShopDomain(context.Background(), "flock-6591.myshopify.com")
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, listErr := client.ListCustomers(ctx, "", 1)
			errs <- listErr
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("ListCustomers() error = %v", err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requestTimes) != 3 {
		t.Fatalf("request count = %d, want 3", len(requestTimes))
	}
	for i := 1; i < len(requestTimes); i++ {
		if elapsed := requestTimes[i].Sub(requestTimes[i-1]); elapsed < interval-5*time.Millisecond {
			t.Fatalf("request interval %d = %s, want close to at least %s", i, elapsed, interval)
		}
	}
}

// TestClientFindCustomerByEmail verifies Shopify customer email search behavior.
func TestClientFindCustomerByEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/customers/search.json" {
			t.Fatalf("request path = %q, want /customers/search.json", request.URL.Path)
		}
		if got := request.URL.Query().Get("query"); got != "email:shop@example.com" {
			t.Fatalf("query = %q, want email:shop@example.com", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"customers":[{"id":123,"email":"shop@example.com","first_name":"Shop","last_name":"Customer"}]}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:      server.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenResolver: staticInstallationResolver{installation: &shopifyport.Installation{
			ShopDomain:  "flock-6591.myshopify.com",
			AccessToken: "token",
		}},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	customer, err := client.FindCustomerByEmail(shopifyport.WithShopDomain(context.Background(), "flock-6591.myshopify.com"), "shop@example.com")
	if err != nil {
		t.Fatalf("FindCustomerByEmail() error = %v", err)
	}
	if customer.ID != "123" {
		t.Fatalf("customer ID = %q, want 123", customer.ID)
	}
}

// TestClientExchangeAuthorizationCode verifies Shopify OAuth token exchange behavior.
func TestClientExchangeAuthorizationCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/admin/oauth/access_token" {
			t.Fatalf("request path = %q, want /admin/oauth/access_token", request.URL.Path)
		}
		var payload map[string]string
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode(request.Body) error = %v", err)
		}
		if payload["client_id"] != "client-id" || payload["client_secret"] != "client-secret" || payload["code"] != "code-123" {
			t.Fatalf("payload = %#v, want expected OAuth exchange values", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"access_token":"offline-token","scope":"read_orders,write_orders"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:      server.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenResolver: staticInstallationResolver{installation: &shopifyport.Installation{
			ShopDomain:  "flock-6591.myshopify.com",
			AccessToken: "unused",
		}},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	accessToken, scopes, err := client.ExchangeAuthorizationCode(context.Background(), "flock-6591.myshopify.com", "code-123")
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode() error = %v", err)
	}
	if accessToken != "offline-token" || scopes != "read_orders,write_orders" {
		t.Fatalf("ExchangeAuthorizationCode() = (%q, %q), want (offline-token, read_orders,write_orders)", accessToken, scopes)
	}
}
