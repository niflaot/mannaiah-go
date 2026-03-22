package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
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

// TestRenderForContactStrictInjectsSignedUnsubscribeURL verifies .Custom.unsubscribe_url is generated from env-backed settings and includes signed payload values.
func TestRenderForContactStrictInjectsSignedUnsubscribeURL(t *testing.T) {
	t.Parallel()

	service := &CampaignService{
		contactDataProvider: contactDataProviderStub{
			data: port.ContactData{Name: "Juliana Marcela Villegas Sarmiento"},
		},
		affinityProductProvider: &affinityProductProviderSpy{},
		templateRenderer:        campaigntemplate.NewRenderer(),
	}
	service.SetUnsubscribeURLConfig("https://mannaiah.flockstore.co/", "test-optout-secret", 2*time.Hour)

	campaign := &domain.Campaign{
		ID:           "c-7",
		Slug:         "slug",
		HTMLBody:     `{{ .Custom.brand }}|{{ .Custom.unsubscribe_url }}`,
		TextBody:     ``,
		TemplateVars: map[string]string{"brand": "FLOCK"},
	}

	htmlBody, _, err := service.renderForContactStrict(context.Background(), campaign, "contact-1", "override-test@example.com")
	if err != nil {
		t.Fatalf("renderForContactStrict() error = %v", err)
	}
	if !strings.HasPrefix(htmlBody, "FLOCK|https://mannaiah.flockstore.co/public/marketing/optout/") {
		t.Fatalf("htmlBody = %q, want unsubscribe URL prefix", htmlBody)
	}
	if _, exists := campaign.TemplateVars["unsubscribe_url"]; exists {
		t.Fatalf("campaign.TemplateVars mutated with unsubscribe_url")
	}

	parts := strings.SplitN(htmlBody, "|", 2)
	if len(parts) != 2 {
		t.Fatalf("htmlBody = %q, want custom+url parts", htmlBody)
	}
	tokenEncoded := strings.TrimPrefix(parts[1], "https://mannaiah.flockstore.co/public/marketing/optout/")
	token, decodeErr := url.PathUnescape(tokenEncoded)
	if decodeErr != nil {
		t.Fatalf("PathUnescape() error = %v", decodeErr)
	}
	tokenParts := strings.Split(token, ".")
	if len(tokenParts) != 2 {
		t.Fatalf("token = %q, want payload.signature", token)
	}
	if got, want := tokenParts[1], signMarketingOptOutToken(tokenParts[0], "test-optout-secret"); got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}

	payloadBytes, payloadErr := base64.RawURLEncoding.DecodeString(tokenParts[0])
	if payloadErr != nil {
		t.Fatalf("DecodeString() error = %v", payloadErr)
	}
	var payload marketingOptOutTokenPayload
	if unmarshalErr := json.Unmarshal(payloadBytes, &payload); unmarshalErr != nil {
		t.Fatalf("Unmarshal() error = %v", unmarshalErr)
	}
	if payload.Email != "override-test@example.com" {
		t.Fatalf("payload.Email = %q, want override recipient email", payload.Email)
	}
	if payload.Name == nil || *payload.Name != "Juliana Marcela Villegas Sarmiento" {
		t.Fatalf("payload.Name = %v, want full contact name", payload.Name)
	}
	if payload.CampaignID == nil || *payload.CampaignID != "c-7" {
		t.Fatalf("payload.CampaignID = %v, want campaign id", payload.CampaignID)
	}
	if payload.IssuedAt <= 0 || payload.ExpiresAt <= payload.IssuedAt {
		t.Fatalf("iat/exp invalid: iat=%d exp=%d", payload.IssuedAt, payload.ExpiresAt)
	}
	ttl := payload.ExpiresAt - payload.IssuedAt
	if ttl < int64((2*time.Hour)-time.Minute)/int64(time.Second) || ttl > int64((2*time.Hour)+time.Minute)/int64(time.Second) {
		t.Fatalf("ttl=%d, want approximately %d", ttl, int64((2*time.Hour)/time.Second))
	}
}
