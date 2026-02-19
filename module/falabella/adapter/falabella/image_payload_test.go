package falabella

import (
	"strings"
	"testing"

	"mannaiah/module/falabella/port"
)

// TestBuildImageRequestXML verifies image payload generation behavior.
func TestBuildImageRequestXML(t *testing.T) {
	payload, err := buildImageRequestXML(port.SyncProductImagesRequest{
		SKU:  "SKU-1",
		URLs: []string{"https://cdn.example.com/front.jpg", "https://cdn.example.com/back.jpg"},
	})
	if err != nil {
		t.Fatalf("buildImageRequestXML() error = %v", err)
	}

	text := string(payload)
	expected := []string{
		"<Request>",
		"<ProductImage>",
		"<SellerSku>SKU-1</SellerSku>",
		"<Images>",
		"<Image>https://cdn.example.com/front.jpg</Image>",
		"<Image>https://cdn.example.com/back.jpg</Image>",
		"</Images>",
		"</ProductImage>",
		"</Request>",
	}
	for _, item := range expected {
		if !strings.Contains(text, item) {
			t.Fatalf("payload missing %q: %s", item, text)
		}
	}
}

// TestBuildImageRequestXMLValidation verifies image payload validation behavior.
func TestBuildImageRequestXMLValidation(t *testing.T) {
	if _, err := buildImageRequestXML(port.SyncProductImagesRequest{}); err != ErrImageSKURequired {
		t.Fatalf("buildImageRequestXML(empty-sku) error = %v, want %v", err, ErrImageSKURequired)
	}
	if _, err := buildImageRequestXML(port.SyncProductImagesRequest{SKU: "SKU-1"}); err != ErrImageURLRequired {
		t.Fatalf("buildImageRequestXML(empty-urls) error = %v, want %v", err, ErrImageURLRequired)
	}
}

// TestUniqueTrimmedValues verifies deduplicated value normalization behavior.
func TestUniqueTrimmedValues(t *testing.T) {
	values := uniqueTrimmedValues([]string{" a ", "a", "", "b"})
	if len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("uniqueTrimmedValues() = %#v, want [a b]", values)
	}
}
