package runtime

import (
	"context"
	"sync"

	shopifyport "mannaiah/module/shopify/port"
)

// installationResolver caches active Shopify installations for runtime token resolution.
type installationResolver struct {
	// repo defines persisted installation dependencies.
	repo shopifyport.InstallationRepository
	// mu defines cache synchronization behavior.
	mu sync.RWMutex
	// active defines currently installed Shopify stores keyed by normalized domain.
	active map[string]shopifyport.Installation
}

var (
	// _ ensures installationResolver satisfies installation lookup behavior.
	_ shopifyport.InstallationResolver = (*installationResolver)(nil)
)

// newInstallationResolver creates cached installation resolvers over persisted repositories.
func newInstallationResolver(repo shopifyport.InstallationRepository) *installationResolver {
	return &installationResolver{
		repo:   repo,
		active: map[string]shopifyport.Installation{},
	}
}

// Refresh reloads active Shopify installations from persistent storage.
func (r *installationResolver) Refresh(ctx context.Context) error {
	if r == nil || r.repo == nil {
		return nil
	}

	installations, err := r.repo.ListActive(ctx)
	if err != nil {
		return err
	}

	next := make(map[string]shopifyport.Installation, len(installations))
	for _, installation := range installations {
		normalized := shopifyport.NormalizeShopDomain(installation.ShopDomain)
		if normalized == "" {
			continue
		}
		installation.ShopDomain = normalized
		next[normalized] = installation
	}

	r.mu.Lock()
	r.active = next
	r.mu.Unlock()

	return nil
}

// ResolveInstallation resolves one active Shopify installation from the cache.
func (r *installationResolver) ResolveInstallation(ctx context.Context, shopDomain string) (*shopifyport.Installation, error) {
	if r == nil {
		return nil, shopifyport.ErrInstallationNotFound
	}

	r.mu.RLock()
	cacheEmpty := len(r.active) == 0
	r.mu.RUnlock()
	if cacheEmpty {
		if err := r.Refresh(ctx); err != nil {
			return nil, err
		}
	}

	normalized := shopifyport.NormalizeShopDomain(shopDomain)
	r.mu.RLock()
	defer r.mu.RUnlock()

	if normalized != "" {
		installation, exists := r.active[normalized]
		if !exists {
			return nil, shopifyport.ErrInstallationNotFound
		}
		resolved := installation
		return &resolved, nil
	}

	if len(r.active) == 0 {
		return nil, shopifyport.ErrInstallationNotFound
	}
	if len(r.active) > 1 {
		return nil, shopifyport.ErrAmbiguousInstallations
	}

	for _, installation := range r.active {
		resolved := installation
		return &resolved, nil
	}

	return nil, shopifyport.ErrInstallationNotFound
}