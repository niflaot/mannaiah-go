package tcc

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"mannaiah/module/shipping/domain"
)

var colombiaLocation = func() *time.Location {
	location, err := time.LoadLocation("America/Bogota")
	if err == nil {
		return location
	}

	return time.FixedZone("America/Bogota", -5*60*60)
}()

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
	RelationNumber          string                `json:"numerorelacion"`
	RelationDateTime        string                `json:"fechahorarelacion"`
	PickupRequest           DispatchPickupRequest `json:"solicitudrecogida"`
	BusinessUnit            string                `json:"unidadnegocio"`
	ShipmentNumber          string                `json:"numeroremesa"`
	DispatchDate            string                `json:"fechadespacho"`
	SenderAccount           string                `json:"cuentaremitente"`
	SenderIDType            string                `json:"tipoidentificacionremitente"`
	SenderID                string                `json:"identificacionremitente"`
	SenderBranch            string                `json:"sederemitente"`
	SenderFirstName         string                `json:"primernombreremitente"`
	SenderSecondName        string                `json:"segundonombreremitente"`
	SenderFirstLastName     string                `json:"primerapellidoremitente"`
	SenderSecondLastName    string                `json:"segundoapellidoremitente"`
	SenderCompanyName       string                `json:"razonsocialremitente"`
	SenderNature            string                `json:"naturalezaremitente"`
	SenderAddress           string                `json:"direccionremitente"`
	SenderContact           string                `json:"contactoremitente"`
	SenderEmail             string                `json:"emailremitente"`
	SenderPhone             string                `json:"telefonoremitente"`
	OriginCityCode          string                `json:"ciudadorigen"`
	RecipientIDType         string                `json:"tipoidentificaciondestinatario"`
	RecipientID             string                `json:"identificaciondestinatario"`
	RecipientBranch         string                `json:"sededestinatario"`
	RecipientFirstName      string                `json:"primernombredestinatario"`
	RecipientSecondName     string                `json:"segundonombredestinatario"`
	RecipientFirstLast      string                `json:"primerapellidodestinatario"`
	RecipientSecondLast     string                `json:"segundoapellidodestinatario"`
	RecipientCompany        string                `json:"razonsocialdestinatario"`
	RecipientNature         string                `json:"naturalezadestinatario"`
	RecipientAddress        string                `json:"direcciondestinatario"`
	RecipientContact        string                `json:"contactodestinatario"`
	RecipientEmail          string                `json:"emaildestinatario"`
	RecipientPhone          string                `json:"telefonodestinatario"`
	DestCityCode            string                `json:"ciudaddestinatario"`
	RecipientNeighborhood   string                `json:"barriodestinatario"`
	TotalWeight             string                `json:"totalpeso"`
	TotalVolumeWeight       string                `json:"totalpesovolumen"`
	PaymentForm             string                `json:"formapago"`
	CollectOnDeliveryAmount *string               `json:"recaudoproducto,omitempty"`
	Observations            string                `json:"observaciones"`
	DeliverWarehouse        string                `json:"llevabodega"`
	PickupWarehouse         string                `json:"recogebodega"`
	CostCenter              string                `json:"centrocostos"`
	TotalProductValue       *string               `json:"totalvalorproducto,omitempty"`
	GenerateDocuments       string                `json:"generardocumentos"`
	GenerateBinaries        string                `json:"generarbinarios"`
	Units                   []DispatchUnit        `json:"unidades"`
	ServiceType             string                `json:"tiposervicio"`
	ReferenceDocuments      []DispatchDocument    `json:"documentosreferencia"`
}

// DispatchPickupRequest defines optional pickup-window values.
type DispatchPickupRequest struct {
	Number      string `json:"numero"`
	Date        string `json:"fecha"`
	WindowStart string `json:"ventanainicio"`
	WindowEnd   string `json:"ventanafin"`
}

// DispatchUnit defines TCC dispatch unit values.
type DispatchUnit struct {
	UnitType       string `json:"tipounidad"`
	PackageType    string `json:"tipoempaque"`
	PackageClass   string `json:"claseempaque"`
	Contains       string `json:"dicecontener"`
	RealWeightKG   string `json:"kilosreales"`
	DepthCM        string `json:"largo"`
	HeightCM       string `json:"alto"`
	WidthCM        string `json:"ancho"`
	VolumeWeightKG string `json:"pesovolumen"`
	DeclaredValue  string `json:"valormercancia"`
	Barcode        string `json:"codigobarras"`
	BagNumber      string `json:"numerobolsa"`
	References     string `json:"referencias"`
	InnerUnits     string `json:"unidadesinternas"`
}

// DispatchDocument defines TCC dispatch reference-document values.
type DispatchDocument struct {
	DocumentType   string `json:"tipodocumento"`
	DocumentNumber string `json:"numerodocumento"`
	DocumentDate   string `json:"fechadocumento"`
}

// DispatchResponse defines TCC dispatch response payload values.
type DispatchResponse struct {
	// ResultCode defines dispatch result-code values (grabardespacho8 field name).
	ResultCode string `json:"codigoresultado"`
	// Respuesta defines dispatch result-code values (grabardespacho7 field name).
	Respuesta string `json:"respuesta"`
	// ResultMessage defines dispatch result-message values (grabardespacho8 field name).
	ResultMessage string `json:"mensajeresultado"`
	// Mensaje defines dispatch result-message values (grabardespacho7 field name).
	Mensaje string `json:"mensaje"`
	// ShipmentNumber defines remittance number values (grabardespacho8 field name).
	ShipmentNumber string `json:"numeroremesa"`
	// Remesa defines remittance number values (grabardespacho7 field name).
	Remesa           string                     `json:"remesa"`
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
	Remittances        []TrackingRequestRemittance `json:"remesas"`
	ReferenceDocuments []TrackingRequestReference  `json:"documentosreferencia"`
	GenerateImage      bool                        `json:"generarimagen"`
}

// TrackingRequestRemittance defines remittance filters for tracking lookups.
type TrackingRequestRemittance struct {
	ShipmentNumber string `json:"numeroremesa"`
}

// TrackingRequestReference defines reference-document filters for tracking lookups.
type TrackingRequestReference struct {
	ReferenceDocument string `json:"documentoreferencia"`
}

// TrackingResponse defines TCC tracking response payload values.
type TrackingResponse struct {
	Remittances []TrackingRemittanceResponse `json:"remesas"`
	Result      TrackingResultResponse       `json:"respuesta"`
}

// TrackingState defines TCC tracking state payload values.
type TrackingState struct {
	Code        string `json:"codigo"`
	Description string `json:"descripcion"`
	Date        string `json:"fecha"`
}

// TrackingNovelty defines TCC incident payload values.
type TrackingNovelty struct {
	Code        string `json:"codigo"`
	Date        string `json:"fecha"`
	Description string `json:"descripcion"`
	State       string `json:"estado"`
	Observation string `json:"observacion"`
	IsRejection string `json:"esrechazo"`
	RejectedAt  string `json:"fecharechazo"`
}

// TrackingSingleNovelty defines the legacy singular novelty payload in TCC responses.
type TrackingSingleNovelty struct {
	Code           string `json:"idtiponovedad"`
	Date           string `json:"fechanovedad"`
	Description    string `json:"novedad"`
	Observation    string `json:"observaciones"`
	Complement     string `json:"complementonovedad"`
	State          string `json:"estadonovedad"`
	SolutionKinds  string `json:"tiposolucion"`
	ManagementType string `json:"tipogestion"`
}

// TrackingRemittanceResponse defines per-remittance tracking payload values.
type TrackingRemittanceResponse struct {
	ShipmentNumber string                `json:"numeroremesa"`
	States         []TrackingState       `json:"estados"`
	Novelty        TrackingSingleNovelty `json:"novedad"`
	Novelties      []TrackingNovelty     `json:"novedades"`
	OriginCity     trackingCityResponse  `json:"ciudadorigen"`
	DestCity       trackingCityResponse  `json:"ciudaddestino"`
}

// TrackingResultResponse defines TCC tracking response result values.
type TrackingResultResponse struct {
	Code    string `json:"codigo"`
	Message string `json:"mensaje"`
}

type trackingCityResponse struct {
	Description string `json:"descripcion"`
}

// BuildQuoteRequest maps domain quotation values into TCC quotation request values.
func BuildQuoteRequest(account string, request domain.QuotationRequest) QuoteRequest {
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
		OriginCityCode: NormalizeCityCode(normalized.OriginCityCode),
		DestCityCode:   NormalizeCityCode(normalized.DestCityCode),
		DeclaredValue:  normalized.DeclaredValue,
		Boomerang:      0,
		Identification: "",
		Account:        strings.TrimSpace(account),
		ShipmentDate:   time.Now().UTC().Format("2006-01-02"),
		BusinessUnitID: mapShipmentMode(normalized.ShipmentMode),
		Units:          units,
	}
}

// mapShipmentMode maps domain shipment-mode values to TCC business-unit integers.
func mapShipmentMode(mode domain.ShipmentMode) int {
	if mode == domain.ShipmentModeExpress {
		return 2
	}

	return 1
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
	case "4000":
		return domain.TrackingStatusReturn
	case "4100":
		return domain.TrackingStatusProcessing
	case "4200":
		return domain.TrackingStatusIncidence
	case "4300":
		return domain.TrackingStatusVoided
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

// MapTrackingNoveltyStatus maps TCC novelty payloads into normalized tracking statuses.
func MapTrackingNoveltyStatus(code string, description string) domain.TrackingStatus {
	switch strings.TrimSpace(code) {
	case "252":
		return domain.TrackingStatusIncidence
	}

	status := MapTrackingStatus(code, description)
	if status == domain.TrackingStatusProcessing || status == domain.TrackingStatusOrigin {
		lowerDescription := strings.ToLower(strings.TrimSpace(description))
		if strings.Contains(lowerDescription, "entregad") || strings.Contains(lowerDescription, "destinatario") {
			return domain.TrackingStatusIncidence
		}
		return domain.TrackingStatusIncidence
	}

	return status
}

// ParseTrackingDate parses TCC tracking date values.
func ParseTrackingDate(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Now().UTC()
	}
	normalized := strings.NewReplacer(
		"a. m.", "AM",
		"p. m.", "PM",
		"a.m.", "AM",
		"p.m.", "PM",
		"a. m", "AM",
		"p. m", "PM",
	).Replace(trimmed)
	normalized = strings.Join(strings.Fields(normalized), " ")
	layouts := []string{
		"2/01/2006 3:04:05 PM",
		"2/01/2006 3:04:05 AM",
		"2/01/2006 15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, normalized, colombiaLocation); err == nil {
			return parsed.UTC()
		}
	}

	return time.Now().UTC()
}

// NormalizeCityCode maps internal 5-digit city codes to TCC DANE8 format.
func NormalizeCityCode(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 5 && isNumeric(trimmed) {
		return trimmed + "000"
	}

	return trimmed
}

func max(value float64, minimum float64) float64 {
	if value < minimum {
		return minimum
	}

	return value
}

// BuildShipmentNumber parses response shipment number from possible field variants.
// As a last resort it extracts the "ti" query parameter from urlguia, which TCC
// encodes as https://somos.tcc.com.co/Informesdsp?opc=1&ti=REMESA_NUMBER.
func (r DispatchResponse) BuildShipmentNumber() string {
	if strings.TrimSpace(r.Remesa) != "" {
		return strings.TrimSpace(r.Remesa)
	}
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
	if trimmed := strings.TrimSpace(r.TrackingURL); trimmed != "" {
		if u, err := url.Parse(trimmed); err == nil {
			if ti := strings.TrimSpace(u.Query().Get("ti")); ti != "" {
				return ti
			}
		}
	}

	return ""
}

// ResolveResultCode returns the effective result code from either field variant.
func (r DispatchResponse) ResolveResultCode() string {
	if strings.TrimSpace(r.ResultCode) != "" {
		return strings.TrimSpace(r.ResultCode)
	}

	return strings.TrimSpace(r.Respuesta)
}

// ResolveResultMessage returns the effective result message from either field variant.
func (r DispatchResponse) ResolveResultMessage() string {
	if strings.TrimSpace(r.ResultMessage) != "" {
		return strings.TrimSpace(r.ResultMessage)
	}

	return strings.TrimSpace(r.Mensaje)
}

// BuildMarkDocumentURL resolves the preferred TCC shipping-mark URL from response fields.
// Fields are checked in priority order; non-URL values (plain IDs, empty strings) are skipped.
func (r DispatchResponse) BuildMarkDocumentURL() string {
	candidates := []string{
		r.ShipmentLabelURL,
		r.TrackingURL,
		r.LabelURL,
		r.ShipmentDocURL,
		r.InvoiceURL,
	}
	for _, c := range candidates {
		trimmed := strings.TrimSpace(c)
		if strings.HasPrefix(trimmed, "http") {
			return trimmed
		}
	}

	return ""
}

// BuildManifestURL resolves the preferred TCC shipping-manifest URL from response fields.
func (r DispatchResponse) BuildManifestURL() string {
	trimmed := strings.TrimSpace(r.ShipmentRelation)
	if strings.HasPrefix(trimmed, "http") {
		return trimmed
	}

	return ""
}

// BuildTrackURL resolves the preferred TCC shipping-mark URL from response fields.
// Deprecated: use BuildMarkDocumentURL instead.
func (r DispatchResponse) BuildTrackURL() string {
	return r.BuildMarkDocumentURL()
}

// ParseResultCode parses TCC result-code values.
// An empty code is treated as 0 (success); TCC returns HTTP 200 with no
// top-level codigoresultado when the dispatch succeeds and puts per-remittance
// results in the Remittances array.
func ParseResultCode(code string) int {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return 0
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return -1
	}

	return parsed
}

func isNumeric(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}
