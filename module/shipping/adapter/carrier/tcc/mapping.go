package tcc

import (
	"strconv"
	"strings"
	"time"

	"mannaiah/module/shipping/domain"
)

// QuoteRequest defines TCC quotation request payload values.
type QuoteRequest struct {
	TipoEnvio      string             `json:"tipoenvio"`
	OriginCityCode string             `json:"idciudadorigen"`
	DestCityCode   string             `json:"idciudaddestino"`
	DeclaredValue  float64            `json:"valormercancia"`
	Boomerang      int                `json:"boomerang"`
	Identification string             `json:"identificacion"`
	Account        string             `json:"cuenta"`
	ShipmentDate   string             `json:"fecharemesa"`
	BusinessUnitID int                `json:"idunidadnegocio"`
	Units          []QuoteRequestUnit `json:"unidades"`
}

// QuoteRequestUnit defines TCC quotation unit values.
type QuoteRequestUnit struct {
	NumberOfUnits int     `json:"numerounidades"`
	RealWeight    float64 `json:"pesoreal"`
	VolumeWeight  float64 `json:"pesovolumen"`
	HeightCM      float64 `json:"alto"`
	DepthCM       float64 `json:"largo"`
	WidthCM       float64 `json:"ancho"`
	PackageType   string  `json:"tipoempaque"`
}

// QuoteResponse defines TCC quotation response payload values.
type QuoteResponse struct {
	ResultCode    string `json:"codigoResultado"`
	ResultMessage string `json:"mensajeResultado"`
	Total         struct {
		DispatchTotal float64 `json:"totaldespacho"`
		BusinessUnit  string  `json:"unidadnegocio"`
	} `json:"total"`
}

// DispatchRequest defines TCC dispatch request payload values.
type DispatchRequest struct {
	DispatchNumber       string              `json:"numerodespacho"`
	DispatchDate         string              `json:"fechadespacho"`
	BusinessUnit         int                 `json:"unidadnegocio"`
	SenderAccount        string              `json:"cuentaremitente"`
	SenderBranch         string              `json:"sederemitente"`
	SenderFirstName      string              `json:"primernombreremitente"`
	SenderSecondName     string              `json:"segundonombreremitente"`
	SenderFirstLastName  string              `json:"primerapellidoremitente"`
	SenderSecondLastName string              `json:"segundoapellidoremitente"`
	SenderCompanyName    string              `json:"razonsocialremitente"`
	SenderContact        string              `json:"contactoremitente"`
	SenderIDType         string              `json:"tipoidentificacionremitente"`
	SenderID             string              `json:"identificacionremitente"`
	SenderAddress        string              `json:"direccionremitente"`
	OriginCityCode       string              `json:"ciudadorigen"`
	SenderPhone          string              `json:"telefonoremitente"`
	SenderEmail          string              `json:"emailremitente"`
	Recipients           []DispatchRecipient `json:"destinatarios"`
	ReferenceDocuments   []DispatchDocument  `json:"documentosreferencia"`
}

// DispatchRecipient defines TCC dispatch recipient values.
type DispatchRecipient struct {
	ControlNumber           string         `json:"numerocontrol"`
	ShipmentNumber          string         `json:"numeroremesa"`
	ClientReferenceNumber   string         `json:"numeroreferenciacliente"`
	RecipientIDType         string         `json:"tipoidentificaciondestinatario"`
	RecipientID             string         `json:"identificaciondestinatario"`
	RecipientBranch         string         `json:"sededestinatario"`
	RecipientFirstName      string         `json:"primernombredestinatario"`
	RecipientSecondName     string         `json:"segundonombredestinatario"`
	RecipientFirstLastName  string         `json:"primerapellidodestinatario"`
	RecipientSecondLastName string         `json:"segundoapellidodestinatario"`
	RecipientCompanyName    string         `json:"razonsocialdestinatario"`
	RecipientContact        string         `json:"contactodestinatario"`
	RecipientAddress        string         `json:"direcciondestinatario"`
	RecipientPhone          string         `json:"telefonodestinatario"`
	DestCityCode            string         `json:"ciudaddestino"`
	PaymentForm             int            `json:"formapago"`
	DeliverWarehouse        string         `json:"llevabodega"`
	PickupWarehouse         string         `json:"recogebodega"`
	CostCenter              string         `json:"centrocostos"`
	ServiceType             string         `json:"tiposervicio"`
	Observations            string         `json:"observaciones"`
	ProductCollection       string         `json:"recaudoproducto"`
	Units                   []DispatchUnit `json:"unidades"`
}

// DispatchUnit defines TCC dispatch unit values.
type DispatchUnit struct {
	UnitType       string  `json:"tipounidad"`
	PackageType    string  `json:"tipoempaque"`
	PackageClass   string  `json:"claseempaque"`
	Contains       string  `json:"dicecontener"`
	RealWeightKG   float64 `json:"kilosreales"`
	DepthCM        float64 `json:"largo"`
	HeightCM       float64 `json:"alto"`
	WidthCM        float64 `json:"ancho"`
	VolumeWeightKG float64 `json:"pesovolumen"`
	DeclaredValue  float64 `json:"valormercancia"`
	Barcode        string  `json:"codigodebarras"`
	BagNumber      string  `json:"numerobolsa"`
	References     string  `json:"referencias"`
	InnerUnits     string  `json:"unidadesinternas"`
}

// DispatchDocument defines TCC dispatch reference-document values.
type DispatchDocument struct {
	DocumentType   string `json:"tipodocumento"`
	DocumentNumber string `json:"numerodocumento"`
	DocumentDate   string `json:"fechadocumento"`
}

// DispatchResponse defines TCC dispatch response payload values.
type DispatchResponse struct {
	ResultCode       string                     `json:"codigoresultado"`
	ResultMessage    string                     `json:"mensajeresultado"`
	ShipmentNumber   string                     `json:"numeroremesa"`
	TrackingURL      string                     `json:"urlguia"`
	LabelURL         string                     `json:"urlrotulo"`
	InvoiceURL       string                     `json:"urlfactura"`
	DispatchID       string                     `json:"numerodespacho"`
	ShipmentRelation string                     `json:"urlrelacionenvio"`
	ShipmentDocURL   string                     `json:"urlremesa"`
	ShipmentLabelURL string                     `json:"urlrotulos"`
	Remittances      []DispatchResponseShipment `json:"remesas"`
}

// DispatchResponseShipment defines TCC remittance result rows.
type DispatchResponseShipment struct {
	// ResultCode defines remittance result-code values.
	ResultCode int `json:"codigoresultado"`
	// ShipmentNumber defines remittance tracking-number values.
	ShipmentNumber string `json:"numeroremesa"`
	// ResultMessage defines remittance result-message values.
	ResultMessage string `json:"mensajeresultado"`
}

// TrackingRequest defines TCC tracking request payload values.
type TrackingRequest struct {
	ShipmentNumber string `json:"numeroremesa"`
}

// TrackingResponse defines TCC tracking response payload values.
type TrackingResponse struct {
	ResultCode    string               `json:"codigoresultado"`
	ResultMessage string               `json:"mensajeresultado"`
	States        []TrackingState      `json:"estados"`
	OriginCity    trackingCityResponse `json:"ciudadorigen"`
	DestCity      trackingCityResponse `json:"ciudaddestino"`
}

// TrackingState defines TCC tracking state payload values.
type TrackingState struct {
	Code        string `json:"codigo"`
	Description string `json:"descripcion"`
	Date        string `json:"fecha"`
}

type trackingCityResponse struct {
	Description string `json:"descripcion"`
}

// BuildQuoteRequest maps domain quotation values into TCC quotation request values.
func BuildQuoteRequest(account string, businessUnit int, request domain.QuotationRequest) QuoteRequest {
	normalized := request.Normalize()
	units := make([]QuoteRequestUnit, 0, len(normalized.Units))
	for _, unit := range normalized.Units {
		units = append(units, QuoteRequestUnit{
			NumberOfUnits: 1,
			RealWeight:    max(unit.Dimensions.RealWeightKG, 1),
			VolumeWeight:  max(unit.Dimensions.VolumetricWeightKG, 1),
			HeightCM:      unit.Dimensions.HeightCM,
			DepthCM:       unit.Dimensions.DepthCM,
			WidthCM:       unit.Dimensions.WidthCM,
			PackageType:   "",
		})
	}

	return QuoteRequest{
		TipoEnvio:      "",
		OriginCityCode: normalized.OriginCityCode,
		DestCityCode:   normalized.DestCityCode,
		DeclaredValue:  normalized.DeclaredValue,
		Boomerang:      0,
		Identification: "",
		Account:        strings.TrimSpace(account),
		ShipmentDate:   time.Now().UTC().Format("2006-01-02"),
		BusinessUnitID: businessUnit,
		Units:          units,
	}
}

// ToDomain maps one TCC quotation response into domain quotation values.
func (r QuoteResponse) ToDomain(carrierID string, request domain.QuotationRequest) *domain.QuotationResult {
	normalized := request.Normalize()

	return &domain.QuotationResult{
		CarrierID:      strings.TrimSpace(carrierID),
		OrderID:        normalized.OrderID,
		OriginCityCode: normalized.OriginCityCode,
		DestCityCode:   normalized.DestCityCode,
		FreightCost:    r.Total.DispatchTotal,
		EstimatedDays:  1,
		CurrencyCode:   "COP",
		ExpiresAt:      time.Now().UTC().Add(60 * time.Second),
		RawResponse:    r.ResultMessage,
	}
}

// MapTrackingStatus maps TCC state-code values into normalized tracking statuses.
func MapTrackingStatus(code string, description string) domain.TrackingStatus {
	trimmedCode := strings.TrimSpace(code)
	switch trimmedCode {
	case "901", "205", "2000":
		return domain.TrackingStatusProcessing
	case "500":
		return domain.TrackingStatusOrigin
	case "3000":
		return domain.TrackingStatusCompleted
	}
	lowerDescription := strings.ToLower(strings.TrimSpace(description))
	if strings.Contains(lowerDescription, "devol") {
		return domain.TrackingStatusReturn
	}
	if strings.Contains(lowerDescription, "noved") || strings.Contains(lowerDescription, "inciden") {
		return domain.TrackingStatusIncidence
	}

	return domain.TrackingStatusProcessing
}

// ParseTrackingDate parses TCC tracking date values.
func ParseTrackingDate(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Now().UTC()
	}
	layouts := []string{
		"2/01/2006 3:04:05 p. m.",
		"2/01/2006 3:04:05 a. m.",
		"2/01/2006 3:04:05 PM",
		"2/01/2006 3:04:05 AM",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, trimmed, time.Local); err == nil {
			return parsed.UTC()
		}
	}

	return time.Now().UTC()
}

func max(value float64, minimum float64) float64 {
	if value < minimum {
		return minimum
	}

	return value
}

// BuildShipmentNumber parses response shipment number from possible field variants.
func (r DispatchResponse) BuildShipmentNumber() string {
	for _, remittance := range r.Remittances {
		if strings.TrimSpace(remittance.ShipmentNumber) != "" {
			return strings.TrimSpace(remittance.ShipmentNumber)
		}
	}
	if strings.TrimSpace(r.ShipmentNumber) != "" {
		return strings.TrimSpace(r.ShipmentNumber)
	}
	if strings.TrimSpace(r.DispatchID) != "" {
		return strings.TrimSpace(r.DispatchID)
	}

	return ""
}

// BuildTrackURL resolves the preferred TCC tracking URL from response fields.
func (r DispatchResponse) BuildTrackURL() string {
	if strings.TrimSpace(r.ShipmentLabelURL) != "" {
		return strings.TrimSpace(r.ShipmentLabelURL)
	}
	if strings.TrimSpace(r.ShipmentDocURL) != "" {
		return strings.TrimSpace(r.ShipmentDocURL)
	}
	if strings.TrimSpace(r.ShipmentRelation) != "" {
		return strings.TrimSpace(r.ShipmentRelation)
	}
	if strings.TrimSpace(r.TrackingURL) != "" {
		return strings.TrimSpace(r.TrackingURL)
	}
	if strings.TrimSpace(r.LabelURL) != "" {
		return strings.TrimSpace(r.LabelURL)
	}

	return strings.TrimSpace(r.InvoiceURL)
}

// ParseResultCode parses TCC result-code values.
func ParseResultCode(code string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(code))
	if err != nil {
		return -1
	}

	return parsed
}
