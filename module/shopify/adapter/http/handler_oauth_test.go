package http

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

type oauthInstallTestClient struct{}

// ExchangeAuthorizationCode satisfies the OAuth client interface for install-flow tests.
func (oauthInstallTestClient) ExchangeAuthorizationCode(ctx context.Context, shopDomain string, code string) (string, string, error) {
	return "", "", nil
}

// RegisterWebhooks satisfies the OAuth client interface for install-flow tests.
func (oauthInstallTestClient) RegisterWebhooks(ctx context.Context, shopDomain string, accessToken string, address string) error {
	return nil
}

// TestInstallOAuthRedirectsWithSignedState verifies Shopify install launches redirect to Shopify with a signed state param.
func TestInstallOAuthRedirectsWithSignedState(t *testing.T) {
	handler := &Handler{
		clientID:     "client-id",
		clientSecret: "client-secret",
		oauthClient:  oauthInstallTestClient{},
	}
	requestContext := &launchTestContext{
		queryValues: map[string]string{"shop": "2axh5c-b1.myshopify.com"},
		headers:     map[string]string{"Host": "api.flockstore.co"},
	}

	if err := handler.installOAuth(requestContext); err != nil {
		t.Fatalf("installOAuth() error = %v", err)
	}
	if requestContext.statusCode != 302 {
		t.Fatalf("installOAuth() status = %d, want 302", requestContext.statusCode)
	}
	location := requestContext.headers["Location"]
	if !strings.Contains(location, "2axh5c-b1.myshopify.com/admin/oauth/authorize") {
		t.Fatalf("installOAuth() location = %q, want Shopify authorize redirect", location)
	}
	if !strings.Contains(location, "state=") {
		t.Fatalf("installOAuth() location = %q, want state param", location)
	}
	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse installOAuth() location: %v", err)
	}
	scopes := parsedLocation.Query().Get("scope")
	for _, scope := range []string{"read_orders", "write_orders", "read_customers", "write_customers", "read_products", "read_metaobjects"} {
		if !strings.Contains(scopes, scope) {
			t.Fatalf("installOAuth() scopes = %q, want %q", scopes, scope)
		}
	}
}
