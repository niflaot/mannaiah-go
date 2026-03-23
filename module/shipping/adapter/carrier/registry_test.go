package carrier

import (
	"testing"

	"mannaiah/module/shipping/adapter/carrier/manual"
	"mannaiah/module/shipping/port"
)

// TestNewRegistry verifies provider lookup behavior.
func TestNewRegistry(t *testing.T) {
	provider := manual.NewProvider()
	registry := NewRegistry([]port.CarrierProvider{provider}, []port.TrackingProvider{provider})
	if _, ok := registry.CarrierProvider("manual"); !ok {
		t.Fatalf("manual carrier provider not found")
	}
	if _, ok := registry.TrackingProvider("manual"); !ok {
		t.Fatalf("manual tracking provider not found")
	}
}
