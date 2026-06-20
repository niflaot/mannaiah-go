package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	corecache "mannaiah/module/core/cache"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

const (
	// defaultRotulusLogoURL defines the default logo URL rendered in rotulus PDFs.
	defaultRotulusLogoURL = "https://storageapi.flockstore.co/fl-assets/assets/2308daa9-2a24-436c-bc92-ee00b8d19f35-flock.png"
	// defaultRotulusCacheTTL defines in-memory cache retention for generated rotulus PDFs.
	defaultRotulusCacheTTL = 5 * time.Minute
	// defaultRotulusHTTPTimeout defines outbound timeout values for logo downloads.
	defaultRotulusHTTPTimeout = 20 * time.Second
	// defaultRotulusCacheKeyPrefix defines Redis cache-key prefixes for rotulus PDFs.
	defaultRotulusCacheKeyPrefix = "shipping:mark_rotulus_document:"
	// defaultRotulusSigningSecret defines fallback signing-secret values for QR payloads.
	defaultRotulusSigningSecret = "shipping-rotulus-default-secret-change-me"
	// rotulusDocumentCacheVersion defines cache version values used to invalidate stale document payloads.
	rotulusDocumentCacheVersion = "v3"
)

// markRotulusDocumentCacheEntry defines one cached rotulus document value.
type markRotulusDocumentCacheEntry struct {
	// Body defines rendered PDF payload bytes.
	Body []byte
	// ExpiresAt defines cache expiration timestamps.
	ExpiresAt time.Time
}

// markRotulusDocumentBuilder defines dependencies used by rotulus document generation.
type markRotulusDocumentBuilder struct {
	// cacheMutex guards in-memory cache reads and writes.
	cacheMutex sync.Mutex
	// cache defines per-key cached rotulus payloads.
	cache map[string]markRotulusDocumentCacheEntry
	// cacheTTL defines cache expiration windows.
	cacheTTL time.Duration
	// cacheStore defines optional external cache dependencies (Redis).
	cacheStore corecache.Store
	// cacheKeyPrefix defines external cache-key prefixes.
	cacheKeyPrefix string
	// logoURL defines rotulus logo URL values.
	logoURL string
	// httpClient defines outbound HTTP client dependencies.
	httpClient *http.Client
	// template defines user-facing rotulus labels.
	template markRotulusTemplate
	// signingSecret defines HMAC secret values for QR payload signing.
	signingSecret string
	// orderSummaryResolver defines optional order-summary dependencies used to render content labels.
	orderSummaryResolver RotulusOrderSummaryResolver
}

// markRotulusMeta defines mark metadata rendered in one rotulus PDF.
type markRotulusMeta struct {
	// MarkID defines shipping mark identifier values.
	MarkID string
	// OrderID defines internal order identifier values.
	OrderID string
	// OrderNumber defines public order identifier values.
	OrderNumber string
	// TrackingNumber defines tracking/document identifier values.
	TrackingNumber string
	// CarrierLabel defines carrier label values.
	CarrierLabel string
	// RecipientName defines recipient display-name values.
	RecipientName string
	// RecipientAddressLine defines recipient address-line values.
	RecipientAddressLine string
	// RecipientAddressLine2 defines recipient address-line-2 values.
	RecipientAddressLine2 string
	// RecipientPhone defines recipient phone values.
	RecipientPhone string
	// RecipientCity defines recipient city label values.
	RecipientCity string
	// Content defines rendered shipment content lines shown in the footer.
	Content string
	// CollectOnDeliveryAmount defines cash-on-delivery amount shown as recaudo when present.
	CollectOnDeliveryAmount float64
	// GeneratedAt defines generation timestamp values.
	GeneratedAt time.Time
}

// rotulusQRPayload defines the signed QR payload encoded into rotulus documents.
type rotulusQRPayload struct {
	// Version defines token version values.
	Version string `json:"v"`
	// MarkID defines shipping mark identifier values.
	MarkID string `json:"markId"`
	// OrderID defines internal order identifier values.
	OrderID string `json:"orderId"`
	// GeneratedAtUnix defines generation timestamps in unix seconds.
	GeneratedAtUnix int64 `json:"generatedAt"`
}

// RotulusOrderSummary defines order metadata rendered in rotulus content rows.
type RotulusOrderSummary struct {
	// Items defines display item labels rendered in the content row.
	Items []string
}

// RotulusOrderSummaryResolver defines order metadata lookup behavior for rotulus content rows.
type RotulusOrderSummaryResolver interface {
	// ResolveRotulusOrderSummary resolves one rotulus order summary by order identifier.
	ResolveRotulusOrderSummary(ctx context.Context, orderID string) (*RotulusOrderSummary, error)
}

// newMarkRotulusDocumentBuilder creates default rotulus document builder dependencies.
func newMarkRotulusDocumentBuilder() *markRotulusDocumentBuilder {
	return &markRotulusDocumentBuilder{
		cache:          map[string]markRotulusDocumentCacheEntry{},
		cacheTTL:       defaultRotulusCacheTTL,
		cacheKeyPrefix: defaultRotulusCacheKeyPrefix,
		logoURL:        defaultRotulusLogoURL,
		httpClient:     &http.Client{Timeout: defaultRotulusHTTPTimeout},
		template:       loadDefaultRotulusTemplate(),
		signingSecret:  defaultRotulusSigningSecret,
	}
}

// RotulusDocument builds one rotulus PDF document for the provided mark.
func (s *Service) RotulusDocument(ctx context.Context, markID string) ([]byte, error) {
	if s == nil || s.repository == nil || s.rotulusDocuments == nil {
		return nil, domain.ErrInvalidID
	}
	trimmedMarkID := strings.TrimSpace(markID)
	if trimmedMarkID == "" {
		return nil, domain.ErrInvalidID
	}

	mark, err := s.repository.GetByID(ctx, trimmedMarkID)
	if err != nil {
		return nil, err
	}
	cacheKey := s.rotulusDocumentCacheKey(*mark)
	if payload, ok := s.getCachedRotulusDocument(ctx, cacheKey); ok {
		return payload, nil
	}

	payload, err := s.buildRotulusPDF(ctx, s.buildRotulusMetaForMark(ctx, mark))
	if err != nil {
		return nil, err
	}

	s.cacheRotulusDocument(ctx, cacheKey, payload)

	return append([]byte(nil), payload...), nil
}

// buildRotulusMetaForMark resolves all display metadata required to render one rotulus for the provided mark.
func (s *Service) buildRotulusMetaForMark(ctx context.Context, mark *domain.ShippingMark) markRotulusMeta {
	now := time.Now().UTC()
	orderNumber := strings.TrimSpace(mark.OrderID)
	recipientAddressLine := strings.TrimSpace(mark.Recipient.AddressLine)
	recipientAddressLine2 := ""
	recipientPhone := strings.TrimSpace(mark.Recipient.Phone)
	recipientCity := resolveRotulusCityDisplayName(strings.TrimSpace(mark.Recipient.CityCode))
	orderItemLabels := []string{}
	if s != nil && s.orderSource != nil {
		orderData, orderErr := s.orderSource.GetByIDOrIdentifier(ctx, strings.TrimSpace(mark.OrderID))
		if orderErr == nil && orderData != nil && strings.TrimSpace(orderData.OrderIdentifier) != "" {
			orderNumber = strings.TrimSpace(orderData.OrderIdentifier)
		}
		if orderErr == nil && orderData != nil {
			recipientAddressLine = firstNonEmptyString(strings.TrimSpace(orderData.RecipientAddressLine), recipientAddressLine)
			recipientAddressLine2 = firstNonEmptyString(strings.TrimSpace(orderData.RecipientAddressLine2), recipientAddressLine2)
			recipientPhone = firstNonEmptyString(strings.TrimSpace(orderData.RecipientPhone), recipientPhone)
			recipientCity = firstNonEmptyString(
				resolveRotulusCityDisplayName(strings.TrimSpace(orderData.RecipientCity)),
				resolveRotulusCityDisplayName(strings.TrimSpace(orderData.DestCityCode)),
				recipientCity,
			)
			orderItemLabels = resolveRotulusOrderDataItemLabels(orderData)
		}
	}
	summaryItems := s.resolveRotulusOrderSummaryItems(ctx, mark.OrderID)

	return markRotulusMeta{
		MarkID:                  mark.ID,
		OrderID:                 mark.OrderID,
		OrderNumber:             firstNonEmptyString(orderNumber, mark.OrderID),
		TrackingNumber:          firstNonEmptyString(strings.TrimSpace(mark.TrackingNumber), strings.TrimSpace(mark.DocumentRef), mark.ID),
		CarrierLabel:            resolveRotulusCarrierLabel(*mark),
		RecipientName:           firstNonEmptyString(strings.TrimSpace(mark.Recipient.Name), strings.TrimSpace(mark.Recipient.LegalName)),
		RecipientAddressLine:    recipientAddressLine,
		RecipientAddressLine2:   recipientAddressLine2,
		RecipientPhone:          recipientPhone,
		RecipientCity:           recipientCity,
		Content:                 resolveRotulusContent(*mark, summaryItems, orderItemLabels),
		CollectOnDeliveryAmount: resolveRotulusCollectOnDeliveryAmount(*mark),
		GeneratedAt:             now,
	}
}

// SetRotulusOrderSummaryResolver configures optional order summary lookup dependencies for rotulus content rows.
func (s *Service) SetRotulusOrderSummaryResolver(resolver RotulusOrderSummaryResolver) {
	if s == nil || s.rotulusDocuments == nil {
		return
	}
	s.rotulusDocuments.orderSummaryResolver = resolver
}

// SetRotulusDocumentCacheTTL configures rotulus cache TTL values.
func (s *Service) SetRotulusDocumentCacheTTL(ttl time.Duration) {
	if s == nil || s.rotulusDocuments == nil {
		return
	}
	if ttl <= 0 {
		s.rotulusDocuments.cacheTTL = defaultRotulusCacheTTL
		return
	}
	s.rotulusDocuments.cacheTTL = ttl
}

// SetRotulusDocumentCacheStore configures external cache dependencies used by rotulus cache.
func (s *Service) SetRotulusDocumentCacheStore(store corecache.Store) {
	if s == nil || s.rotulusDocuments == nil {
		return
	}

	s.rotulusDocuments.cacheStore = store
}

// SetRotulusDocumentHTTPClient configures outbound HTTP client dependencies used by logo downloads.
func (s *Service) SetRotulusDocumentHTTPClient(client *http.Client) {
	if s == nil || s.rotulusDocuments == nil || client == nil {
		return
	}

	s.rotulusDocuments.httpClient = client
}

// SetRotulusDocumentLogoURL configures optional rotulus logo URL values.
func (s *Service) SetRotulusDocumentLogoURL(value string) {
	if s == nil || s.rotulusDocuments == nil {
		return
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		s.rotulusDocuments.logoURL = defaultRotulusLogoURL
		return
	}

	s.rotulusDocuments.logoURL = trimmed
}

// SetRotulusDocumentSigningSecret configures HMAC secret values used by QR payload signing.
func (s *Service) SetRotulusDocumentSigningSecret(value string) {
	if s == nil || s.rotulusDocuments == nil {
		return
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		s.rotulusDocuments.signingSecret = defaultRotulusSigningSecret
		return
	}

	s.rotulusDocuments.signingSecret = trimmed
}

// resolveRotulusCarrierLabel resolves the display carrier label for one rotulus document.
func resolveRotulusCarrierLabel(mark domain.ShippingMark) string {
	if domain.IsManualCarrierID(mark.CarrierID) {
		return firstNonEmptyString(strings.TrimSpace(mark.Observations), strings.TrimSpace(mark.CarrierID))
	}

	return strings.TrimSpace(mark.CarrierID)
}

// resolveRotulusCollectOnDeliveryAmount resolves the shown recaudo amount for one rotulus.
func resolveRotulusCollectOnDeliveryAmount(mark domain.ShippingMark) float64 {
	if mark.CollectOnDeliveryChargedAmount > 0 {
		return mark.CollectOnDeliveryChargedAmount
	}

	return mark.CollectOnDeliveryAmount
}

// resolveRotulusContent resolves content lines prioritizing order summary items and ignoring manual placeholders.
func resolveRotulusContent(mark domain.ShippingMark, summaryItems []string, orderItemLabels []string) string {
	items := normalizeRotulusItemLabels(summaryItems)
	if len(items) == 0 {
		items = resolveRotulusUnitLabels(mark)
	}
	if len(items) == 0 {
		items = normalizeRotulusItemLabels(orderItemLabels)
	}
	if len(items) == 0 {
		return "-"
	}

	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, "- "+item)
	}

	return strings.Join(rows, "\n")
}

// resolveRotulusUnitLabels resolves display item labels from mark units while excluding known manual placeholders.
func resolveRotulusUnitLabels(mark domain.ShippingMark) []string {
	items := make([]string, 0, len(mark.Units))
	for _, unit := range mark.Units {
		description := strings.TrimSpace(unit.Description)
		if description == "" || isRotulusManualPlaceholderDescription(description) {
			continue
		}
		items = append(items, description)
	}

	return normalizeRotulusItemLabels(items)
}

// resolveRotulusOrderSummaryItems resolves item labels from optional order-summary dependencies.
func (s *Service) resolveRotulusOrderSummaryItems(ctx context.Context, orderID string) []string {
	if s == nil || s.rotulusDocuments == nil || s.rotulusDocuments.orderSummaryResolver == nil {
		return nil
	}
	summary, err := s.rotulusDocuments.orderSummaryResolver.ResolveRotulusOrderSummary(ctx, strings.TrimSpace(orderID))
	if err != nil || summary == nil {
		return nil
	}

	return normalizeRotulusItemLabels(summary.Items)
}

// resolveRotulusOrderDataItemLabels resolves fallback item labels from order-source line-item identifiers.
func resolveRotulusOrderDataItemLabels(orderData *port.OrderQuotationData) []string {
	if orderData == nil {
		return nil
	}
	labels := make([]string, 0, len(orderData.Items))
	for _, item := range orderData.Items {
		label := firstNonEmptyString(strings.TrimSpace(item.SKU), strings.TrimSpace(item.ProductID))
		if label == "" {
			continue
		}
		quantity := item.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		label = "X" + strconv.Itoa(quantity) + " " + label
		labels = append(labels, label)
	}

	return normalizeRotulusItemLabels(labels)
}

// normalizeRotulusItemLabels normalizes item labels and removes empty/duplicate values.
func normalizeRotulusItemLabels(items []string) []string {
	normalized := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}

// isRotulusManualPlaceholderDescription reports whether one unit description is a synthetic manual placeholder.
func isRotulusManualPlaceholderDescription(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "manual tracking entry" || normalized == "manual entry"
}

// rotulusDocumentCacheKey resolves one versioned cache key for the provided mark state.
func (s *Service) rotulusDocumentCacheKey(mark domain.ShippingMark) string {
	version := mark.UpdatedAt.UTC().Unix()
	if version <= 0 {
		version = mark.CreatedAt.UTC().Unix()
	}
	if version <= 0 {
		version = 1
	}

	return rotulusDocumentCacheVersion + ":" + strings.TrimSpace(mark.ID) + ":" + strconv.FormatInt(version, 10)
}

// buildSignedRotulusQRToken builds a signed QR payload token for the provided mark meta.
func (s *Service) buildSignedRotulusQRToken(meta markRotulusMeta) (string, error) {
	if s == nil || s.rotulusDocuments == nil {
		return "", domain.ErrInvalidID
	}
	payload, err := json.Marshal(rotulusQRPayload{
		Version:         "flk-rotulus-v1",
		MarkID:          strings.TrimSpace(meta.MarkID),
		OrderID:         strings.TrimSpace(meta.OrderID),
		GeneratedAtUnix: meta.GeneratedAt.UTC().Unix(),
	})
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(firstNonEmptyString(s.rotulusDocuments.signingSecret, defaultRotulusSigningSecret)))
	_, _ = mac.Write([]byte(encodedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))

	return "flkrotulus.v1." + encodedPayload + "." + signature, nil
}

// getCachedRotulusDocument resolves one cached rotulus payload when not expired.
func (s *Service) getCachedRotulusDocument(ctx context.Context, cacheKey string) ([]byte, bool) {
	if s == nil || s.rotulusDocuments == nil {
		return nil, false
	}
	builder := s.rotulusDocuments
	if builder.cacheStore != nil {
		cachedBase64, err := builder.cacheStore.Get(ctx, builder.cacheKey(cacheKey))
		if err == nil {
			payload, decodeErr := base64.StdEncoding.DecodeString(strings.TrimSpace(cachedBase64))
			if decodeErr == nil && len(payload) > 0 {
				return payload, true
			}
		}
	}
	builder.cacheMutex.Lock()
	defer builder.cacheMutex.Unlock()
	entry, exists := builder.cache[cacheKey]
	if !exists {
		return nil, false
	}
	if time.Now().UTC().After(entry.ExpiresAt) {
		delete(builder.cache, cacheKey)
		return nil, false
	}

	return append([]byte(nil), entry.Body...), true
}

// cacheRotulusDocument stores one rotulus payload with cache TTL.
func (s *Service) cacheRotulusDocument(ctx context.Context, cacheKey string, payload []byte) {
	if s == nil || s.rotulusDocuments == nil {
		return
	}
	builder := s.rotulusDocuments
	builder.cacheMutex.Lock()
	builder.cache[cacheKey] = markRotulusDocumentCacheEntry{
		Body:      append([]byte(nil), payload...),
		ExpiresAt: time.Now().UTC().Add(builder.cacheTTL),
	}
	builder.cacheMutex.Unlock()
	if builder.cacheStore != nil {
		_ = builder.cacheStore.Set(ctx, builder.cacheKey(cacheKey), base64.StdEncoding.EncodeToString(payload), builder.cacheTTL)
	}
}

// cacheKey resolves the external-cache key for one rotulus payload entry.
func (b *markRotulusDocumentBuilder) cacheKey(key string) string {
	if b == nil {
		return defaultRotulusCacheKeyPrefix + key
	}

	return firstNonEmptyString(strings.TrimSpace(b.cacheKeyPrefix), defaultRotulusCacheKeyPrefix) + key
}

// firstNonEmptyString resolves the first non-empty trimmed string value.
func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}
