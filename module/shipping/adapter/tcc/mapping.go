package tcc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mannaiah/module/shipping/domain"
)

const (
	// volumetricFactor defines TCC volumetric weight multiplication factors.
	volumetricFactor = 0.0004
)

// buildQuotePayload maps domain quote requests into TCC quote payload values.
func (c *Client) buildQuotePayload(request domain.QuoteRequest) (tccQuoteRequest, error) {
	businessUnitID, err := mapBusinessUnit(request.BusinessUnit)
	if err != nil {
		return tccQuoteRequest{}, err
	}

	originCode, err := mapCityCode(request.OriginCityCode)
	if err != nil {
		return tccQuoteRequest{}, fmt.Errorf("%w: %v", domain.ErrOriginCityCodeInvalid, err)
	}
	destinationCode, err := mapCityCode(request.DestinationCityCode)
	if err != nil {
		return tccQuoteRequest{}, fmt.Errorf("%w: %v", domain.ErrDestinationCityCodeInvalid, err)
	}

	units := make([]tccQuoteUnit, 0, len(request.Units))
	for _, unit := range request.Units {
		units = append(units, tccQuoteUnit{
			NumeroUnidades: unit.Number,
			PesoReal:       unit.RealWeight,
			PesoVolumen:    calculateVolumetricWeight(unit),
			Alto:           unit.Height,
			Largo:          unit.Length,
			Ancho:          unit.Width,
			TipoEmpaque:    "",
		})
	}

	return tccQuoteRequest{
		TipoEnvio:       "",
		IDCiudadOrigen:  originCode,
		IDCiudadDestino: destinationCode,
		ValorMercancia:  request.DeclaredValue,
		Boomerang:       0,
		Identificacion:  c.cfg.Identifier,
		Cuenta:          c.cfg.Account,
		FechaRemesa:     time.Now().Format("2006-01-02"),
		IDUnidadNegocio: businessUnitID,
		Unidades:        units,
	}, nil
}

// mapCityCode maps source city code values into TCC city code values.
func mapCityCode(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != 5 && len(trimmed) != 8 {
		return "", errors.New("city code must have 5 or 8 digits")
	}
	if _, err := strconv.Atoi(trimmed); err != nil {
		return "", errors.New("city code must be numeric")
	}
	if len(trimmed) == 8 {
		return trimmed, nil
	}

	return trimmed + "000", nil
}

// mapBusinessUnit maps domain business-unit values into TCC identifiers.
func mapBusinessUnit(value domain.BusinessUnit) (int, error) {
	switch value {
	case domain.BusinessUnitCourier:
		return 1, nil
	case domain.BusinessUnitLocals:
		return 2, nil
	default:
		return 0, domain.ErrInvalidBusinessUnit
	}
}

// calculateVolumetricWeight calculates volumetric weight values for one quote unit.
func calculateVolumetricWeight(unit domain.QuoteUnit) float64 {
	return unit.Height * unit.Width * unit.Length * volumetricFactor
}
