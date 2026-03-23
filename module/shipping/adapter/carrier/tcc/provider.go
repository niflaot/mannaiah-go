package tcc

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"mannaiah/module/shipping/domain"
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
	// AccountNumber defines sender account-number values.
	AccountNumber string
	// Sender defines fallback sender address values.
	Sender domain.Address
	// BusinessUnit defines TCC business-unit identifier values.
	BusinessUnit int
	// PaymentForm defines TCC payment-form values.
	PaymentForm int
	// CODDiscountPercent defines COD surcharge percentage applied to collected values.
	CODDiscountPercent float64
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
	if config.BusinessUnit <= 0 {
		config.BusinessUnit = 1
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
	payload := BuildQuoteRequest(p.cfg.AccountNumber, p.cfg.BusinessUnit, request)
	response, err := p.client.Quote(ctx, payload)
	if err != nil {
		return nil, err
	}
	if ParseResultCode(response.ResultCode) != 0 {
		return nil, fmt.Errorf("tcc quotation rejected: %s", strings.TrimSpace(response.ResultMessage))
	}

	return response.ToDomain(p.CarrierID(), request), nil
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
	collectOnDeliveryDiscountPercent := normalizePercent(p.cfg.CODDiscountPercent)
	collectOnDeliveryChargedAmount := calculateCollectOnDeliveryChargedAmount(collectOnDeliveryAmount, collectOnDeliveryDiscountPercent)
	units := make([]DispatchUnit, 0, len(resolved.Units))
	for _, unit := range resolved.Units {
		normalized := unit.Normalize()
		units = append(units, DispatchUnit{
			UnitType:       "TIPO_UND_PAQ",
			PackageType:    "",
			PackageClass:   fallback(normalized.PackageType, "CLEM_CAJA"),
			Contains:       normalized.Description,
			RealWeightKG:   FormatFloatString(max(normalized.Dimensions.RealWeightKG, 1)),
			DepthCM:        FormatFloatString(normalized.Dimensions.DepthCM),
			HeightCM:       FormatFloatString(normalized.Dimensions.HeightCM),
			WidthCM:        FormatFloatString(normalized.Dimensions.WidthCM),
			VolumeWeightKG: FormatFloatString(max(normalized.Dimensions.VolumetricWeightKG, 1)),
			DeclaredValue:  FormatFloatString(max(normalized.Dimensions.DeclaredValueCOP, 0)),
			Barcode:        "",
			BagNumber:      "",
			References:     "",
			InnerUnits:     "0",
		})
	}
	request := DispatchRequest{
		RelationNumber:          "",
		RelationDateTime:        "",
		PickupRequest:           DispatchPickupRequest{},
		BusinessUnit:            strconv.Itoa(p.cfg.BusinessUnit),
		ShipmentNumber:          "",
		DispatchDate:            time.Now().UTC().Format("2006-01-02"),
		SenderAccount:           strings.TrimSpace(p.cfg.AccountNumber),
		SenderIDType:            sender.IDType,
		SenderID:                sender.ID,
		SenderBranch:            "",
		SenderFirstName:         splitName(sender.Name, 0),
		SenderSecondName:        splitName(sender.Name, 1),
		SenderFirstLastName:     splitName(sender.Name, 2),
		SenderSecondLastName:    splitName(sender.Name, 3),
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
		RecipientCompany:        recipient.Name,
		RecipientNature:         "N",
		RecipientAddress:        recipient.AddressLine,
		RecipientContact:        recipient.Name,
		RecipientEmail:          recipient.Email,
		RecipientPhone:          recipient.Phone,
		DestCityCode:            NormalizeCityCode(recipient.CityCode),
		RecipientNeighborhood:   "",
		TotalWeight:             FormatFloatString(resolved.TotalWeight),
		TotalVolumeWeight:       FormatFloatString(resolved.TotalVolumetricWeight),
		PaymentForm:             strconv.Itoa(p.cfg.PaymentForm),
		CollectOnDeliveryAmount: FormatFloatString(collectOnDeliveryChargedAmount),
		Observations:            resolved.Observations,
		DeliverWarehouse:        "",
		PickupWarehouse:         "",
		CostCenter:              "",
		TotalProductValue:       FormatFloatString(resolved.DeclaredValue),
		GenerateDocuments:       "true",
		GenerateBinaries:        "false",
		Units:                   units,
		ServiceType:             "TISE_NORMAL_PAQ",
		ReferenceDocuments: []DispatchDocument{{
			DocumentType:   "PE",
			DocumentNumber: fallback(resolved.OrderID, resolved.ID),
			DocumentDate:   time.Now().UTC().Format("2006-01-02"),
		}},
	}
	response, err := p.client.Dispatch(ctx, request)
	if err != nil {
		return err
	}
	if ParseResultCode(response.ResultCode) != 0 {
		return fmt.Errorf("tcc dispatch rejected: %s", strings.TrimSpace(response.ResultMessage))
	}
	tracking := response.BuildShipmentNumber()
	if tracking == "" {
		tracking = response.ResultMessage
	}
	mark.TrackingNumber = strings.TrimSpace(tracking)
	mark.DocumentType = domain.MarkDocumentLink
	mark.DocumentRef = response.BuildTrackURL()
	mark.Status = domain.MarkStatusGenerated
	mark.TotalWeight = resolved.TotalWeight
	mark.TotalVolumetricWeight = resolved.TotalVolumetricWeight
	mark.DeclaredValue = resolved.DeclaredValue
	mark.CollectOnDeliveryAmount = collectOnDeliveryAmount
	mark.CollectOnDeliveryDiscountPercent = collectOnDeliveryDiscountPercent
	mark.CollectOnDeliveryChargedAmount = collectOnDeliveryChargedAmount
	mark.Sender = sender
	mark.Recipient = recipient
	mark.Units = resolved.Units
	mark.UpdatedAt = time.Now().UTC()

	return nil
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
		return nil, fmt.Errorf("tcc tracking rejected: %s", strings.TrimSpace(response.Result.Message))
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

func calculateCollectOnDeliveryChargedAmount(amount float64, discountPercent float64) float64 {
	normalizedAmount := max(amount, 0)
	if normalizedAmount == 0 {
		return 0
	}
	normalizedDiscount := normalizePercent(discountPercent)
	if normalizedDiscount == 0 {
		return math.Round(normalizedAmount*100) / 100
	}

	return math.Round((normalizedAmount*(1+(normalizedDiscount/100)))*100) / 100
}
