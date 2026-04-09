package domain

import "time"

// TrackingStatus defines normalized tracking status values.
type TrackingStatus string

const (
	// TrackingStatusProcessing defines in-transit processing statuses.
	TrackingStatusProcessing TrackingStatus = "PROCESSING"
	// TrackingStatusOrigin defines origin-facility statuses.
	TrackingStatusOrigin TrackingStatus = "ORIGIN"
	// TrackingStatusCompleted defines delivered statuses.
	TrackingStatusCompleted TrackingStatus = "COMPLETED"
	// TrackingStatusReturn defines return-to-sender statuses.
	TrackingStatusReturn TrackingStatus = "RETURN"
	// TrackingStatusIncidence defines incident statuses.
	TrackingStatusIncidence TrackingStatus = "INCIDENCE"
	// TrackingStatusVoided defines voided or canceled statuses.
	TrackingStatusVoided TrackingStatus = "VOIDED"
)

// TrackingEvent defines one tracking checkpoint.
type TrackingEvent struct {
	// Date defines checkpoint timestamps.
	Date time.Time `json:"date"`
	// Code defines provider event-code values.
	Code string `json:"code,omitempty"`
	// Text defines provider event-description values.
	Text string `json:"text"`
	// City defines city labels for the checkpoint.
	City string `json:"city,omitempty"`
	// Status defines normalized status values for the checkpoint.
	Status TrackingStatus `json:"status"`
}

// TrackingHistory defines normalized tracking history values.
type TrackingHistory struct {
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// TrackingNumber defines tracking number values.
	TrackingNumber string `json:"trackingNumber"`
	// GlobalStatus defines last known global status values.
	GlobalStatus TrackingStatus `json:"globalStatus"`
	// LastUpdate defines latest update timestamps.
	LastUpdate time.Time `json:"lastUpdate"`
	// History defines chronological history rows.
	History []TrackingEvent `json:"history"`
}
