package service

import (
	"context"
	"errors"
	"testing"

	"mannaiah/module/falabella/port"
)

// sourceMock defines Falabella source test doubles.
type sourceMock struct {
	// validateErr defines Validate() errors.
	validateErr error
	// payload defines GetBrands() payload values.
	payload []byte
	// getErr defines GetBrands() errors.
	getErr error
}

// Validate returns configured validation errors.
func (m *sourceMock) Validate(ctx context.Context) error {
	return m.validateErr
}

// GetBrands returns configured payload/errors.
func (m *sourceMock) GetBrands(ctx context.Context) ([]byte, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}

	return m.payload, nil
}

// SyncProduct returns no-op values for brand-service test doubles.
func (m *sourceMock) SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error) {
	return nil, nil
}

// TestNewServiceValidation verifies constructor validation behavior.
func TestNewServiceValidation(t *testing.T) {
	_, err := NewService(nil)
	if !errors.Is(err, ErrNilSource) {
		t.Fatalf("NewService() error = %v, want %v", err, ErrNilSource)
	}
}

// TestValidateIntegration verifies integration validation behavior.
func TestValidateIntegration(t *testing.T) {
	service, err := NewService(&sourceMock{validateErr: errors.New("boom")})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	err = service.ValidateIntegration(context.Background())
	if !errors.Is(err, ErrIntegrationUnavailable) {
		t.Fatalf("ValidateIntegration() error = %v, want %v", err, ErrIntegrationUnavailable)
	}
}

// TestGetBrands verifies GetBrands behavior.
func TestGetBrands(t *testing.T) {
	expected := []byte(`{"brands":[]}`)
	service, err := NewService(&sourceMock{payload: expected})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	payload, err := service.GetBrands(context.Background())
	if err != nil {
		t.Fatalf("GetBrands() error = %v", err)
	}
	if string(payload) != string(expected) {
		t.Fatalf("payload = %q, want %q", string(payload), string(expected))
	}
}

// TestGetBrandsError verifies source error behavior.
func TestGetBrandsError(t *testing.T) {
	service, err := NewService(&sourceMock{getErr: errors.New("boom")})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.GetBrands(context.Background())
	if !errors.Is(err, ErrIntegrationUnavailable) {
		t.Fatalf("GetBrands() error = %v, want %v", err, ErrIntegrationUnavailable)
	}
}
