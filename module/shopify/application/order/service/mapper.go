package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	contactsdomain "mannaiah/module/contacts/domain"
	"mannaiah/module/core/citycode"
	ordersdomain "mannaiah/module/orders/domain"
	shopifyport "mannaiah/module/shopify/port"
)

// BuildOrderContactSyncCommand maps one Shopify order into normalized contact values.
func BuildOrderContactSyncCommand(order shopifyport.ShopifyOrder) (shopifyport.ContactSyncCommand, error) {
	email := preferString(order.ContactEmail, customerEmail(order.Customer))
	if email == "" {
		return shopifyport.ContactSyncCommand{}, ErrOrderContactEmailRequired
	}

	documentNumber := extractDefaultAddressDocumentNumber(order.Customer)
	var documentType contactsdomain.DocumentType
	if documentNumber != "" {
		documentType = contactsdomain.DocumentTypeCC
	}
	address := resolvePrimaryAddress(order)
	firstName := preferString(customerFirstName(order.Customer), addressFirstName(address), billingFirstName(order.BillingAddress))
	lastName := preferString(customerLastName(order.Customer), addressLastName(address), billingLastName(order.BillingAddress))
	legalName := ""
	if documentType == contactsdomain.DocumentTypeNIT {
		legalName = preferString(addressCompany(address), billingCompany(order.BillingAddress))
		firstName = ""
		lastName = ""
	}
	if legalName == "" {
		firstName = preferString(firstName, "Shopify")
		lastName = preferString(lastName, "Customer")
	}

	command := shopifyport.ContactSyncCommand{
		ShopDomain:     strings.TrimSpace(order.ShopDomain),
		ShopifyID:      customerID(order.Customer),
		Email:          email,
		DocumentType:   documentType,
		DocumentNumber: documentNumber,
		LegalName:      legalName,
		FirstName:      firstName,
		LastName:       lastName,
		Phone:          preferString(customerPhone(order.Customer), addressPhone(address), billingPhone(order.BillingAddress)),
		Address:        addressLine1(address),
		AddressExtra:   addressLine2(address),
		CityCode:       resolveAddressCityCode(address),
		Metadata:       buildOrderContactMetadata(order),
	}
	if order.Customer != nil && !order.Customer.CreatedAt.IsZero() {
		createdAt := order.Customer.CreatedAt.UTC()
		command.CreatedAt = &createdAt
	}

	return command, nil
}

// BuildOrderSyncCommand maps one Shopify order into normalized mainstream order values.
func BuildOrderSyncCommand(order shopifyport.ShopifyOrder, contactID string, realm string, trigger string) shopifyport.OrderSyncCommand {
	metadata := buildOrderMetadata(order)
	shippingAddress := buildShippingAddress(order)
	addShippingCityResolutionMetadata(metadata, resolvePrimaryAddress(order), shippingAddress)

	command := shopifyport.OrderSyncCommand{
		ShopDomain:        strings.TrimSpace(order.ShopDomain),
		ShopifyID:         strings.TrimSpace(order.ID),
		Identifier:        resolveOrderIdentifier(order),
		Realm:             resolveRealm(realm),
		ContactID:         strings.TrimSpace(contactID),
		Items:             buildOrderItems(order.LineItems),
		Status:            mapOrderStatus(order),
		StatusDescription: buildStatusDescription(order),
		ShippingAddress:   shippingAddress,
		ShippingCharges:   buildShippingCharges(order.ShippingLines),
		AppliedCoupons:    buildAppliedCoupons(order.DiscountCodes, order.CreatedAt),
		PaymentMethod:     strings.Join(order.PaymentGatewayNames, ", "),
		Metadata:          metadata,
		Source:            syncMutationSource,
	}
	if !order.CreatedAt.IsZero() {
		createdAt := order.CreatedAt.UTC()
		command.CreatedAt = &createdAt
	}

	return command
}

func mapOrderStatus(order shopifyport.ShopifyOrder) ordersdomain.Status {
	if order.CancelledAt != nil || strings.TrimSpace(order.CancelReason) != "" {
		return ordersdomain.StatusCancelled
	}
	fulfillmentStatus := strings.ToLower(strings.TrimSpace(order.FulfillmentStatus))
	financialStatus := strings.ToLower(strings.TrimSpace(order.FinancialStatus))

	switch fulfillmentStatus {
	case "fulfilled":
		return ordersdomain.StatusCompleted
	case "on_hold", "restocked":
		return ordersdomain.StatusHold
	}

	switch financialStatus {
	case "paid":
		return ordersdomain.StatusCreated
	case "voided", "refunded":
		return ordersdomain.StatusCancelled
	case "pending", "authorized", "partially_paid":
		if isCODOrder(order) {
			return ordersdomain.StatusCreated
		}
		return ordersdomain.StatusPending
	}

	return ordersdomain.StatusCreated
}

func shouldMarkCODOrderPaid(order shopifyport.ShopifyOrder) bool {
	if order.CancelledAt != nil || strings.TrimSpace(order.CancelReason) != "" {
		return false
	}
	if !isCODOrder(order) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(order.FinancialStatus)) {
	case "pending", "authorized", "partially_paid":
		return true
	default:
		return false
	}
}

func isCODOrder(order shopifyport.ShopifyOrder) bool {
	for _, gateway := range order.PaymentGatewayNames {
		if isCODPaymentGateway(gateway) {
			return true
		}
	}
	return false
}

func isCODPaymentGateway(value string) bool {
	normalized := normalizeCODPaymentGateway(value)
	if normalized == "" {
		return false
	}
	if strings.Contains(normalized, "cod") {
		return true
	}
	if strings.Contains(normalized, "cash on delivery") {
		return true
	}
	if strings.Contains(normalized, "contra entrega") || strings.Contains(normalized, "contraentrega") {
		return true
	}
	return strings.Contains(normalized, "pago contra entrega")
}

func normalizeCODPaymentGateway(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"á", "a",
		"é", "e",
		"í", "i",
		"ó", "o",
		"ú", "u",
		"ü", "u",
		"ñ", "n",
		"-", " ",
		"_", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func buildStatusDescription(order shopifyport.ShopifyOrder) string {
	return fmt.Sprintf(
		"Shopify sync mapped status from financial=%s fulfillment=%s",
		strings.TrimSpace(order.FinancialStatus),
		strings.TrimSpace(order.FulfillmentStatus),
	)
}

func buildOrderItems(values []shopifyport.ShopifyLineItem) []shopifyport.OrderSyncItemCommand {
	items := make([]shopifyport.OrderSyncItemCommand, 0, len(values))
	for _, value := range values {
		items = append(items, shopifyport.OrderSyncItemCommand{
			SKU:              strings.TrimSpace(value.SKU),
			AlternateName:    buildAlternateName(value.Title, value.VariantTitle, value.ColorLabel),
			ProductID:        strings.TrimSpace(value.MannaiahProductID),
			ShopifyProductID: strings.TrimSpace(value.ProductID),
			ShopifyVariantID: strings.TrimSpace(value.VariantID),
			Quantity:         value.Quantity,
			Value:            parseMoney(value.Price),
		})
	}

	return items
}

func buildShippingAddress(order shopifyport.ShopifyOrder) *shopifyport.OrderSyncShippingAddressCommand {
	address := resolvePrimaryAddress(order)
	if address == nil {
		return nil
	}
	if addressLine1(address) == "" && addressCity(address) == "" && addressPhone(address) == "" {
		return nil
	}

	return &shopifyport.OrderSyncShippingAddressCommand{
		Address:  addressLine1(address),
		Address2: addressLine2(address),
		Phone:    addressPhone(address),
		CityCode: resolveAddressCityCode(address),
	}
}

func resolveAddressCityCode(address *shopifyport.ShopifyAddress) string {
	if address == nil {
		return "-1"
	}
	result, err := citycode.ResolveDetailed(context.Background(), addressCity(address), addressProvince(address))
	if err != nil || !result.Found {
		return citycode.Resolve(addressCity(address))
	}
	return result.Code
}

func addShippingCityResolutionMetadata(metadata map[string]string, address *shopifyport.ShopifyAddress, shippingAddress *shopifyport.OrderSyncShippingAddressCommand) {
	if metadata == nil || address == nil || shippingAddress == nil {
		return
	}
	result, err := citycode.ResolveDetailed(context.Background(), addressCity(address), addressProvince(address))
	if err == nil && result.Found {
		metadata["shipping_city_resolution_status"] = "resolved"
		metadata["shipping_city_resolution_code"] = result.Code
		metadata["shipping_city_resolution_name"] = result.Name
		metadata["shipping_city_resolution_department"] = result.Department
		return
	}
	metadata["shipping_city_resolution_status"] = "unresolved"
	metadata["shipping_city_resolution_input_city"] = addressCity(address)
	metadata["shipping_city_resolution_input_department"] = addressProvince(address)
	if err != nil {
		metadata["shipping_city_resolution_reason"] = "error"
		metadata["shipping_city_resolution_error"] = err.Error()
		return
	}
	metadata["shipping_city_resolution_reason"] = result.Reason
	if len(result.Suggestions) > 0 {
		metadata["shipping_city_resolution_suggestions"] = formatCityResolutionSuggestions(result.Suggestions)
	}
}

func formatCityResolutionSuggestions(values []citycode.ResolveSuggestion) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strings.TrimSpace(value.Name)+" ("+strings.TrimSpace(value.Code)+", "+strings.TrimSpace(value.Department)+")")
	}
	return strings.Join(parts, "; ")
}

func buildShippingCharges(values []shopifyport.ShopifyShippingLine) []shopifyport.OrderSyncShippingChargeCommand {
	charges := make([]shopifyport.OrderSyncShippingChargeCommand, 0, len(values))
	for _, value := range values {
		charges = append(charges, shopifyport.OrderSyncShippingChargeCommand{
			MethodID:    strings.TrimSpace(value.Code),
			MethodTitle: strings.TrimSpace(value.Title),
			Price:       parseMoney(value.Price),
		})
	}

	return charges
}

func buildAppliedCoupons(values []shopifyport.ShopifyDiscountCode, createdAt time.Time) []shopifyport.OrderSyncAppliedCouponCommand {
	coupons := make([]shopifyport.OrderSyncAppliedCouponCommand, 0, len(values))
	for _, value := range values {
		coupon := shopifyport.OrderSyncAppliedCouponCommand{
			Code:           strings.TrimSpace(value.Code),
			DiscountType:   strings.TrimSpace(value.Type),
			DiscountAmount: parseMoney(value.Amount),
		}
		if !createdAt.IsZero() {
			appliedAt := createdAt.UTC()
			coupon.AppliedAt = &appliedAt
		}
		coupons = append(coupons, coupon)
	}

	return coupons
}

func buildOrderMetadata(order shopifyport.ShopifyOrder) map[string]string {
	metadata := map[string]string{}
	if order.ID != "" {
		metadata["shopify_order_id"] = strings.TrimSpace(order.ID)
	}
	if order.Name != "" {
		metadata["shopify_order_name"] = strings.TrimSpace(order.Name)
	}
	if order.FinancialStatus != "" {
		metadata["shopify_financial_status"] = strings.TrimSpace(order.FinancialStatus)
	}
	if order.FulfillmentStatus != "" {
		metadata["shopify_fulfillment_status"] = strings.TrimSpace(order.FulfillmentStatus)
	}
	if order.Currency != "" {
		metadata["shopify_currency"] = strings.TrimSpace(order.Currency)
	}
	if order.Tags != "" {
		metadata["shopify_order_tags"] = strings.TrimSpace(order.Tags)
	}
	if order.TotalPrice != "" {
		metadata["shopify_total_price"] = strings.TrimSpace(order.TotalPrice)
	}
	if order.TotalTax != "" {
		metadata["shopify_total_tax"] = strings.TrimSpace(order.TotalTax)
	}
	if order.TotalDiscounts != "" {
		metadata["shopify_total_discounts"] = strings.TrimSpace(order.TotalDiscounts)
	}
	if len(order.PaymentGatewayNames) > 0 {
		metadata["shopify_payment_gateway_names"] = strings.Join(order.PaymentGatewayNames, ", ")
	}

	return metadata
}

func buildOrderContactMetadata(order shopifyport.ShopifyOrder) map[string]string {
	metadata := map[string]string{}
	if order.Customer != nil && order.Customer.ID != "" {
		metadata["shopify_customer_id"] = strings.TrimSpace(order.Customer.ID)
	}
	if order.Customer != nil && order.Customer.Tags != "" {
		metadata["shopify_customer_tags"] = strings.TrimSpace(order.Customer.Tags)
	}
	if !order.CreatedAt.IsZero() {
		metadata["privacy.accepted"] = "true"
		metadata["privacy.acceptedDate"] = order.CreatedAt.UTC().Format(time.RFC3339)
	}
	addOrderContactMarketingMetadata(metadata, order)
	return metadata
}

func addOrderContactMarketingMetadata(metadata map[string]string, order shopifyport.ShopifyOrder) {
	emailState := ""
	var emailConsentAt *time.Time
	smsState := ""
	var smsConsentAt *time.Time
	if order.Customer != nil {
		emailState = strings.TrimSpace(order.Customer.EmailMarketingState)
		emailConsentAt = order.Customer.EmailMarketingConsentUpdatedAt
		smsState = strings.TrimSpace(order.Customer.SMSMarketingState)
		smsConsentAt = order.Customer.SMSMarketingConsentUpdatedAt
	}
	if emailState != "" {
		metadata["shopify_email_marketing_state"] = emailState
	}
	if smsState != "" {
		metadata["shopify_sms_marketing_state"] = smsState
	}
	optedIn := isOrderMarketingOptedIn(emailState) || isOrderMarketingOptedIn(smsState) || hasSubscriptionMarker(order)
	metadata["membership.opt_in"] = strconv.FormatBool(optedIn)
	if !optedIn {
		return
	}
	consentedAt := firstOrderConsentTime(emailConsentAt, smsConsentAt)
	if consentedAt == nil && !order.CreatedAt.IsZero() {
		createdAt := order.CreatedAt.UTC()
		consentedAt = &createdAt
	}
	if consentedAt != nil {
		metadata["membership.opt_in_date"] = consentedAt.UTC().Format(time.RFC3339)
	}
}

func firstOrderConsentTime(values ...*time.Time) *time.Time {
	for _, value := range values {
		if value != nil && !value.IsZero() {
			resolved := value.UTC()
			return &resolved
		}
	}
	return nil
}

func isOrderMarketingOptedIn(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "subscribed", "confirmed", "accepted", "opted_in", "opted-in", "sms_marketing_subscribed":
		return true
	default:
		return false
	}
}

func hasSubscriptionMarker(order shopifyport.ShopifyOrder) bool {
	for _, attr := range append(order.NoteAttributes, customerNoteAttributes(order.Customer)...) {
		name := strings.ToLower(strings.TrimSpace(attr.Name))
		value := strings.ToLower(strings.TrimSpace(attr.Value))
		if name == "" {
			continue
		}
		if strings.Contains(name, "subscription") || strings.Contains(name, "newsletter") || strings.Contains(name, "membership") || strings.Contains(name, "circle") {
			return value == "true" || value == "1" || value == "yes" || value == "si" || value == "sí" || value == "accepted" || value == "subscribed"
		}
	}
	return false
}

func customerNoteAttributes(customer *shopifyport.ShopifyCustomer) []shopifyport.ShopifyNoteAttribute {
	if customer == nil {
		return nil
	}
	return customer.NoteAttributes
}

func resolveOrderIdentifier(order shopifyport.ShopifyOrder) string {
	return preferString(order.Name, order.ID)
}

func resolveRealm(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return "shopify"
}

func buildAlternateName(title string, variantTitle string, colorLabel string) string {
	trimmedTitle := strings.TrimSpace(title)
	trimmedVariant := strings.TrimSpace(variantTitle)
	trimmedColor := strings.TrimSpace(colorLabel)
	colorSuffix := strings.ToUpper(trimmedColor)
	base := ""
	if trimmedTitle == "" {
		base = trimmedVariant
	} else if trimmedVariant == "" || strings.EqualFold(trimmedVariant, "default title") || strings.EqualFold(trimmedVariant, trimmedTitle) {
		base = trimmedTitle
	} else {
		base = trimmedTitle + " - " + trimmedVariant
	}
	if colorSuffix == "" || strings.Contains(strings.ToLower(base), strings.ToLower(trimmedColor)) {
		return base
	}

	return strings.TrimSpace(base + " " + colorSuffix)
}

func parseMoney(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

func resolvePrimaryAddress(order shopifyport.ShopifyOrder) *shopifyport.ShopifyAddress {
	if order.ShippingAddress != nil {
		return order.ShippingAddress
	}
	if order.BillingAddress != nil {
		return order.BillingAddress
	}
	if order.Customer != nil {
		return order.Customer.DefaultAddress
	}
	return nil
}

func customerID(customer *shopifyport.ShopifyCustomer) string {
	if customer == nil {
		return ""
	}
	return strings.TrimSpace(customer.ID)
}

func customerEmail(customer *shopifyport.ShopifyCustomer) string {
	if customer == nil {
		return ""
	}
	return strings.TrimSpace(customer.Email)
}

func customerFirstName(customer *shopifyport.ShopifyCustomer) string {
	if customer == nil {
		return ""
	}
	return strings.TrimSpace(customer.FirstName)
}

func customerLastName(customer *shopifyport.ShopifyCustomer) string {
	if customer == nil {
		return ""
	}
	return strings.TrimSpace(customer.LastName)
}

func customerPhone(customer *shopifyport.ShopifyCustomer) string {
	if customer == nil {
		return ""
	}
	return strings.TrimSpace(customer.Phone)
}

func addressFirstName(address *shopifyport.ShopifyAddress) string {
	if address == nil {
		return ""
	}
	return strings.TrimSpace(address.FirstName)
}

func addressLastName(address *shopifyport.ShopifyAddress) string {
	if address == nil {
		return ""
	}
	return strings.TrimSpace(address.LastName)
}

func addressCompany(address *shopifyport.ShopifyAddress) string {
	if address == nil {
		return ""
	}
	return strings.TrimSpace(address.Company)
}

func addressLine1(address *shopifyport.ShopifyAddress) string {
	if address == nil {
		return ""
	}
	return strings.TrimSpace(address.Address1)
}

func addressLine2(address *shopifyport.ShopifyAddress) string {
	if address == nil {
		return ""
	}
	return strings.TrimSpace(address.Address2)
}

func addressPhone(address *shopifyport.ShopifyAddress) string {
	if address == nil {
		return ""
	}
	return strings.TrimSpace(address.Phone)
}

func addressCity(address *shopifyport.ShopifyAddress) string {
	if address == nil {
		return ""
	}
	return strings.TrimSpace(address.City)
}

func addressProvince(address *shopifyport.ShopifyAddress) string {
	if address == nil {
		return ""
	}
	return strings.TrimSpace(address.Province)
}

func billingFirstName(address *shopifyport.ShopifyAddress) string {
	return addressFirstName(address)
}

func billingLastName(address *shopifyport.ShopifyAddress) string {
	return addressLastName(address)
}

func billingCompany(address *shopifyport.ShopifyAddress) string {
	return addressCompany(address)
}

func billingPhone(address *shopifyport.ShopifyAddress) string {
	return addressPhone(address)
}

func extractDefaultAddressDocumentNumber(customer *shopifyport.ShopifyCustomer) string {
	if customer == nil || customer.DefaultAddress == nil {
		return ""
	}
	var digits strings.Builder
	for _, r := range customer.DefaultAddress.Company {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	return digits.String()
}

func preferString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
