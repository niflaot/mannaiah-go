package woocommerce

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"mannaiah/module/woocommerce/port"
)

// rawMeta defines tolerant metadata payload decoding values.
type rawMeta struct {
	// Key defines metadata key values.
	Key string `json:"key"`
	// Value defines metadata value payload values.
	Value any `json:"value"`
}

// rawLineItem defines tolerant raw order line-item payload values.
type rawLineItem struct {
	// Name defines line-item display-name values.
	Name string `json:"name"`
	// SKU defines line-item SKU values.
	SKU string `json:"sku"`
	// Quantity defines line-item quantity values.
	Quantity flexibleInt `json:"quantity"`
	// MetaData defines line-item metadata values.
	MetaData []rawMeta `json:"meta_data"`
	// Total defines line-item total values.
	Total flexibleFloat64 `json:"total"`
}

// rawShippingLine defines tolerant raw order shipping-line payload values.
type rawShippingLine struct {
	// MethodTitle defines shipping method title values.
	MethodTitle string `json:"method_title"`
	// MethodID defines shipping method identifier values.
	MethodID string `json:"method_id"`
	// Total defines shipping total values.
	Total flexibleFloat64 `json:"total"`
}

// rawFeeLine defines tolerant raw order fee-line payload values.
type rawFeeLine struct {
	// Name defines fee-line display-name values.
	Name string `json:"name"`
	// Total defines fee-line total values.
	Total flexibleFloat64 `json:"total"`
}

// listOrdersRaw performs tolerant order decoding for metadata values unsupported by SDK structs.
func (c *Client) listOrdersRaw(ctx context.Context, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error) {
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(pageSize))
	query.Set("order", "asc")
	query.Set("orderby", "id")
	query.Set("consumer_key", c.consumerKey)
	query.Set("consumer_secret", c.consumerSecret)

	endpoint := c.baseURL + "/wp-json/wc/v3/orders?" + query.Encode()
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if requestErr != nil {
		return nil, false, fmt.Errorf("create raw orders request: %w", requestErr)
	}

	httpClient := &http.Client{
		Timeout: c.timeout,
	}
	if !c.verifySSL {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	response, responseErr := httpClient.Do(request)
	if responseErr != nil {
		return nil, false, fmt.Errorf("execute raw orders request: %w", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, false, fmt.Errorf("raw orders request returned status %d", response.StatusCode)
	}

	type rawOrder struct {
		ID           int    `json:"id"`
		Status       string `json:"status"`
		DateCreated  string `json:"date_created"`
		DateModified string `json:"date_modified"`
		Billing      struct {
			Email     string `json:"email"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Company   string `json:"company"`
			Phone     string `json:"phone"`
			Address1  string `json:"address_1"`
			Address2  string `json:"address_2"`
			City      string `json:"city"`
		} `json:"billing"`
		Shipping struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Company   string `json:"company"`
			Phone     string `json:"phone"`
			Address1  string `json:"address_1"`
			Address2  string `json:"address_2"`
			City      string `json:"city"`
		} `json:"shipping"`
		CustomerNote  string            `json:"customer_note"`
		LineItems     []rawLineItem     `json:"line_items"`
		ShippingLines []rawShippingLine `json:"shipping_lines"`
		FeeLines      []rawFeeLine      `json:"fee_lines"`
		MetaData      []rawMeta         `json:"meta_data"`
	}

	var payload []rawOrder
	if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
		return nil, false, fmt.Errorf("decode raw orders response: %w", decodeErr)
	}

	result := make([]port.WooOrder, 0, len(payload))
	for _, item := range payload {
		metadata := map[string]string{}
		for _, meta := range item.MetaData {
			key := strings.TrimSpace(meta.Key)
			if key == "" {
				continue
			}
			metadata[key] = normalizeMetadataValue(meta.Value)
		}

		result = append(result, port.WooOrder{
			ID:                     item.ID,
			Status:                 strings.TrimSpace(item.Status),
			BillingEmail:           strings.TrimSpace(item.Billing.Email),
			BillingFirstName:       strings.TrimSpace(item.Billing.FirstName),
			BillingLastName:        strings.TrimSpace(item.Billing.LastName),
			BillingCompany:         strings.TrimSpace(item.Billing.Company),
			BillingPhone:           strings.TrimSpace(item.Billing.Phone),
			BillingAddress1:        strings.TrimSpace(item.Billing.Address1),
			BillingAddress2:        strings.TrimSpace(item.Billing.Address2),
			BillingCity:            strings.TrimSpace(item.Billing.City),
			BillingAddressLine1:    strings.TrimSpace(item.Billing.Address1),
			BillingAddressLine2:    strings.TrimSpace(item.Billing.Address2),
			BillingCityCode:        strings.TrimSpace(item.Billing.City),
			BillingPhoneNormalized: strings.TrimSpace(item.Billing.Phone),
			ShippingFirstName:      strings.TrimSpace(item.Shipping.FirstName),
			ShippingLastName:       strings.TrimSpace(item.Shipping.LastName),
			ShippingCompany:        strings.TrimSpace(item.Shipping.Company),
			ShippingPhone:          strings.TrimSpace(item.Shipping.Phone),
			ShippingAddressLine1:   strings.TrimSpace(item.Shipping.Address1),
			ShippingAddressLine2:   strings.TrimSpace(item.Shipping.Address2),
			ShippingCityCode:       strings.TrimSpace(item.Shipping.City),
			Items:                  append(mapRawOrderItems(item.LineItems), mapRawFeeItems(item.FeeLines)...),
			ShippingCharges:        mapRawShippingCharges(item.ShippingLines),
			Comments:               mapRawOrderComments(item.CustomerNote, item.DateModified, item.DateCreated),
			CreatedAt:              parseWooOrderTime(item.DateCreated),
			Metadata:               metadata,
		})
	}

	totalPages, _ := strconv.Atoi(response.Header.Get("X-Wp-Totalpages"))
	isLastPage := page >= totalPages && totalPages > 0
	return result, resolveHasNextPage(page, pageSize, len(result), totalPages, isLastPage), nil
}
