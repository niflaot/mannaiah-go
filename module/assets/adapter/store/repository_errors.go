package store

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// isUniqueConstraintError reports whether a persistence error originates from unique-key conflicts.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate") || strings.Contains(message, "unique constraint")
}
