package woocommerce

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	coretelemetry "mannaiah/module/core/telemetry"
	"mannaiah/module/woocommerce/port"
)

// rawCouponPayload defines tolerant raw WooCommerce coupon payload values.
type rawCouponPayload struct {
	// ID defines coupon identifier values.
	ID int `json:"id"`
	// Code defines coupon code values.
	Code string `json:"code"`
	// DiscountType defines WooCommerce discount type values.
	DiscountType string `json:"discount_type"`
	// Amount defines discount amount values (WooCommerce returns as string).
	Amount string `json:"amount"`
	// UsageLimit defines global usage limit values (0 = unlimited).
	UsageLimit *int `json:"usage_limit"`
	// UsageLimitPerUser defines per-user usage limit values (0 = unlimited).
	UsageLimitPerUser *int `json:"usage_limit_per_user"`
	// UsageCount defines current redemption counts.
	UsageCount int `json:"usage_count"`
	// ProductIDs defines restricted product WooCommerce IDs.
	ProductIDs []int `json:"product_ids"`
	// ProductCategories defines restricted category WooCommerce IDs.
	ProductCategories []int `json:"product_categories"`
	// EmailRestrictions defines restricted email values.
	EmailRestrictions []string `json:"email_restrictions"`
	// MetaData defines coupon metadata payload values.
	MetaData []rawMeta `json:"meta_data"`
	// DateCreated defines coupon creation timestamp values.
	DateCreated string `json:"date_created"`
	// DateModified defines coupon modification timestamp values.
	DateModified string `json:"date_modified"`
}

// ListCoupons retrieves paginated WooCommerce coupon values.
func (c *Client) ListCoupons(ctx context.Context, page int, pageSize int) (coupons []port.WooCoupon, hasNext bool, err error) {
	startedAt := time.Now()
	spanCtx, span := coretelemetry.StartSpan(ctx, "mannaiah/dependency", "woocommerce.list_coupons")
	defer func() {
		coretelemetry.RecordDependency("woocommerce", "list_coupons", startedAt, err)
		coretelemetry.EndSpan(span, err)
	}()

	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(pageSize))
	query.Set("order", "asc")
	query.Set("orderby", "id")

	endpoint := c.baseURL + "/wp-json/wc/v3/coupons?" + query.Encode()
	request, requestErr := http.NewRequestWithContext(spanCtx, http.MethodGet, endpoint, nil)
	if requestErr != nil {
		return nil, false, fmt.Errorf("create list coupons request: %w", requestErr)
	}
	c.applyWooAuth(request)

	response, responseErr := c.rawHTTPClient().Do(request)
	if responseErr != nil {
		return nil, false, fmt.Errorf("execute list coupons request: %w", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, false, fmt.Errorf("list coupons request returned status %d", response.StatusCode)
	}

	var payload []rawCouponPayload
	if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
		return nil, false, fmt.Errorf("decode list coupons response: %w", decodeErr)
	}

	result := make([]port.WooCoupon, 0, len(payload))
	for _, item := range payload {
		result = append(result, mapRawCoupon(item))
	}

	totalPages, _ := strconv.Atoi(response.Header.Get("X-Wp-Totalpages"))
	isLastPage := page >= totalPages && totalPages > 0
	return result, resolveHasNextPage(page, pageSize, len(result), totalPages, isLastPage), nil
}

// GetCouponByID retrieves one WooCommerce coupon by identifier.
func (c *Client) GetCouponByID(ctx context.Context, id int) (coupon port.WooCoupon, err error) {
	startedAt := time.Now()
	spanCtx, span := coretelemetry.StartSpan(ctx, "mannaiah/dependency", "woocommerce.get_coupon_by_id")
	defer func() {
		coretelemetry.RecordDependency("woocommerce", "get_coupon_by_id", startedAt, err)
		coretelemetry.EndSpan(span, err)
	}()

	endpoint := fmt.Sprintf("%s/wp-json/wc/v3/coupons/%d", c.baseURL, id)
	request, requestErr := http.NewRequestWithContext(spanCtx, http.MethodGet, endpoint, nil)
	if requestErr != nil {
		return port.WooCoupon{}, fmt.Errorf("create get coupon request: %w", requestErr)
	}
	c.applyWooAuth(request)

	response, responseErr := c.rawHTTPClient().Do(request)
	if responseErr != nil {
		return port.WooCoupon{}, fmt.Errorf("execute get coupon request: %w", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return port.WooCoupon{}, fmt.Errorf("woocommerce coupon %d not found", id)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return port.WooCoupon{}, fmt.Errorf("get coupon request returned status %d", response.StatusCode)
	}

	var payload rawCouponPayload
	if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
		return port.WooCoupon{}, fmt.Errorf("decode get coupon response: %w", decodeErr)
	}

	return mapRawCoupon(payload), nil
}

// UpsertCoupon creates or updates a WooCommerce coupon from a sync command.
func (c *Client) UpsertCoupon(ctx context.Context, command port.CouponSyncCommand) (result port.CouponSyncResult, err error) {
	startedAt := time.Now()
	spanCtx, span := coretelemetry.StartSpan(ctx, "mannaiah/dependency", "woocommerce.upsert_coupon")
	defer func() {
		coretelemetry.RecordDependency("woocommerce", "upsert_coupon", startedAt, err)
		coretelemetry.EndSpan(span, err)
	}()

	body := buildCouponRequestBody(command)
	bodyBytes, marshalErr := json.Marshal(body)
	if marshalErr != nil {
		return port.CouponSyncResult{}, fmt.Errorf("marshal coupon request body: %w", marshalErr)
	}

	var endpoint string
	var method string
	if command.WooCommerceID != nil && *command.WooCommerceID > 0 {
		endpoint = fmt.Sprintf("%s/wp-json/wc/v3/coupons/%d", c.baseURL, *command.WooCommerceID)
		method = http.MethodPut
	} else {
		endpoint = c.baseURL + "/wp-json/wc/v3/coupons"
		method = http.MethodPost
	}

	request, requestErr := http.NewRequestWithContext(spanCtx, method, endpoint, bytes.NewReader(bodyBytes))
	if requestErr != nil {
		return port.CouponSyncResult{}, fmt.Errorf("create upsert coupon request: %w", requestErr)
	}
	request.Header.Set("Content-Type", "application/json")
	c.applyWooAuth(request)

	response, responseErr := c.rawHTTPClient().Do(request)
	if responseErr != nil {
		return port.CouponSyncResult{}, fmt.Errorf("execute upsert coupon request: %w", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return port.CouponSyncResult{}, fmt.Errorf("upsert coupon request returned status %d", response.StatusCode)
	}

	var payload rawCouponPayload
	if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
		return port.CouponSyncResult{}, fmt.Errorf("decode upsert coupon response: %w", decodeErr)
	}

	created := method == http.MethodPost
	return port.CouponSyncResult{
		WooCommerceID: payload.ID,
		Created:       created,
	}, nil
}

// buildCouponRequestBody maps a sync command to a WooCommerce coupon request payload.
func buildCouponRequestBody(command port.CouponSyncCommand) map[string]any {
	body := map[string]any{
		"code":          strings.TrimSpace(strings.ToLower(command.Code)),
		"discount_type": mapDiscountType(command.DiscountType),
		"amount":        strconv.FormatFloat(command.DiscountAmount, 'f', 2, 64),
	}

	if command.MaxUsagesGlobal != nil {
		body["usage_limit"] = *command.MaxUsagesGlobal
	} else {
		body["usage_limit"] = nil
	}

	if command.MaxUsagesPerEmail != nil {
		body["usage_limit_per_user"] = *command.MaxUsagesPerEmail
	} else {
		body["usage_limit_per_user"] = nil
	}

	if len(command.AssignedEmails) > 0 {
		body["email_restrictions"] = command.AssignedEmails
	} else {
		body["email_restrictions"] = []string{}
	}

	if len(command.IncludedProductWooIDs) > 0 {
		body["product_ids"] = command.IncludedProductWooIDs
	} else {
		body["product_ids"] = []int{}
	}

	if len(command.IncludedCategoryWooIDs) > 0 {
		body["product_categories"] = command.IncludedCategoryWooIDs
	} else {
		body["product_categories"] = []int{}
	}

	return body
}

// mapDiscountType maps internal discount type values to WooCommerce discount type values.
func mapDiscountType(discountType string) string {
	switch strings.TrimSpace(discountType) {
	case "percentage":
		return "percent"
	case "fixed":
		return "fixed_cart"
	default:
		return "fixed_cart"
	}
}

// mapRawCoupon maps tolerant raw coupon payload values to transport coupon values.
func mapRawCoupon(item rawCouponPayload) port.WooCoupon {
	metadata := map[string]string{}
	for _, meta := range item.MetaData {
		key := strings.TrimSpace(meta.Key)
		if key == "" {
			continue
		}
		metadata[key] = normalizeMetadataValue(meta.Value)
	}

	usageLimit := 0
	if item.UsageLimit != nil {
		usageLimit = *item.UsageLimit
	}
	usageLimitPerUser := 0
	if item.UsageLimitPerUser != nil {
		usageLimitPerUser = *item.UsageLimitPerUser
	}

	return port.WooCoupon{
		ID:                item.ID,
		Code:              strings.TrimSpace(strings.ToLower(item.Code)),
		DiscountType:      strings.TrimSpace(item.DiscountType),
		Amount:            strings.TrimSpace(item.Amount),
		UsageLimit:        usageLimit,
		UsageLimitPerUser: usageLimitPerUser,
		UsageCount:        item.UsageCount,
		ProductIDs:        item.ProductIDs,
		ProductCategories: item.ProductCategories,
		EmailRestrictions: item.EmailRestrictions,
		MetaData:          metadata,
		DateCreated:       parseWooOrderTime(item.DateCreated),
		DateModified:      parseWooOrderTime(item.DateModified),
	}
}
