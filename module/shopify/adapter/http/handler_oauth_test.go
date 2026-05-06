package http

import (
	"context"
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

// TestInstallOAuthSetsCrossSiteStateCookie verifies Shopify install launches emit a state cookie usable from Shopify-admin cross-site flows.
func TestInstallOAuthSetsCrossSiteStateCookie(t *testing.T) {
	handler := &Handler{
		clientID:     "client-id",
		clientSecret: "client-secret",
		oauthClient:  oauthInstallTestClient{},
	}
	requestContext := &launchTestContext{
		queryValues: map[string]string{"shop": "2axh5c-b1.myshopify.com"},
		headers: map[string]string{
			"Host": "api.flockstore.co",
		},
	}

	if err := handler.installOAuth(requestContext); err != nil {
		t.Fatalf("installOAuth() error = %v", err)
	}
	if requestContext.statusCode != 302 {
		t.Fatalf("installOAuth() status = %d, want %d", requestContext.statusCode, 302)
	}
	if location := requestContext.headers["Location"]; !strings.Contains(location, "2axh5c-b1.myshopify.com/admin/oauth/authorize") {
		t.Fatalf("installOAuth() location = %q, want Shopify authorize redirect", location)
	}
	setCookie := requestContext.headers["Set-Cookie"]
	if !strings.Contains(setCookie, "shopify_oauth_state=") {
		t.Fatalf("installOAuth() set-cookie = %q, want oauth state cookie", setCookie)
	}
	if !strings.Contains(setCookie, "SameSite=None") {
		t.Fatalf("installOAuth() set-cookie = %q, want SameSite=None", setCookie)
	}
	if !strings.Contains(setCookie, "Secure") {
		t.Fatalf("installOAuth() set-cookie = %q, want Secure", setCookie)
	}
	if !strings.Contains(setCookie, "HttpOnly") {
		t.Fatalf("installOAuth() set-cookie = %q, want HttpOnly", setCookie)
	}
	if !strings.Contains(setCookie, "Path=/shopify/oauth") {
		t.Fatalf("installOAuth() set-cookie = %q, want oauth cookie path", setCookie)
	}
}
