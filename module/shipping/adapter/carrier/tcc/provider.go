package tcc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"mannaiah/module/shipping/domain"
)

const (
	// defaultDispatchUnitDeclaredValueCOP defines fallback declared-value amounts used when one dispatch unit has no value.
	defaultDispatchUnitDeclaredValueCOP = 10000.0
)

// ProviderConfig defines TCC provider configuration values.
type ProviderConfig struct {
	// Enabled defines whether TCC provider wiring should be active.
	Enabled bool
	// IsSandbox defines whether TCC sandbox endpoints should be used.
	IsSandbox bool
	// BaseURLOverride defines optional base URL override values for local tests.
	BaseURLOverride string
	// AccessToken defines TCC access-token values.
	AccessToken string
	// ParcelAccountNumber defines sender account-number used for parcel (standard) shipments.
	ParcelAccountNumber string
	// ExpressAccountNumber defines sender account-number used for express shipments.
	ExpressAccountNumber string
	// Declaration defines the default unit contents description (dicecontener) when none is provided.
	Declaration string
	// Sender defines fallback sender address values.
	Sender domain.Address
	// PaymentForm defines TCC payment-form values.
	PaymentForm int
	// CODFeePercent defines COD fee percentage applied to collected values.
	CODFeePercent float64
	// RequestTimeout defines outbound request timeout values.
	RequestTimeout time.Duration
}

// Provider defines TCC carrier provider behavior.
type Provider struct {
	// client defines TCC HTTP client dependencies.
	client *Client
	// cfg defines static provider configuration values.
	cfg ProviderConfig
}

// NewProvider creates TCC providers.
func NewProvider(config ProviderConfig) (*Provider, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("tcc provider is disabled")
	}
	if config.PaymentForm <= 0 {
		config.PaymentForm = 1
	}
	client, err := NewClient(ClientConfig{
		IsSandbox:       config.IsSandbox,
		BaseURLOverride: config.BaseURLOverride,
		AccessToken:     config.AccessToken,
		RequestTimeout:  config.RequestTimeout,
	})
	if err != nil {
		return nil, err
	}

	return &Provider{client: client, cfg: config}, nil
}

// CarrierID returns TCC carrier identifiers.
func (p *Provider) CarrierID() string {
	return "tcc"
}

// Carrier returns TCC carrier descriptors.
func (p *Provider) Carrier() domain.Carrier {
	return domain.Carrier{
		ID:                   "tcc",
		Name:                 "TCC",
		Type:                 domain.CarrierTypeAPI,
		Active:               true,
		RequiresBalanceCheck: false,
		HasQuotation:         true,
		HasManifestDocument:  true,
		HasTracking:          true,
		NeedsURL:             false,
	}
}

// SupportsQuotation reports quotation support for TCC providers.
func (p *Provider) SupportsQuotation() bool {
	return true
}

// CheckBalance validates TCC account balance.
func (p *Provider) CheckBalance(ctx context.Context) error {
	return nil
}

// Quote retrieves quotation results from TCC APIs.
func (p *Provider) Quote(ctx context.Context, request domain.QuotationRequest) (*domain.QuotationResult, error) {
	if p == nil || p.client == nil {
		return nil, domain.ErrCarrierNotSupported
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	payload := BuildQuoteRequest(p.resolveAccountNumber(request.ShipmentMode), request)
	response, requestBody, responseBody, err := p.client.QuoteRaw(ctx, payload)
	if err != nil {
		return nil, err
	}
	if ParseResultCode(response.ResultCode) != 0 {
		msg := strings.TrimSpace(response.ResultMessage)
		zap.L().Error("tcc quotation rejected",
			zap.String("result_code", response.ResultCode),
			zap.String("result_message", msg),
			zap.String("origin_city", request.OriginCityCode),
			zap.String("dest_city", request.DestCityCode),
		)
		return nil, fmt.Errorf("tcc quotation rejected: %s", msg)
	}
	result := response.ToDomain(p.CarrierID(), request)
	collectOnDeliveryAmount := max(request.Normalize().CollectOnDeliveryAmount, 0)
	collectOnDeliveryFeePercent := normalizePercent(p.cfg.CODFeePercent)
	collectOnDeliveryChargedAmount := calculateCollectOnDeliveryChargedAmount(collectOnDeliveryAmount, collectOnDeliveryFeePercent)
	result.CollectOnDeliveryAmount = collectOnDeliveryAmount
	result.CollectOnDeliveryFeePercent = collectOnDeliveryFeePercent
	result.CollectOnDeliveryFeeAmount = max(collectOnDeliveryChargedAmount-collectOnDeliveryAmount, 0)
	result.CollectOnDeliveryChargedAmount = collectOnDeliveryChargedAmount
	result.RequestSnapshot = encodePayloadSnapshot(requestBody)
	result.RawResponse = strings.TrimSpace(string(responseBody))

	return result, nil
}

// GenerateMark creates shipping marks via TCC dispatch endpoints.
func (p *Provider) GenerateMark(ctx context.Context, mark *domain.ShippingMark) error {
	if p == nil || p.client == nil {
		return domain.ErrCarrierNotSupported
	}
	if mark == nil {
		return domain.ErrInvalidID
	}
	resolved := mark.Normalize()
	sender := p.cfg.Sender.Normalize()
	if resolved.Sender.Name != "" {
		sender = resolved.Sender
	}
	recipient := resolved.Recipient.Normalize()
	collectOnDeliveryAmount := max(resolved.CollectOnDeliveryAmount, 0)
	collectOnDeliveryFeePercent := normalizePercent(resolved.CollectOnDeliveryFeePercent)
	collectOnDeliveryChargedAmount := max(resolved.CollectOnDeliveryChargedAmount, collectOnDeliveryAmount)
	units := make([]DispatchUnit, 0, len(resolved.Units))
	for _, unit := range resolved.Units {
		normalized := unit.Normalize()
		unitDeclaredValue := normalizeDispatchDeclaredValue(normalized.Dimensions.DeclaredValueCOP)
		units = append(units, DispatchUnit{
			UnitType:       "TIPO_UND_PAQ",
			PackageType:    "",
			PackageClass:   "CLEM_CAJA",
			Contains:       fallback(normalized.Description, p.cfg.Declaration),
			RealWeightKG:   FormatFloatString(max(normalized.Dimensions.RealWeightKG, 1)),
			DepthCM:        FormatFloatString(normalized.Dimensions.DepthCM),
			HeightCM:       FormatFloatString(normalized.Dimensions.HeightCM),
			WidthCM:        FormatFloatString(normalized.Dimensions.WidthCM),
			VolumeWeightKG: FormatFloatString(max(normalized.Dimensions.VolumetricWeightKG, 1)),
			DeclaredValue:  FormatFloatString(unitDeclaredValue),
			Barcode:        "",
			BagNumber:      "",
			References:     "",
			InnerUnits:     "0",
		})
	}
	// TODO(rework): TCC COD payment form logic.
	// TCC requires formapago=2 when collecting cash on delivery, and expects the
	// courier to collect both the freight cost AND the COD amount from the recipient.
	// Since we want the recipient to pay only the COD total (not freight + COD), we
	// subtract the quoted freight cost from the mark's stored chargedAmount so that TCC
	// collects (freight + netCOD) == (freight + (codCharged - freight)) == codCharged.
	// collectOnDeliveryChargedAmount is taken from the mark's stored field (set at draft/quote
	// time); the provider config CODFeePercent is NOT re-applied here to avoid double-charging.
	// Example: COD $150 000, freight quote $25 000 → netCOD = $125 000 sent to TCC;
	// courier collects $25 000 + $125 000 = $150 000 (the original COD total).
	// When COD is not active, formapago uses the configured value and COD-only payload keys
	// (recaudoproducto/totalvalorproducto) are omitted completely.
	paymentForm := strconv.Itoa(p.cfg.PaymentForm)
	var codCollectStr string
	var codCollectValue *string
	var totalProductValue *string
	if collectOnDeliveryChargedAmount > 0 {
		paymentForm = "2"
		codCollectStr = FormatFloatString(max(collectOnDeliveryChargedAmount-resolved.QuotedFreightCost, 0))
		codCollectValue = formatDispatchCODAmountPointer(codCollectStr)
		totalProductValue = formatDispatchCODAmountPointer(codCollectStr)
	}
	request := DispatchRequest{
		RelationNumber:          "",
		RelationDateTime:        "",
		PickupRequest:           DispatchPickupRequest{},
		BusinessUnit:            strconv.Itoa(mapShipmentMode(mark.ShipmentMode)),
		ShipmentNumber:          "",
		DispatchDate:            time.Now().UTC().Format("2006-01-02"),
		SenderAccount:           p.resolveAccountNumber(mark.ShipmentMode),
		SenderIDType:            sender.IDType,
		SenderID:                sender.ID,
		SenderBranch:            "",
		SenderFirstName:         "",
		SenderSecondName:        "",
		SenderFirstLastName:     splitName(sender.Name, 0),
		SenderSecondLastName:    splitName(sender.Name, 1),
		SenderCompanyName:       sender.Name,
		SenderNature:            "N",
		SenderAddress:           sender.AddressLine,
		SenderContact:           sender.Name,
		SenderEmail:             sender.Email,
		SenderPhone:             sender.Phone,
		OriginCityCode:          NormalizeCityCode(sender.CityCode),
		RecipientIDType:         recipient.IDType,
		RecipientID:             recipient.ID,
		RecipientBranch:         "",
		RecipientFirstName:      splitName(recipient.Name, 0),
		RecipientSecondName:     splitName(recipient.Name, 1),
		RecipientFirstLast:      splitName(recipient.Name, 2),
		RecipientSecondLast:     splitName(recipient.Name, 3),
		RecipientCompany:        recipient.LegalName,
		RecipientNature:         "N",
		RecipientAddress:        recipient.AddressLine,
		RecipientContact:        recipient.Phone,
		RecipientEmail:          recipient.Email,
		RecipientPhone:          recipient.Phone,
		DestCityCode:            NormalizeCityCode(recipient.CityCode),
		RecipientNeighborhood:   "",
		TotalWeight:             FormatFloatString(resolved.TotalWeight),
		TotalVolumeWeight:       FormatFloatString(resolved.TotalVolumetricWeight),
		PaymentForm:             paymentForm,
		CollectOnDeliveryAmount: codCollectValue,
		Observations:            resolved.Observations,
		DeliverWarehouse:        "",
		PickupWarehouse:         "",
		CostCenter:              "",
		TotalProductValue:       totalProductValue,
		GenerateDocuments:       "true",
		GenerateBinaries:        "false",
		Units:                   units,
		ServiceType:             "",
		ReferenceDocuments:      []DispatchDocument{},
	}
	if err := validateDispatchGuardrails(resolved, request); err != nil {
		requestBody, marshalErr := json.Marshal(request)
		if marshalErr == nil {
			mark.DraftSnapshot = encodePayloadSnapshot(requestBody)
		}
		return err
	}
	response, requestBody, responseBody, err := p.client.DispatchRaw(ctx, request)
	mark.DraftSnapshot = encodePayloadSnapshot(requestBody)
	mark.ResponseSnapshot = encodePayloadSnapshot(responseBody)
	if err != nil {
		return err
	}
	if ParseResultCode(response.ResolveResultCode()) != 0 {
		msg := response.ResolveResultMessage()
		remittanceMsg := ""
		if len(response.Remittances) > 0 {
			remittanceMsg = response.Remittances[0].ResultMessage
		}
		zap.L().Error("tcc dispatch rejected",
			zap.String("result_code", response.ResolveResultCode()),
			zap.String("result_message", msg),
			zap.String("remittance_message", remittanceMsg),
			zap.String("order_id", resolved.OrderID),
			zap.String("mark_id", resolved.ID),
			zap.String("origin_city", NormalizeCityCode(sender.CityCode)),
			zap.String("dest_city", NormalizeCityCode(recipient.CityCode)),
		)
		if msg == "" && remittanceMsg != "" {
			msg = remittanceMsg
		}
		return fmt.Errorf("tcc dispatch rejected: %s", msg)
	}
	tracking := response.BuildShipmentNumber()
	if tracking == "" {
		tracking = response.ResolveResultMessage()
	}
	markDocumentURL := response.BuildMarkDocumentURL()
	if strings.TrimSpace(markDocumentURL) == "" {
		return fmt.Errorf("tcc dispatch rejected: shipping mark document URL is missing")
	}
	manifestURL := response.BuildManifestURL()

	mark.TrackingNumber = strings.TrimSpace(tracking)
	mark.DocumentType = domain.MarkDocumentLink
	mark.DocumentRef = markDocumentURL
	mark.ManifestType = ""
	mark.ManifestRef = ""
	if strings.TrimSpace(manifestURL) != "" {
		mark.ManifestType = domain.MarkDocumentLink
		mark.ManifestRef = manifestURL
	}
	mark.Status = domain.MarkStatusGenerated
	mark.TotalWeight = resolved.TotalWeight
	mark.TotalVolumetricWeight = resolved.TotalVolumetricWeight
	mark.DeclaredValue = resolved.DeclaredValue
	mark.CollectOnDeliveryAmount = collectOnDeliveryAmount
	mark.CollectOnDeliveryFeePercent = collectOnDeliveryFeePercent
	mark.CollectOnDeliveryChargedAmount = collectOnDeliveryChargedAmount
	mark.Sender = sender
	mark.Recipient = recipient
	mark.Units = resolved.Units
	mark.UpdatedAt = time.Now().UTC()

	return nil
}

func encodePayloadSnapshot(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}

	return base64.StdEncoding.EncodeToString(payload)
}

// VoidMark voids marks in TCC providers.
func (p *Provider) VoidMark(ctx context.Context, trackingNumber string) error {
	return nil
}

// SupportsCourier reports whether TCC providers support one carrier identifier.
func (p *Provider) SupportsCourier(carrierID string) bool {
	return strings.EqualFold(strings.TrimSpace(carrierID), p.CarrierID())
}

// GetTrackingHistory requests tracking details from TCC APIs.
func (p *Provider) GetTrackingHistory(ctx context.Context, trackingNumber string) (*domain.TrackingHistory, error) {
	trimmedTracking := strings.TrimSpace(trackingNumber)
	if trimmedTracking == "" {
		return nil, domain.ErrInvalidID
	}
	response, err := p.client.Track(ctx, TrackingRequest{
		Remittances: []TrackingRequestRemittance{
			{ShipmentNumber: trimmedTracking},
		},
		ReferenceDocuments: []TrackingRequestReference{
			{ReferenceDocument: ""},
		},
		GenerateImage: false,
	})
	if err != nil {
		return nil, err
	}
	if ParseResultCode(response.Result.Code) != 0 {
		msg := strings.TrimSpace(response.Result.Message)
		zap.L().Error("tcc tracking rejected",
			zap.String("result_code", response.Result.Code),
			zap.String("result_message", msg),
			zap.String("tracking_number", trimmedTracking),
		)
		return nil, fmt.Errorf("tcc tracking rejected: %s", msg)
	}
	if len(response.Remittances) == 0 {
		return nil, fmt.Errorf("tcc tracking returned no remittance data")
	}
	remittance := response.Remittances[0]
	history := make([]domain.TrackingEvent, 0, len(remittance.States))
	latestStatus := domain.TrackingStatusProcessing
	latestDate := time.Time{}
	for _, state := range remittance.States {
		status := MapTrackingStatus(state.Code, state.Description)
		eventDate := ParseTrackingDate(state.Date)
		city := remittance.OriginCity.Description
		if status == domain.TrackingStatusCompleted {
			city = remittance.DestCity.Description
		}
		history = append(history, domain.TrackingEvent{
			Date:   eventDate,
			Code:   strings.TrimSpace(state.Code),
			Text:   strings.TrimSpace(state.Description),
			City:   strings.TrimSpace(city),
			Status: status,
		})
		if eventDate.After(latestDate) {
			latestDate = eventDate
			latestStatus = status
		}
	}
	if latestDate.IsZero() {
		latestDate = time.Now().UTC()
	}

	return &domain.TrackingHistory{
		CarrierID:      p.CarrierID(),
		TrackingNumber: trimmedTracking,
		GlobalStatus:   latestStatus,
		LastUpdate:     latestDate,
		History:        history,
	}, nil
}

func splitName(value string, index int) string {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 0 {
		return ""
	}
	if index >= len(parts) {
		return ""
	}

	return parts[index]
}

func fallback(value string, defaultValue string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultValue
	}

	return trimmed
}

// FormatFloatString formats float values as integer-friendly strings for references.
func FormatFloatString(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}

	return strconv.FormatFloat(value, 'f', 2, 64)
}

func normalizePercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}

	return math.Round(value*100) / 100
}

func (p *Provider) resolveAccountNumber(mode domain.ShipmentMode) string {
	if mode == domain.ShipmentModeExpress {
		return strings.TrimSpace(p.cfg.ExpressAccountNumber)
	}

	return strings.TrimSpace(p.cfg.ParcelAccountNumber)
}

func calculateCollectOnDeliveryChargedAmount(amount float64, feePercent float64) float64 {
	normalizedAmount := max(amount, 0)
	if normalizedAmount == 0 {
		return 0
	}
	normalizedFee := normalizePercent(feePercent)
	if normalizedFee == 0 {
		return math.Round(normalizedAmount*100) / 100
	}

	return math.Round((normalizedAmount*(1+(normalizedFee/100)))*100) / 100
}

// normalizeDispatchDeclaredValue applies fallback declared-value amounts for TCC dispatch requests.
func normalizeDispatchDeclaredValue(value float64) float64 {
	if value <= 0 {
		return defaultDispatchUnitDeclaredValueCOP
	}

	return value
}
