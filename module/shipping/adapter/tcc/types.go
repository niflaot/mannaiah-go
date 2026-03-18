package tcc

import (
	"strconv"
	"strings"
)

// tccQuoteRequest defines TCC quote request payload values.
type tccQuoteRequest struct {
	// TipoEnvio defines shipment type values.
	TipoEnvio string `json:"tipoenvio"`
	// IDCiudadOrigen defines source city code values.
	IDCiudadOrigen string `json:"idciudadorigen"`
	// IDCiudadDestino defines destination city code values.
	IDCiudadDestino string `json:"idciudaddestino"`
	// ValorMercancia defines declared merchandise values.
	ValorMercancia float64 `json:"valormercancia"`
	// Boomerang defines return-shipment flag values.
	Boomerang int `json:"boomerang"`
	// Identificacion defines customer identifier values.
	Identificacion string `json:"identificacion"`
	// Cuenta defines customer account values.
	Cuenta string `json:"cuenta"`
	// FechaRemesa defines shipment date values.
	FechaRemesa string `json:"fecharemesa"`
	// IDUnidadNegocio defines business-unit identifier values.
	IDUnidadNegocio int `json:"idunidadnegocio"`
	// Unidades defines package unit payload values.
	Unidades []tccQuoteUnit `json:"unidades"`
}

// tccQuoteUnit defines TCC quote unit payload values.
type tccQuoteUnit struct {
	// NumeroUnidades defines sequential unit numbers.
	NumeroUnidades int `json:"numerounidades"`
	// PesoReal defines real-weight values.
	PesoReal float64 `json:"pesoreal"`
	// PesoVolumen defines volumetric-weight values.
	PesoVolumen float64 `json:"pesovolumen"`
	// Alto defines height values.
	Alto float64 `json:"alto"`
	// Largo defines length values.
	Largo float64 `json:"largo"`
	// Ancho defines width values.
	Ancho float64 `json:"ancho"`
	// TipoEmpaque defines package-type values.
	TipoEmpaque string `json:"tipoempaque"`
}

// tccQuoteResponse defines TCC quote response payload values.
type tccQuoteResponse struct {
	// CodigoResultado defines provider result code values.
	CodigoResultado string `json:"codigoResultado"`
	// MensajeResultado defines provider result message values.
	MensajeResultado string `json:"mensajeResultado"`
	// Total defines quote total payload values.
	Total *tccQuoteTotal `json:"total"`
}

// tccQuoteTotal defines TCC quote total payload values.
type tccQuoteTotal struct {
	// TotalDespacho defines quote amount values.
	TotalDespacho flexibleFloat `json:"totaldespacho"`
	// UnidadNegocio defines provider business-unit label values.
	UnidadNegocio string `json:"unidadnegocio"`
}

// flexibleFloat defines float values decoded from JSON numbers or strings.
type flexibleFloat float64

// UnmarshalJSON decodes float values from numeric or string JSON payload values.
func (f *flexibleFloat) UnmarshalJSON(value []byte) error {
	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" || trimmed == "null" {
		*f = 0
		return nil
	}
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		parsed, err := strconv.ParseFloat(strings.Trim(trimmed, "\""), 64)
		if err != nil {
			return err
		}
		*f = flexibleFloat(parsed)
		return nil
	}

	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return err
	}
	*f = flexibleFloat(parsed)
	return nil
}
