package application

import (
	"fmt"
	"strings"

	"mannaiah/module/campaign/domain"
)

// normalizeSenderError maps provider-specific send failures to campaign domain errors.
func normalizeSenderError(err error) error {
	if err == nil {
		return nil
	}
	if isSenderUnavailableError(err.Error()) {
		return fmt.Errorf("%w: %v", domain.ErrSenderUnavailable, err)
	}

	return err
}

// isSenderUnavailableError reports whether a sender error is due to provider unavailability/config constraints.
func isSenderUnavailableError(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	if strings.Contains(normalized, "messagerejected") && strings.Contains(normalized, "email address is not verified") {
		return true
	}
	if strings.Contains(normalized, "mail from domain is not verified") {
		return true
	}
	if strings.Contains(normalized, "account is still in the sandbox") {
		return true
	}

	return false
}
