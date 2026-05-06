package shopify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
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

// TestClientUpdateOrderFromMainstreamAddsCompletedTag verifies outbound note/tag updates.
func TestClientUpdateOrderFromMainstreamAddsCompletedTag(t *testing.T) {
	var putBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"order":{"id":321,"name":"#321","email":"buyer@example.com","note":"existing note","tags":"vip","line_items":[],"shipping_lines":[],"discount_codes":[],"payment_gateway_names":[],"created_at":"2026-05-05T00:00:00Z"}}`))
		case http.MethodPut:
			var err error
			putBody, err = io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("ReadAll(request.Body) error = %v", err)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"order":{"id":321}}`))
		default:
			t.Fatalf("request method = %q, want GET or PUT", request.Method)
		}
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

	if err := client.UpdateOrderFromMainstream(shopifyport.WithShopDomain(context.Background(), "flock-6591.myshopify.com"), "321", shopifyport.MainstreamOrderUpdateCommand{Status: ordersdomain.StatusCompleted}); err != nil {
		t.Fatalf("UpdateOrderFromMainstream() error = %v", err)
	}

	var requestBody struct {
		Order struct {
			Note string `json:"note"`
			Tags string `json:"tags"`
		} `json:"order"`
	}
	if err := json.Unmarshal(putBody, &requestBody); err != nil {
		t.Fatalf("json.Unmarshal(putBody) error = %v", err)
	}
	if !strings.Contains(requestBody.Order.Note, "existing note") {
		t.Fatalf("updated note = %q, want existing note retained", requestBody.Order.Note)
	}
	if !strings.Contains(requestBody.Order.Note, "[Mannaiah] Order marked as completed") {
		t.Fatalf("updated note = %q, want completed note", requestBody.Order.Note)
	}
	if !strings.Contains(requestBody.Order.Tags, "vip") {
		t.Fatalf("updated tags = %q, want existing tag retained", requestBody.Order.Tags)
	}
	if !strings.Contains(requestBody.Order.Tags, "mannaiah:completed") {
		t.Fatalf("updated tags = %q, want completion tag", requestBody.Order.Tags)
	}
}

// TestBuildOutboundTagsCleansStatusTransitions verifies outbound status tag hygiene.
func TestBuildOutboundTagsCleansStatusTransitions(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		status   ordersdomain.Status
		want     string
	}{
		{name: "completed removes stale pending hold", existing: "vip, mannaiah:pending, mannaiah:hold", status: ordersdomain.StatusCompleted, want: "vip, mannaiah:completed"},
		{name: "cancelled removes stale active tags", existing: "vip, mannaiah:pending, mannaiah:completed, mannaiah:hold", status: ordersdomain.StatusCancelled, want: "vip, mannaiah:cancelled"},
		{name: "pending retains existing", existing: "vip", status: ordersdomain.StatusPending, want: "vip, mannaiah:pending"},
		{name: "hold removes pending", existing: "vip, mannaiah:pending", status: ordersdomain.StatusHold, want: "vip, mannaiah:hold"},
		{name: "created adds created", existing: "vip", status: ordersdomain.StatusCreated, want: "vip, mannaiah:created"},
		{name: "deduplicates existing", existing: "vip, vip, mannaiah:pending", status: ordersdomain.StatusPending, want: "vip, mannaiah:pending"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildOutboundTags(tc.existing, tc.status); got != tc.want {
				t.Fatalf("buildOutboundTags() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestClientUpdateCustomerTagsMergesWithoutDuplicates verifies customer tag write-back deduplication.
func TestClientUpdateCustomerTagsMergesWithoutDuplicates(t *testing.T) {
	var putBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"customer":{"id":123,"email":"buyer@example.com","tags":"vip, mannaiah:synced"}}`))
		case http.MethodPut:
			var err error
			putBody, err = io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("ReadAll(request.Body) error = %v", err)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"customer":{"id":123}}`))
		default:
			t.Fatalf("request method = %q, want GET or PUT", request.Method)
		}
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

	if err := client.UpdateCustomerTags(shopifyport.WithShopDomain(context.Background(), "flock-6591.myshopify.com"), "123", []string{"mannaiah:synced", "new"}); err != nil {
		t.Fatalf("UpdateCustomerTags() error = %v", err)
	}

	var requestBody struct {
		Customer struct {
			Tags string `json:"tags"`
		} `json:"customer"`
	}
	if err := json.Unmarshal(putBody, &requestBody); err != nil {
		t.Fatalf("json.Unmarshal(putBody) error = %v", err)
	}
	if requestBody.Customer.Tags != "vip, mannaiah:synced, new" {
		t.Fatalf("updated tags = %q, want deduplicated merge", requestBody.Customer.Tags)
	}
}

// TestClientAppendCustomerNoteDoesNotDuplicate verifies customer note append idempotency.
func TestClientAppendCustomerNoteDoesNotDuplicate(t *testing.T) {
	var putBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"customer":{"id":123,"email":"buyer@example.com","note":"existing\n[Mannaiah] contact_id=contact-1"}}`))
		case http.MethodPut:
			var err error
			putBody, err = io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("ReadAll(request.Body) error = %v", err)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"customer":{"id":123}}`))
		default:
			t.Fatalf("request method = %q, want GET or PUT", request.Method)
		}
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

	if err := client.AppendCustomerNote(shopifyport.WithShopDomain(context.Background(), "flock-6591.myshopify.com"), "123", "[Mannaiah] contact_id=contact-1"); err != nil {
		t.Fatalf("AppendCustomerNote() error = %v", err)
	}

	var requestBody struct {
		Customer struct {
			Note string `json:"note"`
		} `json:"customer"`
	}
	if err := json.Unmarshal(putBody, &requestBody); err != nil {
		t.Fatalf("json.Unmarshal(putBody) error = %v", err)
	}
	if strings.Count(requestBody.Customer.Note, "[Mannaiah] contact_id=contact-1") != 1 {
		t.Fatalf("updated note = %q, want one contact note", requestBody.Customer.Note)
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
