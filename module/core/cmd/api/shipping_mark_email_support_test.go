package main

import (
	"strings"
	"testing"

	shippingdomain "mannaiah/module/shipping/domain"
)

// TestResolveShippingTrackingURLUsesManualCarrierSlug verifies manual carrier links use the operator-provided slug instead of "manual".
func TestResolveShippingTrackingURLUsesManualCarrierSlug(t *testing.T) {
	url := resolveShippingTrackingURL(
		shippingEmailConsumerDependencies{trackingBaseURL: "https://rastreo.flockstore.co"},
		shippingdomain.ShippingMark{CarrierID: "manual", Observations: "Inter rapidísimo"},
		"11515151",
		"1024751",
	)

	if strings.Contains(url, "carrier=manual") {
		t.Fatalf("tracking url should not use manual carrier id: %q", url)
	}
	if !strings.Contains(url, "carrier=interrapidisimo") {
		t.Fatalf("tracking url should use normalized manual carrier slug: %q", url)
	}
}
