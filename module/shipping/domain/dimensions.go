package domain

import "math"

const (
	// DefaultVolumetricFactor defines volumetric weight conversion factors used by TCC flows.
	DefaultVolumetricFactor = 0.0004
)

// Dimensions defines package dimensions and derived weights.
type Dimensions struct {
	// HeightCM defines package height values in centimeters.
	HeightCM float64 `json:"heightCm"`
	// WidthCM defines package width values in centimeters.
	WidthCM float64 `json:"widthCm"`
	// DepthCM defines package depth/length values in centimeters.
	DepthCM float64 `json:"depthCm"`
	// RealWeightKG defines actual package-weight values in kilograms.
	RealWeightKG float64 `json:"realWeightKg"`
	// VolumetricWeightKG defines package volumetric-weight values in kilograms.
	VolumetricWeightKG float64 `json:"volumetricWeightKg"`
	// DeclaredValueCOP defines declared package-value amounts in COP.
	DeclaredValueCOP float64 `json:"declaredValueCop"`
}

// Normalize normalizes dimension values and computes volumetric weight when absent.
func (d Dimensions) Normalize() Dimensions {
	copy := d
	if copy.HeightCM < 0 {
		copy.HeightCM = 0
	}
	if copy.WidthCM < 0 {
		copy.WidthCM = 0
	}
	if copy.DepthCM < 0 {
		copy.DepthCM = 0
	}
	if copy.RealWeightKG < 0 {
		copy.RealWeightKG = 0
	}
	if copy.DeclaredValueCOP < 0 {
		copy.DeclaredValueCOP = 0
	}
	if copy.VolumetricWeightKG <= 0 {
		copy.VolumetricWeightKG = copy.HeightCM * copy.WidthCM * copy.DepthCM * DefaultVolumetricFactor
	}
	if copy.VolumetricWeightKG < 0 {
		copy.VolumetricWeightKG = 0
	}
	copy.RealWeightKG = round2(copy.RealWeightKG)
	copy.VolumetricWeightKG = round2(copy.VolumetricWeightKG)
	copy.DeclaredValueCOP = round2(copy.DeclaredValueCOP)

	return copy
}

// round2 rounds one decimal value to two fraction digits.
func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
