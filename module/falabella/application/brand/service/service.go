package service

import (
	"context"
	"errors"
	"fmt"

	"mannaiah/module/falabella/port"
)

var (
	// ErrNilSource is returned when source dependencies are nil.
	ErrNilSource = errors.New("falabella source must not be nil")
	// ErrIntegrationUnavailable is returned when Falabella integration cannot be reached.
	ErrIntegrationUnavailable = errors.New("falabella integration is unavailable")
)

// Service defines Falabella brand use-case behavior.
type Service interface {
	// ValidateIntegration verifies Falabella integration availability.
	ValidateIntegration(ctx context.Context) error
	// GetBrands retrieves Falabella brand payload.
	GetBrands(ctx context.Context) ([]byte, error)
}

// BrandService defines Falabella brand use-case dependencies.
type BrandService struct {
	// source defines Falabella source dependencies.
	source port.Source
}

var (
	// _ ensures BrandService satisfies service contracts.
	_ Service = (*BrandService)(nil)
)

// NewService creates Falabella brand use-case services.
func NewService(source port.Source) (*BrandService, error) {
	if source == nil {
		return nil, ErrNilSource
	}

	return &BrandService{source: source}, nil
}

// ValidateIntegration verifies Falabella integration availability.
func (s *BrandService) ValidateIntegration(ctx context.Context) error {
	if err := s.source.Validate(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
	}

	return nil
}

// GetBrands retrieves Falabella brand payload.
func (s *BrandService) GetBrands(ctx context.Context) ([]byte, error) {
	if err := s.ValidateIntegration(ctx); err != nil {
		return nil, err
	}

	payload, err := s.source.GetBrands(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
	}

	return payload, nil
}
