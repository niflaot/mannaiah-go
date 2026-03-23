package carrier

import (
	"strings"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// Registry defines carrier and tracking provider lookup behavior.
type Registry struct {
	// providers maps carrier identifiers to carrier providers.
	providers map[string]port.CarrierProvider
	// trackingProviders defines available tracking provider rows.
	trackingProviders []port.TrackingProvider
}

// NewRegistry creates carrier registries from providers.
func NewRegistry(providers []port.CarrierProvider, trackingProviders []port.TrackingProvider) *Registry {
	resolvedProviders := map[string]port.CarrierProvider{}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(provider.CarrierID()))
		if key == "" {
			continue
		}
		resolvedProviders[key] = provider
	}
	resolvedTracking := make([]port.TrackingProvider, 0, len(trackingProviders))
	for _, provider := range trackingProviders {
		if provider == nil {
			continue
		}
		resolvedTracking = append(resolvedTracking, provider)
	}

	return &Registry{providers: resolvedProviders, trackingProviders: resolvedTracking}
}

// CarrierProvider resolves one carrier provider by carrier identifier.
func (r *Registry) CarrierProvider(carrierID string) (port.CarrierProvider, bool) {
	if r == nil {
		return nil, false
	}
	provider, exists := r.providers[strings.ToLower(strings.TrimSpace(carrierID))]

	return provider, exists
}

// TrackingProvider resolves one tracking provider by carrier identifier.
func (r *Registry) TrackingProvider(carrierID string) (port.TrackingProvider, bool) {
	if r == nil {
		return nil, false
	}
	for _, provider := range r.trackingProviders {
		if provider.SupportsCourier(carrierID) {
			return provider, true
		}
	}

	return nil, false
}

// Carriers returns all available carrier descriptors.
func (r *Registry) Carriers() []domain.Carrier {
	if r == nil {
		return nil
	}
	result := make([]domain.Carrier, 0, len(r.providers))
	for _, provider := range r.providers {
		result = append(result, provider.Carrier())
	}

	return result
}
