package runtime

import (
	"context"
	"errors"
	"testing"

	"mannaiah/module/falabella/port"
)

// sourceMock defines source behavior for runtime source tests.
type sourceMock struct {
	// validateErr defines Validate() errors.
	validateErr error
	// payload defines GetBrands() payload values.
	payload []byte
	// getErr defines GetBrands() errors.
	getErr error
	// syncPayload defines SyncProduct() payload values.
	syncPayload []byte
	// syncErr defines SyncProduct() errors.
	syncErr error
}

// Validate returns configured validation errors.
func (m sourceMock) Validate(ctx context.Context) error {
	return m.validateErr
}

// GetBrands returns configured payload/errors.
func (m sourceMock) GetBrands(ctx context.Context) ([]byte, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}

	return m.payload, nil
}

// SyncProduct returns configured payload/errors.
func (m sourceMock) SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error) {
	if m.syncErr != nil {
		return nil, m.syncErr
	}

	return m.syncPayload, nil
}

// breakerMock defines circuit-breaker behavior for runtime source tests.
type breakerMock struct {
	// executeErr defines Execute() errors.
	executeErr error
	// open defines IsOpenError() behavior.
	open bool
}

// Execute executes operations or returns configured errors.
func (m breakerMock) Execute(operation func() error) error {
	if m.executeErr != nil {
		return m.executeErr
	}

	return operation()
}

// IsOpenError reports open-circuit behavior.
func (m breakerMock) IsOpenError(err error) bool {
	return m.open
}

// TestProtectedSourceValidate verifies breaker-protected Validate behavior.
func TestProtectedSourceValidate(t *testing.T) {
	wrapped := protectedSource{source: sourceMock{}, breaker: breakerMock{}}
	if err := wrapped.Validate(context.Background()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// TestProtectedSourceGetBrands verifies breaker-protected GetBrands behavior.
func TestProtectedSourceGetBrands(t *testing.T) {
	expected := []byte(`{"ok":true}`)
	wrapped := protectedSource{source: sourceMock{payload: expected}, breaker: breakerMock{}}
	payload, err := wrapped.GetBrands(context.Background())
	if err != nil {
		t.Fatalf("GetBrands() error = %v", err)
	}
	if string(payload) != string(expected) {
		t.Fatalf("payload = %q, want %q", string(payload), string(expected))
	}
}

// TestProtectedSourceSyncProduct verifies breaker-protected SyncProduct behavior.
func TestProtectedSourceSyncProduct(t *testing.T) {
	expected := []byte("<ok/>")
	wrapped := protectedSource{source: sourceMock{syncPayload: expected}, breaker: breakerMock{}}
	payload, err := wrapped.SyncProduct(context.Background(), port.SyncProductRequest{SKU: "SKU-1"})
	if err != nil {
		t.Fatalf("SyncProduct() error = %v", err)
	}
	if string(payload) != string(expected) {
		t.Fatalf("payload = %q, want %q", string(payload), string(expected))
	}
}

// TestProtectedSourceOpenBreaker verifies open-breaker error mapping behavior.
func TestProtectedSourceOpenBreaker(t *testing.T) {
	wrapped := protectedSource{source: sourceMock{}, breaker: breakerMock{executeErr: errors.New("open"), open: true}}
	if err := wrapped.Validate(context.Background()); err == nil {
		t.Fatalf("Validate() expected error")
	}
}

// TestFailingSource verifies failing-source behavior.
func TestFailingSource(t *testing.T) {
	failure := errors.New("invalid")
	source := failingSource{err: failure}
	if err := source.Validate(context.Background()); !errors.Is(err, failure) {
		t.Fatalf("Validate() error = %v, want %v", err, failure)
	}
	if _, err := source.GetBrands(context.Background()); !errors.Is(err, failure) {
		t.Fatalf("GetBrands() error = %v, want %v", err, failure)
	}
	if _, err := source.SyncProduct(context.Background(), port.SyncProductRequest{SKU: "SKU-1"}); !errors.Is(err, failure) {
		t.Fatalf("SyncProduct() error = %v, want %v", err, failure)
	}
}
