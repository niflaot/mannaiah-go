package service

import (
	"errors"
	"fmt"
	"strings"

	"mannaiah/module/falabella/port"
)

var (
	// ErrSKURequired is returned when product SKU values are missing.
	ErrSKURequired = errors.New("product sku is required")
	// ErrNameRequired is returned when falabella product names are missing.
	ErrNameRequired = errors.New("falabella product name is required")
)

const (
	// defaultFalabellaBrand defines fallback Falabella brand values.
	defaultFalabellaBrand = "GENERIC"
)

// mapProduct maps catalog products into Falabella sync payload values.
func mapProduct(product port.CatalogProduct, cfg Config) (port.SyncProductRequest, string, error) {
	trimmedSKU := strings.TrimSpace(product.SKU)
	if trimmedSKU == "" {
		return port.SyncProductRequest{}, "", ErrSKURequired
	}

	datasheet, ok := findDatasheetByRealm(product.Datasheets, cfg.Realm)
	if !ok {
		return port.SyncProductRequest{}, "missing_falabella_realm", nil
	}

	attributes := toStringMap(datasheet.Attributes)
	name := firstNonEmpty(attributes["name"], datasheet.Name)
	if strings.TrimSpace(name) == "" {
		return port.SyncProductRequest{}, "", ErrNameRequired
	}

	description := firstNonEmpty(attributes["description"], datasheet.Description)
	request := port.SyncProductRequest{
		SKU:             trimmedSKU,
		Name:            strings.TrimSpace(name),
		Brand:           firstNonEmpty(attributes["brand"], defaultFalabellaBrand),
		Model:           strings.TrimSpace(attributes["model"]),
		Description:     strings.TrimSpace(description),
		PrimaryCategory: strings.TrimSpace(cfg.CategoryID),
		TaxClass:        strings.TrimSpace(attributes["tax_percentage"]),
		Price:           strings.TrimSpace(attributes["price_falabella"]),
		SalePrice:       strings.TrimSpace(attributes["sale_price_falabella"]),
		SaleStartDate:   strings.TrimSpace(attributes["sale_start_date_falabella"]),
		SaleEndDate:     strings.TrimSpace(attributes["sale_end_date_falabella"]),
		Attributes:      attributes,
	}

	if request.Attributes == nil {
		request.Attributes = map[string]string{}
	}
	if trimmed := strings.TrimSpace(cfg.GlobalIdentifier); trimmed != "" {
		request.Attributes["global_identifier"] = trimmed
	}
	if trimmed := strings.TrimSpace(cfg.AttributeSetID); trimmed != "" {
		request.Attributes["attribute_set_id"] = trimmed
	}
	if trimmed := strings.TrimSpace(cfg.CategoryID); trimmed != "" {
		request.Attributes["category_id"] = trimmed
	}
	if request.Brand == "" {
		request.Brand = defaultFalabellaBrand
	}

	return request, "", nil
}

// findDatasheetByRealm resolves datasheets for configured realm values.
func findDatasheetByRealm(datasheets []port.CatalogDatasheet, realm string) (port.CatalogDatasheet, bool) {
	trimmedRealm := strings.TrimSpace(realm)
	for _, datasheet := range datasheets {
		if strings.EqualFold(strings.TrimSpace(datasheet.Realm), trimmedRealm) {
			return datasheet, true
		}
	}

	return port.CatalogDatasheet{}, false
}

// toStringMap converts generic attribute maps into string-key/string-value maps.
func toStringMap(attributes map[string]any) map[string]string {
	if len(attributes) == 0 {
		return map[string]string{}
	}

	mapped := make(map[string]string, len(attributes))
	for key, value := range attributes {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || value == nil {
			continue
		}
		mapped[trimmedKey] = strings.TrimSpace(fmt.Sprint(value))
	}

	return mapped
}

// firstNonEmpty resolves the first non-empty value from provided candidates.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

