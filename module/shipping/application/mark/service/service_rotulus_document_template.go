package service

import (
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"strings"
)

var (
	// errInvalidRotulusTemplate reports invalid rotulus template payload values.
	errInvalidRotulusTemplate = errors.New("invalid rotulus template")

	//go:embed templates/rotulus.es.json
	defaultRotulusTemplateJSON []byte
)

// markRotulusTemplate defines user-facing strings rendered in rotulus PDFs.
type markRotulusTemplate struct {
	// OrderTitlePrefix defines document title prefix values.
	OrderTitlePrefix string `json:"orderTitlePrefix"`
	// OrderLabel defines order identifier label values.
	OrderLabel string `json:"orderLabel"`
	// TrackingLabel defines tracking identifier label values.
	TrackingLabel string `json:"trackingLabel"`
	// CarrierLabel defines carrier label values.
	CarrierLabel string `json:"carrierLabel"`
	// RecipientLabel defines recipient label values.
	RecipientLabel string `json:"recipientLabel"`
	// AddressLabel defines address label values.
	AddressLabel string `json:"addressLabel"`
	// Address2Label defines address-line-2 label values.
	Address2Label string `json:"address2Label"`
	// PhoneLabel defines phone label values.
	PhoneLabel string `json:"phoneLabel"`
	// CityLabel defines city label values.
	CityLabel string `json:"cityLabel"`
	// FooterLabel defines footer timestamp label values.
	FooterLabel string `json:"footerLabel"`
	// EmptyValueFallback defines fallback values for empty fields.
	EmptyValueFallback string `json:"emptyValueFallback"`
}

// loadDefaultRotulusTemplate resolves default embedded rotulus template values.
func loadDefaultRotulusTemplate() markRotulusTemplate {
	template, err := parseRotulusTemplate(defaultRotulusTemplateJSON)
	if err != nil {
		return markRotulusTemplate{
			OrderTitlePrefix:   "Pedido #",
			OrderLabel:         "Pedido",
			TrackingLabel:      "Guía",
			CarrierLabel:       "Transportadora",
			RecipientLabel:     "Destinatario",
			AddressLabel:       "Dirección",
			Address2Label:      "Dirección 2",
			PhoneLabel:         "Teléfono",
			CityLabel:          "Ciudad",
			FooterLabel:        "Emitido",
			EmptyValueFallback: "-",
		}
	}

	return template
}

// SetRotulusDocumentTemplate configures rotulus template values from raw JSON payload.
func (s *Service) SetRotulusDocumentTemplate(rawTemplate []byte) error {
	if s == nil || s.rotulusDocuments == nil {
		return nil
	}
	template, err := parseRotulusTemplate(rawTemplate)
	if err != nil {
		return err
	}
	s.rotulusDocuments.template = template

	return nil
}

// SetRotulusDocumentTemplateFromFile loads and configures rotulus template values from a JSON file path.
func (s *Service) SetRotulusDocumentTemplateFromFile(filePath string) error {
	trimmedPath := strings.TrimSpace(filePath)
	if trimmedPath == "" {
		return nil
	}
	rawTemplate, err := os.ReadFile(trimmedPath)
	if err != nil {
		return err
	}

	return s.SetRotulusDocumentTemplate(rawTemplate)
}

// parseRotulusTemplate parses and validates raw JSON template payload values.
func parseRotulusTemplate(rawTemplate []byte) (markRotulusTemplate, error) {
	if len(rawTemplate) == 0 {
		return markRotulusTemplate{}, errInvalidRotulusTemplate
	}

	var template markRotulusTemplate
	if err := json.Unmarshal(rawTemplate, &template); err != nil {
		return markRotulusTemplate{}, errInvalidRotulusTemplate
	}
	template = template.normalize()
	if err := template.validate(); err != nil {
		return markRotulusTemplate{}, err
	}

	return template, nil
}

// normalize trims template values and applies fallback defaults.
func (t markRotulusTemplate) normalize() markRotulusTemplate {
	t.OrderTitlePrefix = strings.TrimSpace(t.OrderTitlePrefix)
	t.OrderLabel = strings.TrimSpace(t.OrderLabel)
	t.TrackingLabel = strings.TrimSpace(t.TrackingLabel)
	t.CarrierLabel = strings.TrimSpace(t.CarrierLabel)
	t.RecipientLabel = strings.TrimSpace(t.RecipientLabel)
	t.AddressLabel = strings.TrimSpace(t.AddressLabel)
	t.Address2Label = strings.TrimSpace(t.Address2Label)
	t.PhoneLabel = strings.TrimSpace(t.PhoneLabel)
	t.CityLabel = strings.TrimSpace(t.CityLabel)
	t.FooterLabel = strings.TrimSpace(t.FooterLabel)
	t.EmptyValueFallback = strings.TrimSpace(t.EmptyValueFallback)
	if t.OrderTitlePrefix == "" {
		t.OrderTitlePrefix = "Pedido #"
	}
	if t.OrderLabel == "" {
		t.OrderLabel = "Pedido"
	}
	if t.TrackingLabel == "" {
		t.TrackingLabel = "Guía"
	}
	if t.CarrierLabel == "" {
		t.CarrierLabel = "Transportadora"
	}
	if t.RecipientLabel == "" {
		t.RecipientLabel = "Destinatario"
	}
	if t.AddressLabel == "" {
		t.AddressLabel = "Dirección"
	}
	if t.Address2Label == "" {
		t.Address2Label = "Dirección 2"
	}
	if t.PhoneLabel == "" {
		t.PhoneLabel = "Teléfono"
	}
	if t.CityLabel == "" {
		t.CityLabel = "Ciudad"
	}
	if t.FooterLabel == "" {
		t.FooterLabel = "Emitido"
	}
	if t.EmptyValueFallback == "" {
		t.EmptyValueFallback = "-"
	}

	return t
}

// validate ensures all required template fields are present.
func (t markRotulusTemplate) validate() error {
	requiredValues := []string{
		t.OrderTitlePrefix,
		t.OrderLabel,
		t.TrackingLabel,
		t.CarrierLabel,
		t.RecipientLabel,
		t.AddressLabel,
		t.Address2Label,
		t.PhoneLabel,
		t.CityLabel,
		t.FooterLabel,
	}
	for _, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			return errInvalidRotulusTemplate
		}
	}

	return nil
}
