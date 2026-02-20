package falabella

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"mannaiah/module/falabella/port"

	"go.uber.org/zap"
)

const (
	// getBrandsAction defines Falabella action values used for connection checks.
	getBrandsAction = "GetBrands"
	// getProductsAction defines Falabella action values used for SKU existence checks.
	getProductsAction = "GetProducts"
	// productCreateAction defines Falabella action values used for product creation.
	productCreateAction = "ProductCreate"
	// productUpdateAction defines Falabella action values used for product updates.
	productUpdateAction = "ProductUpdate"
	// imageAction defines Falabella action values used for product image updates.
	imageAction = "Image"
	// feedStatusAction defines Falabella action values used for feed status queries.
	feedStatusAction = "FeedStatus"
	// defaultFormat defines default Falabella format values for read actions.
	defaultFormat = "XML"
	// defaultUserAgent defines fallback User-Agent header values for Falabella requests.
	defaultUserAgent = "UNKNOWN/mannaiah-go/1.0/PROPIA/FACO"
	// timestampLayoutNoColon defines ISO8601 timestamp layout without timezone separator.
	timestampLayoutNoColon = "2006-01-02T15:04:05-0700"
	// jsonContentType defines JSON request content type values.
	jsonContentType = "application/json"
	// xmlContentType defines XML request content type values.
	xmlContentType = "application/xml"
	// signatureInputRFC3986 signs canonical query strings with RFC3986 encoding.
	signatureInputRFC3986 = "rfc3986"
	// signatureInputRaw signs canonical query strings without URL encoding.
	signatureInputRaw = "raw"
)

var (
	// ErrMissingURL is returned when Falabella URL values are missing.
	ErrMissingURL = errors.New("falabella url is required")
	// ErrMissingUserID is returned when Falabella user values are missing.
	ErrMissingUserID = errors.New("falabella user id is required")
	// ErrMissingAPIKey is returned when Falabella api key values are missing.
	ErrMissingAPIKey = errors.New("falabella api key is required")
	// ErrInvalidURL is returned when Falabella URL values are invalid.
	ErrInvalidURL = errors.New("falabella url is invalid")
	// ErrEmptyResponse is returned when Falabella responses contain no payload.
	ErrEmptyResponse = errors.New("falabella response body is empty")
)

// httpDoer defines outbound HTTP behavior required by this adapter.
type httpDoer interface {
	// Do executes outbound HTTP requests.
	Do(req *http.Request) (*http.Response, error)
}

// Client defines Falabella source adapter behavior.
type Client struct {
	// cfg defines Falabella client configuration values.
	cfg Config
	// httpClient defines outbound HTTP dependencies.
	httpClient httpDoer
	// now resolves current timestamp values for signed requests.
	now func() time.Time
	// signingProfiles defines ordered Falabella signing profile candidates.
	signingProfiles []signingProfile
	// activeSigningProfileIdx defines active signing profile index values.
	activeSigningProfileIdx atomic.Int32
}

// signingContext defines request-signing diagnostic values for Falabella actions.
type signingContext struct {
	// Profile defines signing profile values used by this request.
	Profile string
	// Action defines signed action values.
	Action string
	// Canonical defines canonical query values used for HMAC generation.
	Canonical string
	// Signature defines signed signature values.
	Signature string
	// Timestamp defines signed timestamp values.
	Timestamp string
	// UserAgent defines outbound User-Agent header values.
	UserAgent string
	// Format defines optional outbound format parameter values.
	Format string
}

// signingProfile defines a Falabella request-signing strategy candidate.
type signingProfile struct {
	// Name defines stable signing profile identifiers.
	Name string
	// TimestampLayout defines timestamp serialization format values.
	TimestampLayout string
	// Format defines optional Falabella Format parameter values.
	Format string
	// UppercaseSignature defines uppercase hexadecimal signature formatting behavior.
	UppercaseSignature bool
	// SignatureInput defines canonical-signature input serialization variants.
	SignatureInput string
}

// actionCallResponse defines normalized HTTP response values.
type actionCallResponse struct {
	// StatusCode defines HTTP status-code values.
	StatusCode int
	// Body defines raw response payload values.
	Body []byte
	// HTTPResponse defines raw HTTP response metadata.
	HTTPResponse *http.Response
}

// falabellaErrorHead defines normalized Falabella error-response head values.
type falabellaErrorHead struct {
	// RequestAction defines Falabella error action values.
	RequestAction string `xml:"RequestAction" json:"RequestAction"`
	// ErrorType defines Falabella error type values.
	ErrorType string `xml:"ErrorType" json:"ErrorType"`
	// ErrorCode defines Falabella error code values.
	ErrorCode string `xml:"ErrorCode" json:"ErrorCode"`
	// ErrorMessage defines Falabella error message values.
	ErrorMessage string `xml:"ErrorMessage" json:"ErrorMessage"`
}

// falabellaXMLErrorResponse defines Falabella XML error payload values.
type falabellaXMLErrorResponse struct {
	XMLName xml.Name           `xml:"ErrorResponse"`
	Head    falabellaErrorHead `xml:"Head"`
}

// falabellaJSONErrorResponse defines Falabella JSON error payload values.
type falabellaJSONErrorResponse struct {
	ErrorResponse struct {
		Head falabellaErrorHead `json:"Head"`
	} `json:"ErrorResponse"`
}

var (
	// _ ensures Client satisfies port contracts.
	_ interface {
		Validate(ctx context.Context) error
		GetBrands(ctx context.Context) ([]byte, error)
		SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error)
		SyncProductImages(ctx context.Context, request port.SyncProductImagesRequest) ([]byte, error)
		GetFeedStatus(ctx context.Context, feedID string) ([]byte, error)
	} = (*Client)(nil)
)

// NewClient creates Falabella source adapters backed by direct HTTP requests.
func NewClient(cfg Config) (*Client, error) {
	resolvedCfg, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: resolvedCfg.Timeout}
	cookieStore, jarErr := cookiejar.New(nil)
	if jarErr == nil {
		httpClient.Jar = cookieStore
	}

	return newClientWithDependencies(resolvedCfg, httpClient, time.Now), nil
}

// Validate verifies integration availability by checking configuration completeness.
// Configuration credentials and URL are validated at construction time; this method
// confirms the client was properly initialized without issuing outbound API calls.
func (c *Client) Validate(_ context.Context) error {
	return nil
}

// GetBrands retrieves Falabella brand payload using signed requests.
func (c *Client) GetBrands(ctx context.Context) ([]byte, error) {
	body, signing, err := c.executeAction(
		ctx,
		http.MethodGet,
		"get brands",
		getBrandsAction,
		"",
		nil,
		nil,
		c.xmlAndNoFormatSigningProfileIndices(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w (%s)", err, signing)
	}

	return body, nil
}

// SyncProduct upserts a product into Falabella using GetProducts existence checks.
func (c *Client) SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error) {
	productExists, existsErr := c.shouldUpdateProduct(ctx, request)
	if existsErr != nil {
		return nil, existsErr
	}

	actionLabel := "product create"
	apiAction := productCreateAction
	payloadBuilder := buildProductCreateRequestXML
	if productExists {
		actionLabel = "product update"
		apiAction = productUpdateAction
		payloadBuilder = buildProductRequestXML
	}

	payload, err := payloadBuilder(request)
	if err != nil {
		return nil, fmt.Errorf("build falabella product payload: %w", err)
	}

	body, signing, execErr := c.executeAction(
		ctx,
		http.MethodPost,
		actionLabel,
		apiAction,
		xmlContentType,
		payload,
		nil,
		c.xmlAndNoFormatSigningProfileIndices(),
	)
	if execErr != nil {
		return nil, fmt.Errorf("%w (%s); product_payload=%s", execErr, signing, trimBody(payload))
	}

	return body, nil
}

// shouldUpdateProduct resolves whether sync requests should use ProductUpdate actions.
// For variant payloads, Falabella catalogs can already contain parent SKUs while child SKUs
// are still absent; in those cases ProductUpdate is preferred to avoid duplicate-create failures.
func (c *Client) shouldUpdateProduct(ctx context.Context, request port.SyncProductRequest) (bool, error) {
	trimmedSKU := strings.TrimSpace(request.SKU)
	if trimmedSKU == "" {
		return false, errors.New("falabella product sku is required")
	}

	exists, err := c.productExists(ctx, trimmedSKU)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	trimmedParentSKU := strings.TrimSpace(request.ParentSKU)
	if trimmedParentSKU == "" || strings.EqualFold(trimmedParentSKU, trimmedSKU) {
		return false, nil
	}

	parentExists, parentErr := c.productExists(ctx, trimmedParentSKU)
	if parentErr != nil {
		return false, fmt.Errorf("check falabella parent sku existence: %w", parentErr)
	}

	return parentExists, nil
}

// SyncProductImages configures product image URLs in Falabella.
func (c *Client) SyncProductImages(ctx context.Context, request port.SyncProductImagesRequest) ([]byte, error) {
	payload, err := buildImageRequestXML(request)
	if err != nil {
		return nil, fmt.Errorf("build falabella image payload: %w", err)
	}

	result, signing, execErr := c.executeAction(
		ctx,
		http.MethodPost,
		"image",
		imageAction,
		xmlContentType,
		payload,
		nil,
		c.xmlAndNoFormatSigningProfileIndices(),
	)
	if execErr != nil {
		return nil, fmt.Errorf("%w (%s)", execErr, signing)
	}

	return result, nil
}

// GetFeedStatus retrieves Falabella feed status by feed identifier.
func (c *Client) GetFeedStatus(ctx context.Context, feedID string) ([]byte, error) {
	trimmedFeedID := strings.TrimSpace(feedID)
	if trimmedFeedID == "" {
		return nil, errors.New("falabella feed id is required")
	}

	body, signing, err := c.executeAction(
		ctx,
		http.MethodGet,
		"feed status",
		feedStatusAction,
		"",
		nil,
		map[string]string{"FeedID": trimmedFeedID},
		c.xmlAndNoFormatSigningProfileIndices(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w (%s)", err, signing)
	}

	return body, nil
}

// executeAction executes Falabella actions with guarded signing-profile retries.
func (c *Client) executeAction(
	ctx context.Context,
	method string,
	actionLabel string,
	apiAction string,
	contentType string,
	payload []byte,
	queryParams map[string]string,
	profileIndices []int,
) ([]byte, *signingContext, error) {
	orderedIndices := append([]int(nil), profileIndices...)
	if len(orderedIndices) == 0 {
		orderedIndices = []int{c.activeSigningProfileIndex()}
	}

	var (
		lastErr     error
		lastSigning *signingContext
	)
	for _, profileIdx := range orderedIndices {
		profile := c.signingProfilesOrDefault()[profileIdx]
		signing := &signingContext{}
		lastSigning = signing

		req, reqErr := c.newRequest(ctx, method, contentType, payload)
		if reqErr != nil {
			lastErr = fmt.Errorf("build falabella %s request: %w", actionLabel, reqErr)
			continue
		}
		if signErr := c.signRequest(req, apiAction, signing, profile, queryParams); signErr != nil {
			lastErr = fmt.Errorf("sign falabella %s request: %w", actionLabel, signErr)
			continue
		}
		c.logRequestAttempt(apiAction, contentType, payload, signing)

		response, callErr := c.doRequest(req)
		if callErr != nil {
			lastErr = fmt.Errorf("falabella %s request (%s): %w", actionLabel, signing, callErr)
			continue
		}

		c.logFullResponse(apiAction, response.StatusCode, response.HTTPResponse, response.Body, signing)
		validationErr, retryable := validateActionResponse(actionLabel, response.StatusCode, response.Body, response.HTTPResponse)
		if validationErr == nil {
			c.activeSigningProfileIdx.Store(int32(profileIdx))
			return response.Body, signing, nil
		}

		wrapped := fmt.Errorf("%w (%s)", validationErr, signing)
		if isSignatureMismatch(response.Body) {
			lastErr = wrapped
			continue
		}
		if !retryable {
			return nil, signing, wrapped
		}

		lastErr = wrapped
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("falabella %s failed without diagnostics", actionLabel)
	}

	return nil, lastSigning, lastErr
}

// newRequest builds outbound Falabella requests.
func (c *Client) newRequest(ctx context.Context, method string, contentType string, payload []byte) (*http.Request, error) {
	endpoint, err := resolveActionEndpoint(c.cfg.URL)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if len(payload) > 0 {
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if strings.TrimSpace(c.cfg.UserAgent) != "" {
		req.Header.Set("User-Agent", c.cfg.UserAgent)
	}

	return req, nil
}

// resolveActionEndpoint resolves Falabella action endpoint URLs.
func resolveActionEndpoint(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.Path) == "" {
		parsed.Path = "/"
	} else if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed.String(), nil
}

// signRequest applies Falabella query signing parameters to outbound requests.
func (c *Client) signRequest(req *http.Request, action string, signing *signingContext, profile signingProfile, queryParams map[string]string) error {
	if req == nil || req.URL == nil {
		return errors.New("falabella request is nil")
	}

	profileFormat := strings.TrimSpace(profile.Format)
	timestamp := c.now().UTC().Format(profile.TimestampLayout)
	params := map[string]string{
		"Action":    action,
		"Timestamp": timestamp,
		"UserID":    c.cfg.UserID,
		"Version":   c.cfg.Version,
	}
	if profileFormat != "" {
		params["Format"] = profileFormat
	}
	for key, value := range queryParams {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue
		}
		params[trimmedKey] = trimmedValue
	}
	canonical := canonicalQuery(params)
	if strings.EqualFold(strings.TrimSpace(profile.SignatureInput), signatureInputRaw) {
		canonical = canonicalQueryRaw(params)
	}
	requestCanonical := canonicalQuery(params)
	signature := signCanonical(c.cfg.APIKey, canonical, profile.UppercaseSignature)
	if signing != nil {
		signing.Profile = profile.Name
		signing.Action = action
		signing.Canonical = canonical
		signing.Signature = signature
		signing.Timestamp = timestamp
		signing.UserAgent = c.cfg.UserAgent
		signing.Format = profileFormat
	}

	// Keep request query order aligned with signing-reference implementations:
	// canonical sorted params first, then Signature appended at the end.
	req.URL.RawQuery = requestCanonical + "&Signature=" + encodeRFC3986(signature)

	return nil
}

// productExists resolves whether one seller SKU already exists in Falabella.
func (c *Client) productExists(ctx context.Context, sku string) (bool, error) {
	trimmedSKU := strings.TrimSpace(sku)
	if trimmedSKU == "" {
		return false, errors.New("falabella product sku is required")
	}

	skuListRaw, marshalErr := json.Marshal([]string{trimmedSKU})
	if marshalErr != nil {
		return false, fmt.Errorf("marshal SkuSellerList query param: %w", marshalErr)
	}

	body, signing, err := c.executeAction(
		ctx,
		http.MethodGet,
		"get products",
		getProductsAction,
		"",
		nil,
		map[string]string{
			"SkuSellerList": string(skuListRaw),
			"Filter":        "all",
			"Limit":         "1",
		},
		c.xmlAndNoFormatSigningProfileIndices(),
	)
	if err != nil {
		return false, fmt.Errorf("%w (%s)", err, signing)
	}

	return responseContainsSellerSKU(body, trimmedSKU), nil
}

// responseContainsSellerSKU reports whether GetProducts payloads contain the provided SellerSku value.
func responseContainsSellerSKU(body []byte, sku string) bool {
	trimmedSKU := strings.TrimSpace(sku)
	trimmedBody := strings.TrimSpace(string(body))
	if trimmedSKU == "" || trimmedBody == "" {
		return false
	}

	if strings.HasPrefix(trimmedBody, "<") {
		return xmlContainsSellerSKU([]byte(trimmedBody), trimmedSKU)
	}
	if strings.HasPrefix(trimmedBody, "{") {
		return jsonContainsSellerSKU([]byte(trimmedBody), trimmedSKU)
	}

	return false
}

// xmlContainsSellerSKU reports whether XML payloads contain provided SellerSku values.
func xmlContainsSellerSKU(payload []byte, sku string) bool {
	decoder := xml.NewDecoder(bytes.NewReader(payload))
	inSellerSKU := false
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch typed := token.(type) {
		case xml.StartElement:
			inSellerSKU = strings.EqualFold(strings.TrimSpace(typed.Name.Local), "SellerSku")
		case xml.EndElement:
			if strings.EqualFold(strings.TrimSpace(typed.Name.Local), "SellerSku") {
				inSellerSKU = false
			}
		case xml.CharData:
			if !inSellerSKU {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(string(typed)), strings.TrimSpace(sku)) {
				return true
			}
		}
	}

	return false
}

// jsonContainsSellerSKU reports whether JSON payloads contain provided SellerSku values.
func jsonContainsSellerSKU(payload []byte, sku string) bool {
	var parsed any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return false
	}

	return anyContainsSellerSKU(parsed, sku)
}

// anyContainsSellerSKU recursively reports whether parsed payload values contain provided SellerSku values.
func anyContainsSellerSKU(value any, sku string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if strings.EqualFold(strings.TrimSpace(key), "SellerSku") {
				if skuValue, ok := item.(string); ok && strings.EqualFold(strings.TrimSpace(skuValue), strings.TrimSpace(sku)) {
					return true
				}
			}
			if anyContainsSellerSKU(item, sku) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if anyContainsSellerSKU(item, sku) {
				return true
			}
		}
	}

	return false
}

// doRequest executes outbound requests and normalizes body values.
func (c *Client) doRequest(req *http.Request) (actionCallResponse, error) {
	if c == nil || c.httpClient == nil {
		return actionCallResponse{}, errors.New("falabella http client is nil")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return actionCallResponse{}, err
	}
	if resp == nil {
		return actionCallResponse{}, errors.New("falabella response is nil")
	}

	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		return actionCallResponse{}, readErr
	}

	return actionCallResponse{
		StatusCode:   resp.StatusCode,
		Body:         body,
		HTTPResponse: resp,
	}, nil
}

// String renders signing-context diagnostics without exposing API-key values.
func (s *signingContext) String() string {
	if s == nil {
		return "signing_ctx=<nil>"
	}

	return fmt.Sprintf(
		"signing_ctx[profile=%s action=%s timestamp=%s signature=%s canonical=%s format=%s user_agent=%s]",
		s.Profile,
		s.Action,
		s.Timestamp,
		s.Signature,
		s.Canonical,
		s.Format,
		s.UserAgent,
	)
}

// signingProfilesOrDefault resolves configured signing profiles or defaults when missing.
func (c *Client) signingProfilesOrDefault() []signingProfile {
	if c == nil || len(c.signingProfiles) == 0 {
		return defaultSigningProfiles()
	}

	return c.signingProfiles
}

// activeSigningProfileIndex resolves active signing-profile index values.
func (c *Client) activeSigningProfileIndex() int {
	profiles := c.signingProfilesOrDefault()
	if len(profiles) == 0 {
		return 0
	}

	index := int(c.activeSigningProfileIdx.Load())
	if index < 0 || index >= len(profiles) {
		return 0
	}

	return index
}

// jsonSigningProfileIndices resolves JSON-format signing profile indexes.
func (c *Client) jsonSigningProfileIndices() []int {
	profiles := c.signingProfilesOrDefault()
	indices := make([]int, 0, len(profiles))
	for idx, profile := range profiles {
		if strings.EqualFold(strings.TrimSpace(profile.Format), "JSON") {
			indices = append(indices, idx)
		}
	}
	if len(indices) == 0 {
		indices = append(indices, c.activeSigningProfileIndex())
	}

	return indices
}

// xmlAndNoFormatSigningProfileIndices resolves signing profile indexes preferring XML/default format without JSON fallback.
func (c *Client) xmlAndNoFormatSigningProfileIndices() []int {
	profiles := c.signingProfilesOrDefault()
	indices := make([]int, 0, len(profiles))
	used := make(map[int]struct{}, len(profiles))

	appendByFormat := func(format string) {
		for idx, profile := range profiles {
			if !strings.EqualFold(strings.TrimSpace(profile.Format), format) {
				continue
			}
			if _, exists := used[idx]; exists {
				continue
			}
			indices = append(indices, idx)
			used[idx] = struct{}{}
		}
	}

	appendByFormat("XML")
	appendByFormat("")

	if len(indices) == 0 {
		indices = append(indices, c.activeSigningProfileIndex())
	}

	return indices
}

// xmlSigningProfileIndices resolves XML-format signing profile indexes.
func (c *Client) xmlSigningProfileIndices() []int {
	profiles := c.signingProfilesOrDefault()
	indices := make([]int, 0, len(profiles))
	for idx, profile := range profiles {
		if strings.EqualFold(strings.TrimSpace(profile.Format), "XML") {
			indices = append(indices, idx)
		}
	}
	if len(indices) == 0 {
		indices = append(indices, c.activeSigningProfileIndex())
	}

	return indices
}

// rotateProfileIndices rotates profile indexes to start with the currently active index when present.
func (c *Client) rotateProfileIndices(indices []int) []int {
	if len(indices) <= 1 {
		return indices
	}

	active := c.activeSigningProfileIndex()
	start := 0
	for idx, candidate := range indices {
		if candidate == active {
			start = idx
			break
		}
	}

	ordered := make([]int, 0, len(indices))
	for offset := 0; offset < len(indices); offset++ {
		ordered = append(ordered, indices[(start+offset)%len(indices)])
	}

	return ordered
}

// newClientWithDependencies creates client instances with injected dependencies.
func newClientWithDependencies(cfg Config, httpClient httpDoer, now func() time.Time) *Client {
	resolved := cfg
	if strings.TrimSpace(resolved.UserAgent) == "" {
		resolved.UserAgent = defaultUserAgent
	}
	if strings.TrimSpace(resolved.Version) == "" {
		resolved.Version = defaultVersion
	}

	profiles := defaultSigningProfiles()
	client := &Client{
		cfg:             resolved,
		httpClient:      httpClient,
		now:             now,
		signingProfiles: profiles,
	}
	client.activeSigningProfileIdx.Store(0)

	return client
}

// normalizeConfig resolves config defaults and validates mandatory values.
func normalizeConfig(cfg Config) (Config, error) {
	resolved := cfg
	resolved.URL = strings.TrimSpace(resolved.URL)
	resolved.UserID = strings.TrimSpace(resolved.UserID)
	resolved.APIKey = strings.TrimSpace(resolved.APIKey)
	resolved.UserAgent = strings.TrimSpace(resolved.UserAgent)
	resolved.Version = strings.TrimSpace(resolved.Version)

	if resolved.URL == "" {
		return Config{}, ErrMissingURL
	}
	if _, err := url.ParseRequestURI(resolved.URL); err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	if resolved.UserID == "" {
		return Config{}, ErrMissingUserID
	}
	if resolved.APIKey == "" {
		return Config{}, ErrMissingAPIKey
	}
	if resolved.Timeout <= 0 {
		resolved.Timeout = 5 * time.Second
	}
	if resolved.UserAgent == "" {
		resolved.UserAgent = defaultUserAgent
	}
	if resolved.Version == "" {
		resolved.Version = defaultVersion
	}

	return resolved, nil
}

// trimBody resolves trimmed response body strings for diagnostics.
func trimBody(body []byte) string {
	value := strings.TrimSpace(string(body))
	if value == "" {
		return "<empty>"
	}
	if len(value) > 1024 {
		return value[:1024] + "..."
	}

	return value
}

// validateActionResponse validates Falabella action response status/payload and surfaces retry behavior.
func validateActionResponse(action string, statusCode int, body []byte, response *http.Response) (error, bool) {
	if statusCode >= http.StatusBadRequest {
		message := trimBody(body)
		if parsed, ok := parseFalabellaError(body); ok {
			message = formatFalabellaError(parsed)
		}

		return fmt.Errorf("falabella %s status %d: %s%s", action, statusCode, message, httpContextSuffix(response, body)), isRetryableStatus(statusCode)
	}
	if len(body) == 0 {
		return fmt.Errorf("falabella %s response: %w%s", action, ErrEmptyResponse, httpContextSuffix(response, body)), true
	}
	if parsed, ok := parseFalabellaError(body); ok {
		return fmt.Errorf("falabella %s business error: %s", action, formatFalabellaError(parsed)), false
	}
	if hasBusinessError(body) {
		return fmt.Errorf("falabella %s business error: %s", action, trimBody(body)), false
	}

	return nil, false
}

// hasBusinessError reports whether Falabella response payload contains explicit error markers.
func hasBusinessError(body []byte) bool {
	if _, ok := parseFalabellaError(body); ok {
		return true
	}

	normalized := strings.ToLower(strings.TrimSpace(string(body)))
	if normalized == "" {
		return false
	}

	xmlMarkers := []string{
		"<errorresponse",
		"<responsetype>error</responsetype>",
		"<status>failed</status>",
		"<status>error</status>",
	}
	for _, marker := range xmlMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}

	compact := strings.Join(strings.Fields(normalized), "")
	jsonMarkers := []string{
		`"errorresponse":`,
		`"responsetype":"error"`,
		`"status":"failed"`,
		`"status":"error"`,
	}
	for _, marker := range jsonMarkers {
		if strings.Contains(compact, marker) {
			return true
		}
	}

	return false
}

// isSignatureMismatch reports whether Falabella response payload contains E007 signature-mismatch markers.
func isSignatureMismatch(body []byte) bool {
	if parsed, ok := parseFalabellaError(body); ok {
		normalizedCode := strings.ToLower(strings.TrimSpace(parsed.ErrorCode))
		normalizedMessage := strings.ToLower(strings.TrimSpace(parsed.ErrorMessage))
		if normalizedCode == "7" || strings.Contains(normalizedMessage, "e007") || strings.Contains(normalizedMessage, "signature") {
			return true
		}
	}

	normalized := strings.ToLower(strings.TrimSpace(string(body)))
	if normalized == "" {
		return false
	}

	compact := strings.Join(strings.Fields(normalized), "")
	return strings.Contains(compact, `"errorcode":"7"`) ||
		strings.Contains(compact, "signaturemismatch") ||
		strings.Contains(compact, "e007")
}

// parseFalabellaError extracts Falabella ErrorResponse heads from XML/JSON payloads.
func parseFalabellaError(body []byte) (falabellaErrorHead, bool) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return falabellaErrorHead{}, false
	}

	if strings.HasPrefix(trimmed, "<") {
		var parsed falabellaXMLErrorResponse
		if err := xml.Unmarshal([]byte(trimmed), &parsed); err == nil {
			if parsed.XMLName.Local == "ErrorResponse" && hasPopulatedErrorHead(parsed.Head) {
				return trimFalabellaErrorHead(parsed.Head), true
			}
		}
	}

	if strings.HasPrefix(trimmed, "{") {
		var parsed falabellaJSONErrorResponse
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			if hasPopulatedErrorHead(parsed.ErrorResponse.Head) {
				return trimFalabellaErrorHead(parsed.ErrorResponse.Head), true
			}
		}
	}

	return falabellaErrorHead{}, false
}

// hasPopulatedErrorHead reports whether any Falabella error-head fields are present.
func hasPopulatedErrorHead(head falabellaErrorHead) bool {
	return strings.TrimSpace(head.RequestAction) != "" ||
		strings.TrimSpace(head.ErrorType) != "" ||
		strings.TrimSpace(head.ErrorCode) != "" ||
		strings.TrimSpace(head.ErrorMessage) != ""
}

// trimFalabellaErrorHead trims Falabella error-head field values.
func trimFalabellaErrorHead(head falabellaErrorHead) falabellaErrorHead {
	return falabellaErrorHead{
		RequestAction: strings.TrimSpace(head.RequestAction),
		ErrorType:     strings.TrimSpace(head.ErrorType),
		ErrorCode:     strings.TrimSpace(head.ErrorCode),
		ErrorMessage:  strings.TrimSpace(head.ErrorMessage),
	}
}

// formatFalabellaError renders Falabella error-head values for diagnostics.
func formatFalabellaError(head falabellaErrorHead) string {
	parts := make([]string, 0, 4)
	if head.RequestAction != "" {
		parts = append(parts, fmt.Sprintf("request_action=%s", head.RequestAction))
	}
	if head.ErrorType != "" {
		parts = append(parts, fmt.Sprintf("error_type=%s", head.ErrorType))
	}
	if head.ErrorCode != "" {
		parts = append(parts, fmt.Sprintf("error_code=%s", head.ErrorCode))
	}
	if head.ErrorMessage != "" {
		parts = append(parts, fmt.Sprintf("error_message=%s", head.ErrorMessage))
	}
	if len(parts) == 0 {
		return "<empty>"
	}

	return strings.Join(parts, " ")
}

// isRetryableStatus reports whether status codes are retryable for guarded attempts.
func isRetryableStatus(statusCode int) bool {
	if statusCode >= http.StatusInternalServerError {
		return true
	}
	return statusCode == http.StatusTooManyRequests || statusCode == http.StatusRequestTimeout
}

// httpContextSuffix renders HTTP response diagnostics appended to action errors.
func httpContextSuffix(response *http.Response, body []byte) string {
	parts := make([]string, 0, 8)
	if response != nil {
		if requestID := firstHeader(response.Header, "X-Request-Id", "Cf-Ray"); requestID != "" {
			parts = append(parts, fmt.Sprintf("request_id=%s", requestID))
		}
		if contentType := firstHeader(response.Header, "Content-Type"); contentType != "" {
			parts = append(parts, fmt.Sprintf("content_type=%s", contentType))
		}
		if contentLength := firstHeader(response.Header, "Content-Length"); contentLength != "" {
			parts = append(parts, fmt.Sprintf("content_length=%s", contentLength))
		}
		if server := firstHeader(response.Header, "Server"); server != "" {
			parts = append(parts, fmt.Sprintf("server=%s", server))
		}
		if date := firstHeader(response.Header, "Date"); date != "" {
			parts = append(parts, fmt.Sprintf("date=%s", date))
		}
	}
	if len(body) == 0 {
		parts = append(parts, "response_body=empty")
	}
	if len(parts) == 0 {
		return ""
	}

	return " [" + strings.Join(parts, " ") + "]"
}

// firstHeader resolves the first non-empty header value from provided keys.
func firstHeader(headers http.Header, keys ...string) string {
	for _, key := range keys {
		values := headers.Values(key)
		if len(values) == 0 {
			continue
		}
		for _, value := range values {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}

	return ""
}

// logFullResponse logs complete Falabella response diagnostics when debug logging is enabled.
func (c *Client) logFullResponse(action string, statusCode int, response *http.Response, body []byte, signing *signingContext) {
	if c == nil || c.cfg.Logger == nil {
		return
	}
	if !c.cfg.Logger.Core().Enabled(zap.DebugLevel) {
		return
	}

	headers := http.Header{}
	if response != nil && response.Header != nil {
		headers = response.Header
	}

	c.cfg.Logger.Debug(
		"falabella full response",
		zap.String("action", action),
		zap.Int("status_code", statusCode),
		zap.Any("response_headers", headers),
		zap.Int("response_body_bytes", len(body)),
		zap.String("response_body", string(body)),
		zap.String("signing_ctx", signing.String()),
	)
}

// logRequestAttempt logs outbound Falabella request diagnostics when debug logging is enabled.
func (c *Client) logRequestAttempt(action string, contentType string, payload []byte, signing *signingContext) {
	if c == nil || c.cfg.Logger == nil {
		return
	}
	if !c.cfg.Logger.Core().Enabled(zap.DebugLevel) {
		return
	}

	c.cfg.Logger.Debug(
		"falabella request attempt",
		zap.String("action", action),
		zap.String("content_type", contentType),
		zap.Int("request_body_bytes", len(payload)),
		zap.String("request_body_sha256", payloadSHA256(payload)),
		zap.String("request_body_preview", trimBody(payload)),
		zap.Any("request_query_params", requestQueryParamsForLog(signing)),
		zap.Any("request_body_params", requestBodyParamsForLog(contentType, payload)),
		zap.String("signing_ctx", signing.String()),
	)
}

// requestQueryParamsForLog resolves outbound request query params from signing context.
func requestQueryParamsForLog(signing *signingContext) map[string]string {
	params := map[string]string{}
	if signing == nil {
		return params
	}

	for key, value := range parseCanonicalQuery(signing.Canonical) {
		params[key] = value
	}
	if strings.TrimSpace(signing.Signature) != "" {
		params["Signature"] = signing.Signature
	}

	return params
}

// requestBodyParamsForLog resolves outbound body params for diagnostics, excluding description fields.
func requestBodyParamsForLog(contentType string, payload []byte) map[string]any {
	params := map[string]any{}
	if len(payload) == 0 {
		return params
	}

	normalizedContentType := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.Contains(normalizedContentType, "xml"):
		return xmlBodyParamsForLog(payload)
	case strings.Contains(normalizedContentType, "json"):
		return jsonBodyParamsForLog(payload)
	default:
		return params
	}
}

// parseCanonicalQuery parses canonical query strings into decoded key/value maps.
func parseCanonicalQuery(canonical string) map[string]string {
	parsed := map[string]string{}
	trimmed := strings.TrimSpace(canonical)
	if trimmed == "" {
		return parsed
	}

	for _, pair := range strings.Split(trimmed, "&") {
		key, value, found := strings.Cut(pair, "=")
		if !found {
			continue
		}

		decodedKey := decodeCanonicalComponent(key)
		decodedValue := decodeCanonicalComponent(value)
		if strings.TrimSpace(decodedKey) == "" {
			continue
		}
		parsed[decodedKey] = decodedValue
	}

	return parsed
}

// decodeCanonicalComponent decodes canonical query values without converting "+" into spaces.
func decodeCanonicalComponent(value string) string {
	escaped := strings.ReplaceAll(value, "+", "%2B")
	decoded, err := url.QueryUnescape(escaped)
	if err != nil {
		return value
	}

	return decoded
}

// xmlBodyParamsForLog parses XML leaf params for diagnostics while excluding Description values.
func xmlBodyParamsForLog(payload []byte) map[string]any {
	values := map[string][]string{}
	decoder := xml.NewDecoder(bytes.NewReader(payload))
	stack := make([]string, 0, 8)

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch typed := token.(type) {
		case xml.StartElement:
			stack = append(stack, typed.Name.Local)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			key := strings.TrimSpace(stack[len(stack)-1])
			if key == "" || strings.EqualFold(key, "Description") {
				continue
			}

			value := strings.TrimSpace(string(typed))
			if value == "" {
				continue
			}
			values[key] = append(values[key], value)
		}
	}

	return flattenLoggedParams(values)
}

// jsonBodyParamsForLog parses JSON params for diagnostics while excluding Description values.
func jsonBodyParamsForLog(payload []byte) map[string]any {
	parsed := map[string]any{}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return map[string]any{}
	}

	sanitized := sanitizeJSONParams(parsed)
	result, _ := sanitized.(map[string]any)
	if result == nil {
		return map[string]any{}
	}

	return result
}

// sanitizeJSONParams recursively removes description fields from JSON payload params.
func sanitizeJSONParams(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := map[string]any{}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if strings.EqualFold(strings.TrimSpace(key), "Description") {
				continue
			}
			sanitized[key] = sanitizeJSONParams(typed[key])
		}
		return sanitized
	case []any:
		sanitized := make([]any, 0, len(typed))
		for _, item := range typed {
			sanitized = append(sanitized, sanitizeJSONParams(item))
		}
		return sanitized
	default:
		return value
	}
}

// flattenLoggedParams collapses repeated param values to scalar-or-slice representations.
func flattenLoggedParams(values map[string][]string) map[string]any {
	result := map[string]any{}
	if len(values) == 0 {
		return result
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		candidates := values[key]
		if len(candidates) == 0 {
			continue
		}

		if len(candidates) == 1 {
			result[key] = candidates[0]
			continue
		}
		result[key] = candidates
	}

	return result
}

// payloadSHA256 renders hexadecimal SHA256 digests for request payload diagnostics.
func payloadSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// defaultSigningProfiles defines ordered Falabella signing-profile fallback candidates.
func defaultSigningProfiles() []signingProfile {
	return []signingProfile{
		{
			Name:               "json_tz_nocolon_rfc3986_lower",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "JSON",
			UppercaseSignature: false,
			SignatureInput:     signatureInputRFC3986,
		},
		{
			Name:               "json_tz_nocolon_rfc3986_upper",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "JSON",
			UppercaseSignature: true,
			SignatureInput:     signatureInputRFC3986,
		},
		{
			Name:               "json_tz_nocolon_raw_lower",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "JSON",
			UppercaseSignature: false,
			SignatureInput:     signatureInputRaw,
		},
		{
			Name:               "json_tz_nocolon_raw_upper",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "JSON",
			UppercaseSignature: true,
			SignatureInput:     signatureInputRaw,
		},
		{
			Name:               "xml_tz_nocolon_rfc3986_lower",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "XML",
			UppercaseSignature: false,
			SignatureInput:     signatureInputRFC3986,
		},
		{
			Name:               "xml_tz_nocolon_rfc3986_upper",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "XML",
			UppercaseSignature: true,
			SignatureInput:     signatureInputRFC3986,
		},
		{
			Name:               "xml_tz_nocolon_raw_lower",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "XML",
			UppercaseSignature: false,
			SignatureInput:     signatureInputRaw,
		},
		{
			Name:               "xml_tz_nocolon_raw_upper",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "XML",
			UppercaseSignature: true,
			SignatureInput:     signatureInputRaw,
		},
		{
			Name:               "noformat_tz_nocolon_rfc3986_lower",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "",
			UppercaseSignature: false,
			SignatureInput:     signatureInputRFC3986,
		},
		{
			Name:               "noformat_tz_nocolon_rfc3986_upper",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "",
			UppercaseSignature: true,
			SignatureInput:     signatureInputRFC3986,
		},
		{
			Name:               "noformat_tz_nocolon_raw_lower",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "",
			UppercaseSignature: false,
			SignatureInput:     signatureInputRaw,
		},
		{
			Name:               "noformat_tz_nocolon_raw_upper",
			TimestampLayout:    timestampLayoutNoColon,
			Format:             "",
			UppercaseSignature: true,
			SignatureInput:     signatureInputRaw,
		},
	}
}
