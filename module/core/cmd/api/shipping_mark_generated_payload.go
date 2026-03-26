package main

import (
	"encoding/json"
	"errors"
	"strings"

	coremsgbus "mannaiah/module/core/messaging/bus"
	coremsgplatform "mannaiah/module/core/messaging/platform"
)

var (
	// errShippingMarkGeneratedMessageInvalid is returned when mark-generated event payload values are invalid.
	errShippingMarkGeneratedMessageInvalid = errors.New("shipping mark generated payload is invalid")
)

// shippingMarkGeneratedPayload defines shipping mark-generated event payload values.
type shippingMarkGeneratedPayload struct {
	// MarkID defines mark identifier values.
	MarkID string `json:"markId"`
	// OrderID defines order identifier values.
	OrderID string `json:"orderId"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// TrackingNumber defines tracking-number values.
	TrackingNumber string `json:"trackingNumber"`
	// DocumentRef defines document-reference values.
	DocumentRef string `json:"documentRef"`
}

// decodeShippingMarkGeneratedPayload decodes and validates mark-generated event payload values.
func decodeShippingMarkGeneratedPayload(message coremsgbus.Message) (shippingMarkGeneratedPayload, error) {
	payload := shippingMarkGeneratedPayload{}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return shippingMarkGeneratedPayload{}, coremsgplatform.NonRetriable(err)
	}
	payload.MarkID = strings.TrimSpace(payload.MarkID)
	payload.OrderID = strings.TrimSpace(payload.OrderID)
	payload.CarrierID = strings.TrimSpace(payload.CarrierID)
	payload.TrackingNumber = strings.TrimSpace(payload.TrackingNumber)
	payload.DocumentRef = strings.TrimSpace(payload.DocumentRef)
	if payload.MarkID == "" || payload.OrderID == "" {
		return shippingMarkGeneratedPayload{}, coremsgplatform.NonRetriable(errShippingMarkGeneratedMessageInvalid)
	}

	return payload, nil
}
