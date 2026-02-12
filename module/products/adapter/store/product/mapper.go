package product

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	productdomain "mannaiah/module/products/domain/product"
)

// recordPayload defines encoded product composite fields.
type recordPayload struct {
	// Gallery defines encoded gallery values.
	Gallery string
	// Datasheets defines encoded datasheet values.
	Datasheets string
	// Variations defines encoded variation IDs.
	Variations string
	// Variants defines encoded variant values.
	Variants string
}

// toRecordPayload maps products into encoded persistence payloads.
func toRecordPayload(entity productdomain.Product) (recordPayload, error) {
	galleryJSON, err := marshalJSON(entity.Gallery)
	if err != nil {
		return recordPayload{}, fmt.Errorf("marshal gallery: %w", err)
	}
	datasheetsJSON, err := marshalJSON(entity.Datasheets)
	if err != nil {
		return recordPayload{}, fmt.Errorf("marshal datasheets: %w", err)
	}
	variationsJSON, err := marshalJSON(entity.Variations)
	if err != nil {
		return recordPayload{}, fmt.Errorf("marshal variations: %w", err)
	}
	variantsJSON, err := marshalJSON(entity.Variants)
	if err != nil {
		return recordPayload{}, fmt.Errorf("marshal variants: %w", err)
	}

	return recordPayload{
		Gallery:    galleryJSON,
		Datasheets: datasheetsJSON,
		Variations: variationsJSON,
		Variants:   variantsJSON,
	}, nil
}

// toDomain maps persistence rows into product entities.
func toDomain(record productRecord) (productdomain.Product, error) {
	entity := productdomain.Product{
		ID:        record.ID,
		SKU:       record.SKU,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
		IsDeleted: record.DeletedAt.Valid,
		DeletedAt: fromDeletedAt(record.DeletedAt),
	}
	if err := unmarshalJSON(record.Gallery, &entity.Gallery); err != nil {
		return productdomain.Product{}, fmt.Errorf("unmarshal gallery: %w", err)
	}
	if err := unmarshalJSON(record.Datasheets, &entity.Datasheets); err != nil {
		return productdomain.Product{}, fmt.Errorf("unmarshal datasheets: %w", err)
	}
	if err := unmarshalJSON(record.Variations, &entity.Variations); err != nil {
		return productdomain.Product{}, fmt.Errorf("unmarshal variations: %w", err)
	}
	if err := unmarshalJSON(record.Variants, &entity.Variants); err != nil {
		return productdomain.Product{}, fmt.Errorf("unmarshal variants: %w", err)
	}

	return entity, nil
}

// marshalJSON marshals values into JSON strings.
func marshalJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}

// unmarshalJSON unmarshals JSON strings into values.
func unmarshalJSON(value string, destination any) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = "[]"
	}

	return json.Unmarshal([]byte(trimmed), destination)
}

// toDeletedAt converts nullable pointers to GORM deleted-at values.
func toDeletedAt(value *time.Time) gorm.DeletedAt {
	if value == nil {
		return gorm.DeletedAt{}
	}

	return gorm.DeletedAt{Time: *value, Valid: true}
}

// fromDeletedAt converts GORM deleted-at values to nullable timestamps.
func fromDeletedAt(value gorm.DeletedAt) *time.Time {
	if !value.Valid {
		return nil
	}

	timestamp := value.Time
	return &timestamp
}
