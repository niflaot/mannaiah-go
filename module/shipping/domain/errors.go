package domain

import "errors"

var (
	// ErrCarrierRequired is returned when carrier values are empty.
	ErrCarrierRequired = errors.New("shipping carrier is required")
	// ErrUnsupportedCarrier is returned when no carrier adapter exists for requested carrier values.
	ErrUnsupportedCarrier = errors.New("shipping carrier is not supported")
	// ErrBusinessUnitRequired is returned when business-unit values are empty.
	ErrBusinessUnitRequired = errors.New("shipping business unit is required")
	// ErrInvalidBusinessUnit is returned when business-unit values are not recognized.
	ErrInvalidBusinessUnit = errors.New("shipping business unit is invalid")
	// ErrOriginCityCodeRequired is returned when origin city code values are empty.
	ErrOriginCityCodeRequired = errors.New("shipping origin city code is required")
	// ErrOriginCityCodeInvalid is returned when origin city code format values are invalid.
	ErrOriginCityCodeInvalid = errors.New("shipping origin city code is invalid")
	// ErrDestinationCityCodeRequired is returned when destination city code values are empty.
	ErrDestinationCityCodeRequired = errors.New("shipping destination city code is required")
	// ErrDestinationCityCodeInvalid is returned when destination city code format values are invalid.
	ErrDestinationCityCodeInvalid = errors.New("shipping destination city code is invalid")
	// ErrDeclaredValueInvalid is returned when declared merchandise value is negative.
	ErrDeclaredValueInvalid = errors.New("shipping declared value must be greater than or equal to zero")
	// ErrUnitsRequired is returned when quote unit payload values are empty.
	ErrUnitsRequired = errors.New("shipping units are required")
	// ErrUnitNumberInvalid is returned when unit number values are invalid.
	ErrUnitNumberInvalid = errors.New("shipping unit number must be greater than zero")
	// ErrUnitNumberSequenceInvalid is returned when unit numbering values are not sequential.
	ErrUnitNumberSequenceInvalid = errors.New("shipping unit numbers must be sequential from 1")
	// ErrUnitRealWeightInvalid is returned when real weight values are not positive.
	ErrUnitRealWeightInvalid = errors.New("shipping unit real weight must be greater than zero")
	// ErrUnitDimensionInvalid is returned when dimension values are not positive.
	ErrUnitDimensionInvalid = errors.New("shipping unit dimensions must be greater than zero")
	// ErrIntegrationUnavailable is returned when carrier integration dependencies are unavailable.
	ErrIntegrationUnavailable = errors.New("shipping integration is unavailable")
	// ErrQuoteRejected is returned when carriers reject quote requests.
	ErrQuoteRejected = errors.New("shipping quote was rejected by carrier")
)
