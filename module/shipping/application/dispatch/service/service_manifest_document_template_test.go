package service

import (
	"strings"
	"testing"
)

// TestLoadDefaultBatchManifestCoverTemplate verifies default template labels are loaded.
func TestLoadDefaultBatchManifestCoverTemplate(t *testing.T) {
	t.Parallel()

	template := loadDefaultBatchManifestCoverTemplate()
	if strings.TrimSpace(template.Title) == "" {
		t.Fatalf("template.Title is empty")
	}
	if strings.TrimSpace(template.TrackingNumberHeader) == "" {
		t.Fatalf("template.TrackingNumberHeader is empty")
	}
	if strings.TrimSpace(template.ItemBulletPrefix) == "" {
		t.Fatalf("template.ItemBulletPrefix is empty")
	}
}

// TestParseBatchManifestCoverTemplateRejectsInvalid verifies invalid template payloads are rejected.
func TestParseBatchManifestCoverTemplateRejectsInvalid(t *testing.T) {
	t.Parallel()

	if _, err := parseBatchManifestCoverTemplate([]byte(`{"title":"x"}`)); err == nil {
		t.Fatalf("parseBatchManifestCoverTemplate() error = nil, want non-nil")
	}
}

// TestFormatBatchManifestItemsAsList verifies rendered item-list formatting.
func TestFormatBatchManifestItemsAsList(t *testing.T) {
	t.Parallel()

	rendered := formatBatchManifestItemsAsList([]string{"Morrál Ázul", "Neceser Béige"}, "-", "-")
	want := "- Morrál Ázul\n- Neceser Béige"
	if rendered != want {
		t.Fatalf("formatBatchManifestItemsAsList() = %q, want %q", rendered, want)
	}
}
