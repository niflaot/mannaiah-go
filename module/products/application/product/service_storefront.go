package product

import "context"

// StorefrontNavigationRefresher defines background storefront navigation refresh behavior.
type StorefrontNavigationRefresher interface {
	// TriggerRefresh requests a fail-open storefront navigation refresh after product mutations.
	TriggerRefresh(ctx context.Context)
}

// SetStorefrontNavigationRefresher configures storefront navigation refresh dependencies.
func (s *ProductService) SetStorefrontNavigationRefresher(refresher StorefrontNavigationRefresher) {
	if s == nil {
		return
	}

	s.storefrontNavigationRefresher = refresher
}

// triggerStorefrontNavigationRefresh requests a fail-open storefront refresh when configured.
func (s *ProductService) triggerStorefrontNavigationRefresh(ctx context.Context) {
	if s == nil || s.storefrontNavigationRefresher == nil {
		return
	}

	s.storefrontNavigationRefresher.TriggerRefresh(ctx)
}
