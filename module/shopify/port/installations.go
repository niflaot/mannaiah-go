package port

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

var (
	// ErrInstallationNotFound is returned when a Shopify installation cannot be resolved.
	ErrInstallationNotFound = errors.New("shopify installation not found")
	// ErrShopDomainRequired is returned when a Shopify store domain is required but missing.
	ErrShopDomainRequired = errors.New("shopify shop domain is required")
	// ErrAmbiguousInstallations is returned when more than one active installation exists without an explicit shop.
	ErrAmbiguousInstallations = errors.New("shopify shop domain is required when multiple installations are active")
)

// InstallationResolver defines active-installation lookup behavior.
type InstallationResolver interface {
	// ResolveInstallation resolves one active installation, optionally scoped by shop domain.
	ResolveInstallation(ctx context.Context, shopDomain string) (*Installation, error)
	// Refresh reloads active installations from persistent storage.
	Refresh(ctx context.Context) error
}

type shopDomainContextKey struct{}

// WithShopDomain stores a Shopify shop domain in the provided request context.
func WithShopDomain(ctx context.Context, shopDomain string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	resolved := NormalizeShopDomain(shopDomain)
	if resolved == "" {
		return ctx
	}

	return context.WithValue(ctx, shopDomainContextKey{}, resolved)
}

// ShopDomainFromContext resolves one Shopify shop domain from the request context.
func ShopDomainFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(shopDomainContextKey{}).(string)
	return NormalizeShopDomain(value)
}

// NormalizeShopDomain coerces raw values into normalized Shopify store domains.
func NormalizeShopDomain(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err == nil {
			trimmed = strings.TrimSpace(strings.ToLower(parsed.Host))
		}
	}

	trimmed = strings.TrimPrefix(trimmed, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	trimmed = strings.TrimSuffix(trimmed, "/")
	return trimmed
}