package config

import "errors"

var (
	// ErrNilProvider is returned when a nil config provider is used.
	ErrNilProvider = errors.New("config provider must not be nil")
)

// Provider defines a provider-agnostic configuration loading contract.
type Provider interface {
	// Load fills one or more target configuration structs.
	Load(targets ...any) error
}

var (
	// _ ensures Loader implements the abstract Provider contract.
	_ Provider = (*Loader)(nil)
)

// LoadWith loads targets using an abstract configuration provider.
func LoadWith(provider Provider, targets ...any) error {
	if provider == nil {
		return ErrNilProvider
	}

	return provider.Load(targets...)
}
