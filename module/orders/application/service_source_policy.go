package application

import "strings"

const (
	// wooSyncMutationSource defines WooCommerce sync source values that are allowed to mutate Woo orders.
	wooSyncMutationSource = "woocommerce_sync"
)

// shouldIgnoreWooInboundMutation reports whether Woo-origin inbound mutations should be ignored for Woo orders.
func shouldIgnoreWooInboundMutation(source string, realm string) bool {
	if !strings.EqualFold(strings.TrimSpace(realm), "woocommerce") {
		return false
	}

	normalized := strings.ToLower(strings.TrimSpace(source))
	if normalized == "" || normalized == wooSyncMutationSource {
		return false
	}

	return isWooSource(normalized)
}

// isWooSource reports whether mutation source values represent Woo-origin operations.
func isWooSource(source string) bool {
	normalized := strings.ToLower(strings.TrimSpace(source))
	if normalized == "" {
		return false
	}

	return strings.HasPrefix(normalized, "woocommerce")
}
