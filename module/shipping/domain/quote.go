package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Carrier defines supported carrier identifiers.
type Carrier string

const (
	// CarrierTCC defines TCC carrier identifiers.
	CarrierTCC Carrier = "tcc"
)

// BusinessUnit defines supported business-unit identifiers.
type BusinessUnit string

const (
	// BusinessUnitCourier defines parcel/courier business-unit identifiers.
	BusinessUnitCourier BusinessUnit = "courier"
	// BusinessUnitLocals defines local-messenger business-unit identifiers.
	BusinessUnitLocals BusinessUnit = "locals"
)

// QuoteUnit defines one transport unit included in one quote request.
type QuoteUnit struct {
	// Number defines sequential unit numbers.
	Number int
	// RealWeight defines real package weight values in kilograms.
	RealWeight float64
	// Height defines package height values in centimeters.
	Height float64
	// Width defines package width values in centimeters.
	Width float64
	// Length defines package length values in centimeters.
	Length float64
}

// QuoteRequest defines carrier-agnostic quote request values.
type QuoteRequest struct {
	// Carrier defines carrier identifier values.
	Carrier Carrier
	// BusinessUnit defines selected business-unit values.
	BusinessUnit BusinessUnit
	// OriginCityCode defines source city code values.
	OriginCityCode string
	// DestinationCityCode defines destination city code values.
	DestinationCityCode string
	// DeclaredValue defines declared merchandise value.
	DeclaredValue float64
	// Units defines transport unit values.
	Units []QuoteUnit
}

// QuoteResult defines one successful quote response.
type QuoteResult struct {
	// CarrierMessage defines provider success message values.
	CarrierMessage string
	// QuoteValue defines quoted shipment value.
	QuoteValue float64
	// BusinessUnit defines resolved business-unit identifiers.
	BusinessUnit BusinessUnit
}

// NormalizeCarrier normalizes carrier values to canonical lowercase identifiers.
func NormalizeCarrier(value string) Carrier {
	return Carrier(strings.ToLower(strings.TrimSpace(value)))
}

// NormalizeBusinessUnit normalizes business-unit values to canonical lowercase identifiers.
func NormalizeBusinessUnit(value string) BusinessUnit {
	return BusinessUnit(strings.ToLower(strings.TrimSpace(value)))
}

// ValidateQuoteRequest validates quote request values.
func ValidateQuoteRequest(request QuoteRequest) error {
	if request.Carrier == "" {
		return ErrCarrierRequired
	}
	if request.BusinessUnit == "" {
		return ErrBusinessUnitRequired
	}
	if request.BusinessUnit != BusinessUnitCourier && request.BusinessUnit != BusinessUnitLocals {
		return ErrInvalidBusinessUnit
	}
	if strings.TrimSpace(request.OriginCityCode) == "" {
		return ErrOriginCityCodeRequired
	}
	if !isValidCityCode(request.OriginCityCode) {
		return ErrOriginCityCodeInvalid
	}
	if strings.TrimSpace(request.DestinationCityCode) == "" {
		return ErrDestinationCityCodeRequired
	}
	if !isValidCityCode(request.DestinationCityCode) {
		return ErrDestinationCityCodeInvalid
	}
	if request.DeclaredValue < 0 {
		return ErrDeclaredValueInvalid
	}
	if len(request.Units) == 0 {
		return ErrUnitsRequired
	}

	for index, unit := range request.Units {
		if unit.Number <= 0 {
			return fmt.Errorf("unit %d: %w", index+1, ErrUnitNumberInvalid)
		}
		if unit.Number != index+1 {
			return fmt.Errorf("unit %d: %w", index+1, ErrUnitNumberSequenceInvalid)
		}
		if unit.RealWeight <= 0 {
			return fmt.Errorf("unit %d: %w", unit.Number, ErrUnitRealWeightInvalid)
		}
		if unit.Height <= 0 || unit.Width <= 0 || unit.Length <= 0 {
			return fmt.Errorf("unit %d: %w", unit.Number, ErrUnitDimensionInvalid)
		}
	}

	return nil
}

// isValidCityCode reports whether city code values are numeric with supported lengths.
func isValidCityCode(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != 5 && len(trimmed) != 8 {
		return false
	}

	_, err := strconv.Atoi(trimmed)
	return err == nil
}
