package product

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// marshalAttributeValue marshals datasheet attribute values.
func marshalAttributeValue(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal datasheet attribute value: %w", err)
	}

	return string(encoded), nil
}

// unmarshalAttributeValue unmarshals datasheet attribute values.
func unmarshalAttributeValue(raw string) (any, error) {
	if raw == "" {
		return nil, nil
	}

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("unmarshal datasheet attribute value: %w", err)
	}

	return value, nil
}

// fromDeletedAt converts GORM deleted-at values to nullable timestamps.
func fromDeletedAt(value gorm.DeletedAt) *time.Time {
	if !value.Valid {
		return nil
	}

	timestamp := value.Time
	return &timestamp
}
