package main

import (
	"errors"
	"testing"

	coremsgbus "mannaiah/module/core/messaging/bus"
	coremsgplatform "mannaiah/module/core/messaging/platform"
)

// TestDecodeShippingMarkGeneratedPayload verifies payload decode and validation behavior.
func TestDecodeShippingMarkGeneratedPayload(t *testing.T) {
	payload, err := decodeShippingMarkGeneratedPayload(coremsgbus.Message{
		Topic:   "shipping.v1.mark.generated",
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1","carrierId":"tcc","trackingNumber":"6039"}`),
	})
	if err != nil {
		t.Fatalf("decodeShippingMarkGeneratedPayload() error = %v", err)
	}
	if payload.MarkID != "mark-1" || payload.OrderID != "order-1" {
		t.Fatalf("payload = %+v", payload)
	}

	_, err = decodeShippingMarkGeneratedPayload(coremsgbus.Message{
		Topic:   "shipping.v1.mark.generated",
		Payload: []byte(`invalid`),
	})
	if !coremsgplatform.IsNonRetriable(err) {
		t.Fatalf("decode invalid payload error = %v, want non-retriable", err)
	}

	_, err = decodeShippingMarkGeneratedPayload(coremsgbus.Message{
		Topic:   "shipping.v1.mark.generated",
		Payload: []byte(`{"markId":"","orderId":"order-1"}`),
	})
	if !errors.Is(err, errShippingMarkGeneratedMessageInvalid) {
		t.Fatalf("decode invalid fields error = %v, want errShippingMarkGeneratedMessageInvalid", err)
	}
	if !coremsgplatform.IsNonRetriable(err) {
		t.Fatalf("decode invalid fields error = %v, want non-retriable", err)
	}
}
