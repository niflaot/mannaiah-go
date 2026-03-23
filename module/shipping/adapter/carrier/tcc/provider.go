package tcc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mannaiah/module/shipping/domain"
)

// ProviderConfig defines TCC provider configuration values.
type ProviderConfig struct {
	// Enabled defines whether TCC provider wiring should be active.
	Enabled bool
	// BaseURL defines TCC base URL values.
	BaseURL string
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
		BaseURL:        config.BaseURL,
		AccessToken:    config.AccessToken,
		RequestTimeout: config.RequestTimeout,
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
	units := make([]DispatchUnit, 0, len(resolved.Units))
	for _, unit := range resolved.Units {
		normalized := unit.Normalize()
		units = append(units, DispatchUnit{
			UnitType:       "TIPO_UND_PAQ",
			PackageType:    "",
			PackageClass:   fallback(normalized.PackageType, "CLEM_CAJA"),
			Contains:       normalized.Description,
			RealWeightKG:   normalized.Dimensions.RealWeightKG,
			DepthCM:        normalized.Dimensions.DepthCM,
			HeightCM:       normalized.Dimensions.HeightCM,
			WidthCM:        normalized.Dimensions.WidthCM,
			VolumeWeightKG: normalized.Dimensions.VolumetricWeightKG,
			DeclaredValue:  normalized.Dimensions.DeclaredValueCOP,
			Barcode:        "",
			BagNumber:      "",
			References:     "",
			InnerUnits:     "0",
		})
	}
	request := DispatchRequest{
		DispatchNumber:       "",
		DispatchDate:         time.Now().UTC().Format("2006-01-02"),
		BusinessUnit:         p.cfg.BusinessUnit,
		SenderAccount:        strings.TrimSpace(p.cfg.AccountNumber),
		SenderBranch:         "",
		SenderFirstName:      sender.Name,
		SenderSecondName:     sender.Name,
		SenderFirstLastName:  sender.Name,
		SenderSecondLastName: sender.Name,
		SenderCompanyName:    sender.Name,
		SenderContact:        "",
		SenderIDType:         sender.IDType,
		SenderID:             sender.ID,
		SenderAddress:        sender.AddressLine,
		OriginCityCode:       sender.CityCode,
		SenderPhone:          sender.Phone,
		SenderEmail:          sender.Email,
		Recipients: []DispatchRecipient{{
			ControlNumber:           "1",
			ShipmentNumber:          "",
			ClientReferenceNumber:   "",
			RecipientIDType:         recipient.IDType,
			RecipientID:             recipient.ID,
			RecipientBranch:         "",
			RecipientFirstName:      splitName(recipient.Name, 0),
			RecipientSecondName:     splitName(recipient.Name, 1),
			RecipientFirstLastName:  splitName(recipient.Name, 2),
			RecipientSecondLastName: splitName(recipient.Name, 3),
			RecipientCompanyName:    recipient.Name,
			RecipientContact:        "",
			RecipientAddress:        recipient.AddressLine,
			RecipientPhone:          recipient.Phone,
			DestCityCode:            recipient.CityCode,
			PaymentForm:             p.cfg.PaymentForm,
			DeliverWarehouse:        "",
			PickupWarehouse:         "",
			CostCenter:              "",
			ServiceType:             "TISE_NORMAL_PAQ",
			Observations:            resolved.Observations,
			ProductCollection:       "0",
			Units:                   units,
		}},
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
	response, err := p.client.Track(ctx, TrackingRequest{ShipmentNumber: trimmedTracking})
	if err != nil {
		return nil, err
	}
	if ParseResultCode(response.ResultCode) != 0 {
		return nil, fmt.Errorf("tcc tracking rejected: %s", strings.TrimSpace(response.ResultMessage))
	}
	history := make([]domain.TrackingEvent, 0, len(response.States))
	latestStatus := domain.TrackingStatusProcessing
	latestDate := time.Time{}
	for _, state := range response.States {
		status := MapTrackingStatus(state.Code, state.Description)
		eventDate := ParseTrackingDate(state.Date)
		city := response.OriginCity.Description
		if status == domain.TrackingStatusCompleted {
			city = response.DestCity.Description
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
