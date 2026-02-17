package falabella

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"mannaiah/module/falabella/port"
)

// buildProductRequestXML builds Falabella XML request bodies for ProductCreate/ProductUpdate calls.
func buildProductRequestXML(request port.SyncProductRequest) ([]byte, error) {
	writer := &bytes.Buffer{}
	encoder := xml.NewEncoder(writer)

	if err := writeStartElement(encoder, "Request"); err != nil {
		return nil, err
	}
	if err := writeStartElement(encoder, "Product"); err != nil {
		return nil, err
	}

	if err := writeOptionalElement(encoder, "SellerSku", request.SKU); err != nil {
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
	if err := writeOptionalElement(encoder, "Price", request.Price); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "SpecialPrice", request.SalePrice); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "SpecialFromDate", request.SaleStartDate); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "SpecialToDate", request.SaleEndDate); err != nil {
		return nil, err
	}

	if len(request.Attributes) > 0 {
		if err := writeStartElement(encoder, "ProductData"); err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(request.Attributes))
		for key := range request.Attributes {
			if strings.TrimSpace(key) != "" {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := strings.TrimSpace(request.Attributes[key])
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
