package falabella

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"mannaiah/module/falabella/port"
)

var (
	// ErrImageSKURequired is returned when image-sync SKU values are missing.
	ErrImageSKURequired = errors.New("falabella image seller sku is required")
	// ErrImageURLRequired is returned when image-sync URL values are missing.
	ErrImageURLRequired = errors.New("falabella image url is required")
)

// buildImageRequestXML builds Falabella XML request bodies for Image calls.
func buildImageRequestXML(request port.SyncProductImagesRequest) ([]byte, error) {
	sku := strings.TrimSpace(request.SKU)
	if sku == "" {
		return nil, ErrImageSKURequired
	}

	urls := uniqueTrimmedValues(request.URLs)
	if len(urls) == 0 {
		return nil, ErrImageURLRequired
	}

	writer := &bytes.Buffer{}
	encoder := xml.NewEncoder(writer)
	if err := writeStartElement(encoder, "Request"); err != nil {
		return nil, err
	}
	if err := writeStartElement(encoder, "ProductImage"); err != nil {
		return nil, err
	}
	if err := writeOptionalElement(encoder, "SellerSku", sku); err != nil {
		return nil, err
	}
	if err := writeStartElement(encoder, "Images"); err != nil {
		return nil, err
	}
	for _, imageURL := range urls {
		if err := writeOptionalElement(encoder, "Image", imageURL); err != nil {
			return nil, err
		}
	}
	if err := writeEndElement(encoder, "Images"); err != nil {
		return nil, err
	}
	if err := writeEndElement(encoder, "ProductImage"); err != nil {
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

// uniqueTrimmedValues returns deduplicated, non-empty input values preserving first occurrence order.
func uniqueTrimmedValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}
