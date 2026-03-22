package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	campaigntemplate "mannaiah/module/campaign/application/template"
	"mannaiah/module/campaign/domain"
	"mannaiah/module/campaign/port"
)

type affinityProductProviderSpy struct {
	calls int
}

type contactDataProviderStub struct {
	data port.ContactData
	err  error
}

// GetContactData returns predefined contact data for rendering assertions.
func (s contactDataProviderStub) GetContactData(_ context.Context, _ string) (port.ContactData, error) {
	return s.data, s.err
}

// GetProducts tracks calls and returns one deterministic product for rendering assertions.
func (s *affinityProductProviderSpy) GetProducts(_ context.Context, _ string, _ domain.ProductBlock) ([]domain.TemplateProduct, error) {
	s.calls++
	return []domain.TemplateProduct{{
		ID:       "p-1",
		Name:     "Demo Product 1",
		Price:    49.9,
		ImageURL: "https://example.com/p-1.jpg",
		URL:      "https://store.example.com/products/p-1",
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

// TestRenderForContactExposesProductURL verifies product URL values are available in template loops.
func TestRenderForContactExposesProductURL(t *testing.T) {
	t.Parallel()

	spy := &affinityProductProviderSpy{}
	service := &CampaignService{
		contactDataProvider:     port.NoopContactDataProvider{},
		affinityProductProvider: spy,
		templateRenderer:        campaigntemplate.NewRenderer(),
	}
	campaign := &domain.Campaign{
		ID:       "c-2-url",
		Slug:     "slug",
		HTMLBody: `{{ with index .Products "hero_products" }}{{ range . }}{{ .URL }}{{ end }}{{ end }}`,
		TextBody: ``,
		ProductBlocks: []domain.ProductBlock{
			{ID: "hero_products", BaseTags: []string{"offer-tier-1"}},
		},
	}

	htmlBody, _ := service.renderForContact(context.Background(), campaign, "contact-1", "jane@example.com")
	if !strings.Contains(htmlBody, "https://store.example.com/products/p-1") {
		t.Fatalf("htmlBody = %q, want rendered product URL", htmlBody)
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

// TestRenderForContactStrictUsesRecipientEmailInTemplateContext verifies contact email in templates follows the actual send recipient.
func TestRenderForContactStrictUsesRecipientEmailInTemplateContext(t *testing.T) {
	t.Parallel()

	lastSaleDate := time.Now().UTC()
	service := &CampaignService{
		contactDataProvider: contactDataProviderStub{
			data: port.ContactData{
				Name:         "Juliana Marcela",
				Email:        "real-contact@example.com",
				LastSaleDate: &lastSaleDate,
			},
		},
		affinityProductProvider: &affinityProductProviderSpy{},
		templateRenderer:        campaigntemplate.NewRenderer(),
	}
	campaign := &domain.Campaign{
		ID:       "c-4",
		Slug:     "slug",
		HTMLBody: `{{ .Contact.Email }}`,
		TextBody: `{{ .Contact.Email }}`,
	}

	htmlBody, textBody, err := service.renderForContactStrict(context.Background(), campaign, "contact-1", "override-test@example.com")
	if err != nil {
		t.Fatalf("renderForContactStrict() error = %v", err)
	}
	if htmlBody != "override-test@example.com" {
		t.Fatalf("html email = %q, want override email", htmlBody)
	}
	if textBody != "override-test@example.com" {
		t.Fatalf("text email = %q, want override email", textBody)
	}
}

// TestRenderForContactStrictReturnsContactPersonalizationError verifies strict rendering fails when explicit contact personalization lookup fails.
func TestRenderForContactStrictReturnsContactPersonalizationError(t *testing.T) {
	t.Parallel()

	service := &CampaignService{
		contactDataProvider: contactDataProviderStub{
			err: errors.New("contact not found"),
		},
		affinityProductProvider: &affinityProductProviderSpy{},
		templateRenderer:        campaigntemplate.NewRenderer(),
	}
	campaign := &domain.Campaign{
		ID:       "c-5",
		Slug:     "slug",
		HTMLBody: `{{ .Contact.Name }}`,
		TextBody: `{{ .Contact.Name }}`,
	}

	_, _, err := service.renderForContactStrict(context.Background(), campaign, "missing-contact", "override@example.com")
	if err == nil {
		t.Fatalf("expected contact personalization error")
	}
	if !strings.Contains(err.Error(), domain.ErrContactPersonalization.Error()) {
		t.Fatalf("error = %v, want contact personalization sentinel", err)
	}
}

// TestRenderForContactStrictUsesNameOnlyForContactName verifies .Contact.Name resolves to first-name style while .Contact.FullName keeps full display name.
func TestRenderForContactStrictUsesNameOnlyForContactName(t *testing.T) {
	t.Parallel()

	service := &CampaignService{
		contactDataProvider: contactDataProviderStub{
			data: port.ContactData{Name: "Juliana Marcela Villegas Sarmiento"},
		},
		affinityProductProvider: &affinityProductProviderSpy{},
		templateRenderer:        campaigntemplate.NewRenderer(),
	}
	campaign := &domain.Campaign{
		ID:       "c-6",
		Slug:     "slug",
		HTMLBody: `{{ .Contact.Name }}|{{ .Contact.FullName }}|{{ .Contact.FirstName }}`,
		TextBody: ``,
	}

	htmlBody, _, err := service.renderForContactStrict(context.Background(), campaign, "contact-1", "override@example.com")
	if err != nil {
		t.Fatalf("renderForContactStrict() error = %v", err)
	}
	if htmlBody != "Juliana|Juliana Marcela Villegas Sarmiento|Juliana" {
		t.Fatalf("htmlBody = %q, want first|full|first", htmlBody)
	}
}
