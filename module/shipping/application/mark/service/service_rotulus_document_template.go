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
	// Title defines document title values.
	Title string `json:"title"`
	// OrderLabel defines order identifier label values.
	OrderLabel string `json:"orderLabel"`
	// TrackingLabel defines tracking identifier label values.
	TrackingLabel string `json:"trackingLabel"`
	// CarrierLabel defines carrier label values.
	CarrierLabel string `json:"carrierLabel"`
	// RecipientLabel defines recipient label values.
	RecipientLabel string `json:"recipientLabel"`
	// GeneratedLabel defines generated timestamp label values.
	GeneratedLabel string `json:"generatedLabel"`
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
			Title:              "Rótulo de despacho",
			OrderLabel:         "Pedido",
			TrackingLabel:      "Guía",
			CarrierLabel:       "Transportadora",
			RecipientLabel:     "Destinatario",
			GeneratedLabel:     "Generado",
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
	t.Title = strings.TrimSpace(t.Title)
	t.OrderLabel = strings.TrimSpace(t.OrderLabel)
	t.TrackingLabel = strings.TrimSpace(t.TrackingLabel)
	t.CarrierLabel = strings.TrimSpace(t.CarrierLabel)
	t.RecipientLabel = strings.TrimSpace(t.RecipientLabel)
	t.GeneratedLabel = strings.TrimSpace(t.GeneratedLabel)
	t.FooterLabel = strings.TrimSpace(t.FooterLabel)
	t.EmptyValueFallback = strings.TrimSpace(t.EmptyValueFallback)
	if t.EmptyValueFallback == "" {
		t.EmptyValueFallback = "-"
	}

	return t
}

// validate ensures all required template fields are present.
func (t markRotulusTemplate) validate() error {
	requiredValues := []string{
		t.Title,
		t.OrderLabel,
		t.TrackingLabel,
		t.CarrierLabel,
		t.RecipientLabel,
		t.GeneratedLabel,
		t.FooterLabel,
	}
	for _, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			return errInvalidRotulusTemplate
		}
	}

	return nil
}
