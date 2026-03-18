package domain

import (
	"errors"
	"testing"
)

// TestValidateQuoteRequest verifies quote-request validation behavior.
func TestValidateQuoteRequest(t *testing.T) {
	base := QuoteRequest{
		Carrier:             CarrierTCC,
		BusinessUnit:        BusinessUnitCourier,
		OriginCityCode:      "05001",
		DestinationCityCode: "11001",
		DeclaredValue:       100000,
		Units: []QuoteUnit{
			{Number: 1, RealWeight: 2, Height: 10, Width: 20, Length: 30},
		},
	}

	if err := ValidateQuoteRequest(base); err != nil {
		t.Fatalf("ValidateQuoteRequest() error = %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(input *QuoteRequest)
		wantErr error
	}{
		{
			name: "missing carrier",
			mutate: func(input *QuoteRequest) {
				input.Carrier = ""
			},
			wantErr: ErrCarrierRequired,
		},
		{
			name: "invalid business unit",
			mutate: func(input *QuoteRequest) {
				input.BusinessUnit = "other"
			},
			wantErr: ErrInvalidBusinessUnit,
		},
		{
			name: "invalid origin city",
			mutate: func(input *QuoteRequest) {
				input.OriginCityCode = "abc"
			},
			wantErr: ErrOriginCityCodeInvalid,
		},
		{
			name: "invalid destination city",
			mutate: func(input *QuoteRequest) {
				input.DestinationCityCode = "x"
			},
			wantErr: ErrDestinationCityCodeInvalid,
		},
		{
			name: "negative declared value",
			mutate: func(input *QuoteRequest) {
				input.DeclaredValue = -1
			},
			wantErr: ErrDeclaredValueInvalid,
		},
		{
			name: "empty units",
			mutate: func(input *QuoteRequest) {
				input.Units = nil
			},
			wantErr: ErrUnitsRequired,
		},
		{
			name: "invalid unit sequence",
			mutate: func(input *QuoteRequest) {
				input.Units = []QuoteUnit{{Number: 2, RealWeight: 1, Height: 1, Width: 1, Length: 1}}
			},
			wantErr: ErrUnitNumberSequenceInvalid,
		},
		{
			name: "invalid real weight",
			mutate: func(input *QuoteRequest) {
				input.Units = []QuoteUnit{{Number: 1, RealWeight: 0, Height: 1, Width: 1, Length: 1}}
			},
			wantErr: ErrUnitRealWeightInvalid,
		},
		{
			name: "invalid dimensions",
			mutate: func(input *QuoteRequest) {
				input.Units = []QuoteUnit{{Number: 1, RealWeight: 1, Height: 0, Width: 1, Length: 1}}
			},
			wantErr: ErrUnitDimensionInvalid,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			input := base
			testCase.mutate(&input)

			err := ValidateQuoteRequest(input)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ValidateQuoteRequest() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}
