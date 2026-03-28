package domain

import "errors"

var (
	// ErrInvalidID is returned when identifiers are empty.
	ErrInvalidID = errors.New("shipping id is invalid")
	// ErrInvalidCarrierID is returned when carrier identifiers are empty.
	ErrInvalidCarrierID = errors.New("shipping carrier id is invalid")
	// ErrCarrierNotSupported is returned when no provider supports a carrier.
	ErrCarrierNotSupported = errors.New("shipping carrier is not supported")
	// ErrQuotationNotSupported is returned when quotation is not available for one carrier.
	ErrQuotationNotSupported = errors.New("shipping quotation is not supported")
	// ErrTrackingNotSupported is returned when tracking is not available for one carrier.
	ErrTrackingNotSupported = errors.New("shipping tracking is not supported")
	// ErrInsufficientBalance is returned when one carrier account has insufficient balance.
	ErrInsufficientBalance = errors.New("shipping carrier balance is insufficient")
	// ErrInvalidMarkStatus is returned when one mark status transition is invalid.
	ErrInvalidMarkStatus = errors.New("shipping mark status is invalid")
	// ErrInvalidBatchStatus is returned when one batch status transition is invalid.
	ErrInvalidBatchStatus = errors.New("shipping batch status is invalid")
	// ErrBatchClosed is returned when closed batches are mutated.
	ErrBatchClosed = errors.New("shipping batch is closed")
	// ErrBatchCarrierMismatch is returned when one mark carrier does not match the batch carrier.
	ErrBatchCarrierMismatch = errors.New("shipping batch carrier mismatch")
	// ErrBatchMarkStatusMismatch is returned when one mark is not generated before assignment.
	ErrBatchMarkStatusMismatch = errors.New("shipping batch mark status mismatch")
	// ErrMarkNotDraft is returned when a non-QUOTED mark is operated on as a draft.
	ErrMarkNotDraft = errors.New("shipping mark is not a draft")
	// ErrInvalidShipmentMode is returned when a shipment mode is not parcel or express.
	ErrInvalidShipmentMode = errors.New("shipping shipment mode is invalid")
	// ErrNotFound is returned when one shipping resource does not exist.
	ErrNotFound = errors.New("shipping resource not found")
	// ErrNoValidProducts is returned when no order products have the required shipping dimension attributes.
	ErrNoValidProducts = errors.New("no valid products with shipping dimensions found")
)
