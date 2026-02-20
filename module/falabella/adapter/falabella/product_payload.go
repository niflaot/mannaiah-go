package falabella

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"mannaiah/module/falabella/port"
)

var (
	// ErrProductOperatorCodeRequired is returned when operator-code values are missing.
	ErrProductOperatorCodeRequired = errors.New("falabella product operator code is required")
	// ErrProductUpdateBusinessUnitValuesRequired is returned when update payloads contain no mutable business-unit values.
	ErrProductUpdateBusinessUnitValuesRequired = errors.New("falabella product update requires at least one business-unit value")
	// ErrProductCreatePriceRequired is returned when create payloads omit required price values.
	ErrProductCreatePriceRequired = errors.New("falabella product create price is required")
	// ErrProductCreateStockRequired is returned when create payloads omit required stock values.
	ErrProductCreateStockRequired = errors.New("falabella product create stock is required")
	// ErrProductCreateStatusRequired is returned when create payloads omit required status values.
	ErrProductCreateStatusRequired = errors.New("falabella product create status is required")
)

const (
	// defaultProductOperatorCode defines fallback Falabella business-unit operator-code values.
	defaultProductOperatorCode = "FACO"
)

// businessUnitPayload defines Falabella business-unit request values.
type businessUnitPayload struct {
	// OperatorCode defines business-unit operator-code values.
	OperatorCode string
	// Price defines business-unit price values.
	Price string
	// SpecialPrice defines business-unit sale-price values.
	SpecialPrice string
	// SpecialFromDate defines business-unit sale start-date values.
	SpecialFromDate string
	// SpecialToDate defines business-unit sale end-date values.
	SpecialToDate string
	// Stock defines business-unit stock values.
	Stock string
	// Status defines business-unit status values.
	Status string
}

// buildProductRequestXML builds Falabella XML request bodies for ProductUpdate calls.
func buildProductRequestXML(request port.SyncProductRequest) ([]byte, error) {
	return buildProductRequestXMLWithMode(request, false)
}

// buildProductCreateRequestXML builds Falabella XML request bodies for ProductCreate calls.
func buildProductCreateRequestXML(request port.SyncProductRequest) ([]byte, error) {
	return buildProductRequestXMLWithMode(request, true)
}

// buildProductRequestXMLWithMode builds Falabella XML request bodies for ProductCreate/ProductUpdate calls.
func buildProductRequestXMLWithMode(request port.SyncProductRequest, requireCreateMinimum bool) ([]byte, error) {
	writer := &bytes.Buffer{}
	writer.WriteString(xml.Header)
	encoder := xml.NewEncoder(writer)
	canonicalAttributes := canonicalizeProductAttributes(request.Attributes)
	requiredAttributes := requiredProductAttributes(canonicalAttributes)
	businessUnit, err := resolveBusinessUnitPayload(request, canonicalAttributes, requireCreateMinimum)
	if err != nil {
		return nil, err
	}
	delete(canonicalAttributes, "BusinessUnits")
	delete(canonicalAttributes, "OperatorCode")
	delete(canonicalAttributes, "Stock")
	delete(canonicalAttributes, "Status")

	if err := writeStartElement(encoder, "Request"); err != nil {
		return nil, err
	}
	if err := writeStartElement(encoder, "Product"); err != nil {
		return nil, err
	}

	if err := writeOptionalElement(encoder, "SellerSku", request.SKU); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "ParentSku", request.ParentSKU); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "Variation", request.Variation); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "Name", request.Name); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "Brand", request.Brand); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "Model", request.Model); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "Description", request.Description); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "PrimaryCategory", request.PrimaryCategory); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "TaxClass", request.TaxClass); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "Color", requiredAttributes["Color"]); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "Talla", requiredAttributes["Talla"]); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "ColorBasico", requiredAttributes["ColorBasico"]); err != nil {
		return nil, err
	}
	if err := writeBusinessUnitsXML(encoder, businessUnit); err != nil {
		return nil, err
	}

	if len(canonicalAttributes) > 0 {
		if err := writeStartElement(encoder, "ProductData"); err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(canonicalAttributes))
		for key := range canonicalAttributes {
			if strings.TrimSpace(key) != "" {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := strings.TrimSpace(canonicalAttributes[key])
			if value == "" {
				continue
			}
			if err := writeOptionalElement(encoder, sanitizeXMLName(key), value); err != nil {
				return nil, err
			}
		}
		if err := writeEndElement(encoder, "ProductData"); err != nil {
			return nil, err
		}
	}

	if err := writeEndElement(encoder, "Product"); err != nil {
		return nil, err
	}
	if err := writeEndElement(encoder, "Request"); err != nil {
		return nil, err
	}
	if err := encoder.Flush(); err != nil {
		return nil, fmt.Errorf("flush xml encoder: %w", err)
	}

	return writer.Bytes(), nil
}

// buildProductRequestJSON builds Falabella JSON request bodies for ProductCreate/ProductUpdate calls.
func buildProductRequestJSON(request port.SyncProductRequest) ([]byte, error) {
	canonicalAttributes := canonicalizeProductAttributes(request.Attributes)
	requiredAttributes := requiredProductAttributes(canonicalAttributes)
	businessUnit, err := resolveBusinessUnitPayload(request, canonicalAttributes, false)
	if err != nil {
		return nil, err
	}
	delete(canonicalAttributes, "BusinessUnits")
	delete(canonicalAttributes, "OperatorCode")
	delete(canonicalAttributes, "Stock")
	delete(canonicalAttributes, "Status")

	product := map[string]any{}
	setOptionalJSONField(product, "SellerSku", request.SKU)
	setOptionalJSONField(product, "ParentSku", request.ParentSKU)
	setOptionalJSONField(product, "Variation", request.Variation)
	setOptionalJSONField(product, "Name", request.Name)
	setOptionalJSONField(product, "Brand", request.Brand)
	setOptionalJSONField(product, "Model", request.Model)
	setOptionalJSONField(product, "Description", request.Description)
	setOptionalJSONField(product, "PrimaryCategory", request.PrimaryCategory)
	setOptionalJSONField(product, "TaxClass", request.TaxClass)
	setOptionalJSONField(product, "Color", requiredAttributes["Color"])
	setOptionalJSONField(product, "Talla", requiredAttributes["Talla"])
	setOptionalJSONField(product, "ColorBasico", requiredAttributes["ColorBasico"])
	product["BusinessUnits"] = map[string]any{
		"BusinessUnit": map[string]any{
			"OperatorCode":    businessUnit.OperatorCode,
			"Price":           businessUnit.Price,
			"SpecialPrice":    businessUnit.SpecialPrice,
			"SpecialFromDate": businessUnit.SpecialFromDate,
			"SpecialToDate":   businessUnit.SpecialToDate,
			"Stock":           businessUnit.Stock,
			"Status":          businessUnit.Status,
		},
	}

	if len(canonicalAttributes) > 0 {
		productData := make(map[string]string, len(canonicalAttributes))
		for key, value := range canonicalAttributes {
			trimmedValue := strings.TrimSpace(value)
			if trimmedValue == "" {
				continue
			}
			productData[key] = trimmedValue
		}
		if len(productData) > 0 {
			product["ProductData"] = productData
		}
	}

	payload := map[string]any{
		"Request": map[string]any{
			"Product": product,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal product request json: %w", err)
	}

	return body, nil
}

// setOptionalJSONField sets JSON fields when values are non-empty.
func setOptionalJSONField(target map[string]any, key string, value string) {
	if target == nil {
		return
	}
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return
	}
	target[key] = trimmedValue
}

// canonicalizeProductAttributes resolves trimmed, canonicalized Falabella product attributes.
func canonicalizeProductAttributes(attributes map[string]string) map[string]string {
	if len(attributes) == 0 {
		return map[string]string{}
	}

	canonicalized := make(map[string]string, len(attributes))
	for key, value := range attributes {
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue
		}

		canonicalKey := canonicalProductAttributeName(key)
		if canonicalKey == "" {
			continue
		}
		if existing := strings.TrimSpace(canonicalized[canonicalKey]); existing != "" {
			continue
		}
		canonicalized[canonicalKey] = trimmedValue
	}

	if strings.TrimSpace(canonicalized["Color"]) == "" && strings.TrimSpace(canonicalized["ColorBasico"]) != "" {
		canonicalized["Color"] = strings.TrimSpace(canonicalized["ColorBasico"])
	}
	if strings.TrimSpace(canonicalized["ColorBasico"]) == "" && strings.TrimSpace(canonicalized["Color"]) != "" {
		canonicalized["ColorBasico"] = strings.TrimSpace(canonicalized["Color"])
	}

	return canonicalized
}

// requiredProductAttributes resolves required Falabella product-attribute values.
func requiredProductAttributes(attributes map[string]string) map[string]string {
	required := map[string]string{
		"BusinessUnits": strings.TrimSpace(attributes["BusinessUnits"]),
		"Color":         strings.TrimSpace(attributes["Color"]),
		"Talla":         strings.TrimSpace(attributes["Talla"]),
		"ColorBasico":   strings.TrimSpace(attributes["ColorBasico"]),
	}
	if required["Color"] == "" {
		required["Color"] = required["ColorBasico"]
	}
	if required["ColorBasico"] == "" {
		required["ColorBasico"] = required["Color"]
	}

	return required
}

// canonicalProductAttributeName resolves canonical Falabella attribute names.
func canonicalProductAttributeName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}

	switch normalizedProductAttributeToken(trimmed) {
	case "businessunits", "businessunit":
		return "BusinessUnits"
	case "operatorcode":
		return "OperatorCode"
	case "color":
		return "Color"
	case "talla", "size":
		return "Talla"
	case "stock", "stockfalabella", "quantity", "inventory":
		return "Stock"
	case "status":
		return "Status"
	case "colorbasico", "colorbase", "colorbasic", "basiccolor", "basecolor":
		return "ColorBasico"
	default:
		return trimmed
	}
}

// resolveBusinessUnitPayload resolves minimum Falabella BusinessUnit payload values.
func resolveBusinessUnitPayload(request port.SyncProductRequest, attributes map[string]string, requireCreateMinimum bool) (businessUnitPayload, error) {
	payload := businessUnitPayload{
		OperatorCode:    firstNonEmpty(request.OperatorCode, attributes["OperatorCode"], defaultProductOperatorCode),
		Price:           strings.TrimSpace(request.Price),
		SpecialPrice:    strings.TrimSpace(request.SalePrice),
		SpecialFromDate: strings.TrimSpace(request.SaleStartDate),
		SpecialToDate:   strings.TrimSpace(request.SaleEndDate),
		Stock:           strings.TrimSpace(attributes["Stock"]),
		Status:          strings.TrimSpace(attributes["Status"]),
	}
	if payload.OperatorCode == "" {
		return businessUnitPayload{}, ErrProductOperatorCodeRequired
	}
	if requireCreateMinimum && payload.Status == "" {
		payload.Status = "active"
	}
	if requireCreateMinimum {
		if payload.Price == "" {
			return businessUnitPayload{}, ErrProductCreatePriceRequired
		}
		if payload.Stock == "" {
			return businessUnitPayload{}, ErrProductCreateStockRequired
		}
		if payload.Status == "" {
			return businessUnitPayload{}, ErrProductCreateStatusRequired
		}
		return payload, nil
	}

	if payload.Price == "" &&
		payload.SpecialPrice == "" &&
		payload.SpecialFromDate == "" &&
		payload.SpecialToDate == "" &&
		payload.Stock == "" &&
		payload.Status == "" {
		return businessUnitPayload{}, ErrProductUpdateBusinessUnitValuesRequired
	}

	return payload, nil
}

// normalizedProductAttributeToken resolves lowercase alphanumeric attribute tokens.
func normalizedProductAttributeToken(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}

	buffer := make([]rune, 0, len(trimmed))
	for _, runeValue := range trimmed {
		if unicode.IsLetter(runeValue) || unicode.IsDigit(runeValue) {
			buffer = append(buffer, unicode.ToLower(runeValue))
		}
	}

	return string(buffer)
}

// writeStartElement writes XML start-elements for provided names.
func writeStartElement(encoder *xml.Encoder, name string) error {
	if err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: name}}); err != nil {
		return fmt.Errorf("write xml start element %q: %w", name, err)
	}

	return nil
}

// writeEndElement writes XML end-elements for provided names.
func writeEndElement(encoder *xml.Encoder, name string) error {
	if err := encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: name}}); err != nil {
		return fmt.Errorf("write xml end element %q: %w", name, err)
	}

	return nil
}

// writeOptionalElement writes XML elements when values are non-empty.
func writeOptionalElement(encoder *xml.Encoder, name string, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	if err := encoder.EncodeElement(trimmed, xml.StartElement{Name: xml.Name{Local: name}}); err != nil {
		return fmt.Errorf("write xml element %q: %w", name, err)
	}

	return nil
}

// writeBusinessUnitsXML writes Falabella BusinessUnits/BusinessUnit payload values.
func writeBusinessUnitsXML(encoder *xml.Encoder, payload businessUnitPayload) error {
	if err := writeStartElement(encoder, "BusinessUnits"); err != nil {
		return err
	}
	if err := writeStartElement(encoder, "BusinessUnit"); err != nil {
		return err
	}
	if err := writeOptionalElement(encoder, "OperatorCode", payload.OperatorCode); err != nil {
		return err
	}
	if err := writeOptionalElement(encoder, "Price", payload.Price); err != nil {
		return err
	}
	if err := writeOptionalElement(encoder, "SpecialPrice", payload.SpecialPrice); err != nil {
		return err
	}
	if err := writeOptionalElement(encoder, "SpecialFromDate", payload.SpecialFromDate); err != nil {
		return err
	}
	if err := writeOptionalElement(encoder, "SpecialToDate", payload.SpecialToDate); err != nil {
		return err
	}
	if err := writeOptionalElement(encoder, "Stock", payload.Stock); err != nil {
		return err
	}
	if err := writeOptionalElement(encoder, "Status", payload.Status); err != nil {
		return err
	}
	if err := writeEndElement(encoder, "BusinessUnit"); err != nil {
		return err
	}
	if err := writeEndElement(encoder, "BusinessUnits"); err != nil {
		return err
	}

	return nil
}

// sanitizeXMLName normalizes dynamic attribute names into XML-safe tag names.
func sanitizeXMLName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "Field"
	}

	runes := []rune(trimmed)
	for index, runeValue := range runes {
		if unicode.IsLetter(runeValue) || unicode.IsDigit(runeValue) || runeValue == '_' || (index > 0 && (runeValue == '-' || runeValue == '.')) {
			continue
		}
		runes[index] = '_'
	}
	if !(unicode.IsLetter(runes[0]) || runes[0] == '_') {
		runes = append([]rune{'_'}, runes...)
	}

	return string(runes)
}

// firstNonEmpty resolves first non-empty value from provided candidates.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
