package application

import (
	"context"
	"strings"
	"testing"

	campaigntemplate "mannaiah/module/campaign/application/template"
	"mannaiah/module/campaign/domain"
	"mannaiah/module/campaign/port"
)

type affinityProductProviderSpy struct {
	calls int
}

// GetProducts tracks calls and returns one deterministic product for rendering assertions.
func (s *affinityProductProviderSpy) GetProducts(_ context.Context, _ string, _ domain.ProductBlock) ([]domain.TemplateProduct, error) {
	s.calls++
	return []domain.TemplateProduct{{
		ID:       "p-1",
		Name:     "Demo Product 1",
		Price:    49.9,
		ImageURL: "https://example.com/p-1.jpg",
	}}, nil
}

// TestRenderForContactUsesBaseTagsOnlyBlock verifies product blocks with BaseTags (without BaseTag) are resolved.
func TestRenderForContactUsesBaseTagsOnlyBlock(t *testing.T) {
	t.Parallel()

	spy := &affinityProductProviderSpy{}
	service := &CampaignService{
		contactDataProvider:     port.NoopContactDataProvider{},
		affinityProductProvider: spy,
		templateRenderer:        campaigntemplate.NewRenderer(),
	}
	campaign := &domain.Campaign{
		ID:       "c-1",
		Slug:     "slug",
		HTMLBody: `{{ with index .Products "hero_products" }}{{ range . }}{{ .Name }}{{ end }}{{ end }}`,
		TextBody: ``,
		ProductBlocks: []domain.ProductBlock{
			{ID: "hero_products", BaseTags: []string{"offer-tier-1"}},
		},
	}

	htmlBody, _ := service.renderForContact(context.Background(), campaign, "contact-1", "jane@example.com")
	if spy.calls != 1 {
		t.Fatalf("GetProducts calls = %d, want 1", spy.calls)
	}
	if !strings.Contains(htmlBody, "Demo Product 1") {
		t.Fatalf("htmlBody = %q, want rendered product name", htmlBody)
	}
}

// TestRenderForContactUsesPinnedOnlyBlock verifies product blocks with pinned IDs are resolved even without base tags.
func TestRenderForContactUsesPinnedOnlyBlock(t *testing.T) {
	t.Parallel()

	spy := &affinityProductProviderSpy{}
	service := &CampaignService{
		contactDataProvider:     port.NoopContactDataProvider{},
		affinityProductProvider: spy,
		templateRenderer:        campaigntemplate.NewRenderer(),
	}
	campaign := &domain.Campaign{
		ID:       "c-2",
		Slug:     "slug",
		HTMLBody: `{{ with index .Products "hero_products" }}{{ range . }}{{ .Name }}{{ end }}{{ end }}`,
		TextBody: ``,
		ProductBlocks: []domain.ProductBlock{
			{ID: "hero_products", PinnedProductIDs: []string{"pinned-1"}},
		},
	}

	htmlBody, _ := service.renderForContact(context.Background(), campaign, "contact-1", "jane@example.com")
	if spy.calls != 1 {
		t.Fatalf("GetProducts calls = %d, want 1", spy.calls)
	}
	if !strings.Contains(htmlBody, "Demo Product 1") {
		t.Fatalf("htmlBody = %q, want rendered product name", htmlBody)
	}
}

// TestRenderForContactStrictReturnsTemplateError verifies strict rendering returns template parse/execute errors.
func TestRenderForContactStrictReturnsTemplateError(t *testing.T) {
	t.Parallel()

	service := &CampaignService{
		contactDataProvider:     port.NoopContactDataProvider{},
		affinityProductProvider: &affinityProductProviderSpy{},
		templateRenderer:        campaigntemplate.NewRenderer(),
	}
	campaign := &domain.Campaign{
		ID:       "c-3",
		Slug:     "slug",
		HTMLBody: `{{ if .Products }}{{ with index .Products.hero_products o }}{{ end }}{{ end }}`,
		TextBody: ``,
	}

	_, _, err := service.renderForContactStrict(context.Background(), campaign, "contact-1", "jane@example.com")
	if err == nil {
		t.Fatalf("expected strict render error for invalid template syntax")
	}
}
