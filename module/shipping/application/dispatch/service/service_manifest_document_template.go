package service

import (
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"strings"
)

var (
	// errInvalidBatchManifestCoverTemplate reports invalid manifest cover template payload values.
	errInvalidBatchManifestCoverTemplate = errors.New("invalid batch manifest cover template")

	//go:embed templates/batch_manifest_cover.es.json
	defaultBatchManifestCoverTemplateJSON []byte
)

// batchManifestCoverTemplate defines user-facing strings rendered in summary cover pages.
type batchManifestCoverTemplate struct {
	// Title defines cover title values.
	Title string `json:"title"`
	// BatchIDLabel defines batch identifier label values.
	BatchIDLabel string `json:"batchIDLabel"`
	// GeneratedLabel defines generation timestamp label values.
	GeneratedLabel string `json:"generatedLabel"`
	// CarrierLabel defines carrier label values.
	CarrierLabel string `json:"carrierLabel"`
	// QuantityLabel defines row-count label values.
	QuantityLabel string `json:"quantityLabel"`
	// TrackingNumberHeader defines tracking-number column header values.
	TrackingNumberHeader string `json:"trackingNumberHeader"`
	// FreightHeader defines freight-cost column header values.
	FreightHeader string `json:"freightHeader"`
	// RecipientHeader defines recipient column header values.
	RecipientHeader string `json:"recipientHeader"`
	// OrderNumberHeader defines order-number column header values.
	OrderNumberHeader string `json:"orderNumberHeader"`
	// CityHeader defines city column header values.
	CityHeader string `json:"cityHeader"`
	// ItemsHeader defines items column header values.
	ItemsHeader string `json:"itemsHeader"`
	// ItemBulletPrefix defines unordered-list prefix values used in item rows.
	ItemBulletPrefix string `json:"itemBulletPrefix"`
	// EmptyValueFallback defines fallback text for empty values.
	EmptyValueFallback string `json:"emptyValueFallback"`
}

// loadDefaultBatchManifestCoverTemplate resolves default embedded cover template values.
func loadDefaultBatchManifestCoverTemplate() batchManifestCoverTemplate {
	template, err := parseBatchManifestCoverTemplate(defaultBatchManifestCoverTemplateJSON)
	if err != nil {
		return batchManifestCoverTemplate{
			Title:                "RESUMEN DE MANIFIESTOS DEL LOTE",
			BatchIDLabel:         "Lote",
			GeneratedLabel:       "Generado",
			CarrierLabel:         "Transportadora",
			QuantityLabel:        "Cantidad",
			TrackingNumberHeader: "NÚMERO DE GUÍA",
			FreightHeader:        "FLETE",
			RecipientHeader:      "DESTINATARIO",
			OrderNumberHeader:    "PEDIDO #",
			CityHeader:           "CIUDAD",
			ItemsHeader:          "ARTÍCULOS",
			ItemBulletPrefix:     "- ",
			EmptyValueFallback:   "-",
		}
	}

	return template
}

// SetBatchManifestDocumentCoverTemplate configures cover template values from raw JSON payload.
func (s *Service) SetBatchManifestDocumentCoverTemplate(rawTemplate []byte) error {
	if s == nil || s.manifestDocuments == nil {
		return nil
	}
	template, err := parseBatchManifestCoverTemplate(rawTemplate)
	if err != nil {
		return err
	}
	s.manifestDocuments.coverTemplate = template

	return nil
}

// SetBatchManifestDocumentCoverTemplateFromFile loads and configures cover template values from a JSON file path.
func (s *Service) SetBatchManifestDocumentCoverTemplateFromFile(filePath string) error {
	trimmedPath := strings.TrimSpace(filePath)
	if trimmedPath == "" {
		return nil
	}
	rawTemplate, err := os.ReadFile(trimmedPath)
	if err != nil {
		return err
	}

	return s.SetBatchManifestDocumentCoverTemplate(rawTemplate)
}

// parseBatchManifestCoverTemplate parses and validates raw JSON template payload values.
func parseBatchManifestCoverTemplate(rawTemplate []byte) (batchManifestCoverTemplate, error) {
	if len(rawTemplate) == 0 {
		return batchManifestCoverTemplate{}, errInvalidBatchManifestCoverTemplate
	}

	var template batchManifestCoverTemplate
	if err := json.Unmarshal(rawTemplate, &template); err != nil {
		return batchManifestCoverTemplate{}, errInvalidBatchManifestCoverTemplate
	}
	template = template.normalize()
	if err := template.validate(); err != nil {
		return batchManifestCoverTemplate{}, err
	}

	return template, nil
}

// normalize trims template values and applies fallback defaults.
func (t batchManifestCoverTemplate) normalize() batchManifestCoverTemplate {
	t.Title = strings.TrimSpace(t.Title)
	t.BatchIDLabel = strings.TrimSpace(t.BatchIDLabel)
	t.GeneratedLabel = strings.TrimSpace(t.GeneratedLabel)
	t.CarrierLabel = strings.TrimSpace(t.CarrierLabel)
	t.QuantityLabel = strings.TrimSpace(t.QuantityLabel)
	t.TrackingNumberHeader = strings.TrimSpace(t.TrackingNumberHeader)
	t.FreightHeader = strings.TrimSpace(t.FreightHeader)
	t.RecipientHeader = strings.TrimSpace(t.RecipientHeader)
	t.OrderNumberHeader = strings.TrimSpace(t.OrderNumberHeader)
	t.CityHeader = strings.TrimSpace(t.CityHeader)
	t.ItemsHeader = strings.TrimSpace(t.ItemsHeader)
	t.ItemBulletPrefix = strings.TrimSpace(t.ItemBulletPrefix)
	t.EmptyValueFallback = strings.TrimSpace(t.EmptyValueFallback)
	if t.ItemBulletPrefix == "" {
		t.ItemBulletPrefix = "- "
	}
	if t.EmptyValueFallback == "" {
		t.EmptyValueFallback = "-"
	}

	return t
}

// validate ensures all required template fields are present.
func (t batchManifestCoverTemplate) validate() error {
	requiredValues := []string{
		t.Title,
		t.BatchIDLabel,
		t.GeneratedLabel,
		t.CarrierLabel,
		t.QuantityLabel,
		t.TrackingNumberHeader,
		t.FreightHeader,
		t.RecipientHeader,
		t.OrderNumberHeader,
		t.CityHeader,
		t.ItemsHeader,
	}
	for _, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			return errInvalidBatchManifestCoverTemplate
		}
	}

	return nil
}
