package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	contactsdomain "mannaiah/module/contacts/domain"
	ordersdomain "mannaiah/module/orders/domain"
	shopifyport "mannaiah/module/shopify/port"
)

// BuildOrderContactSyncCommand maps one Shopify order into normalized contact values.
func BuildOrderContactSyncCommand(order shopifyport.ShopifyOrder) (shopifyport.ContactSyncCommand, error) {
	email := preferString(order.ContactEmail, customerEmail(order.Customer))
	if email == "" {
		return shopifyport.ContactSyncCommand{}, ErrOrderContactEmailRequired
	}

	attributes := order.NoteAttributes
	if len(attributes) == 0 && order.Customer != nil {
		attributes = order.Customer.NoteAttributes
	}
	documentType, documentNumber := extractDocument(attributes)
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
		CityCode:       addressCity(address),
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
	command := shopifyport.OrderSyncCommand{
		ShopifyID:         strings.TrimSpace(order.ID),
		Identifier:        resolveOrderIdentifier(order),
		Realm:             resolveRealm(realm),
		ContactID:         strings.TrimSpace(contactID),
		Items:             buildOrderItems(order.LineItems),
		Status:            mapOrderStatus(order),
		StatusDescription: buildStatusDescription(order),
		ShippingAddress:   buildShippingAddress(order),
		ShippingCharges:   buildShippingCharges(order.ShippingLines),
		AppliedCoupons:    buildAppliedCoupons(order.DiscountCodes, order.CreatedAt),
		PaymentMethod:     strings.Join(order.PaymentGatewayNames, ", "),
		Metadata:          buildOrderMetadata(order),
		Source:            resolveTrigger(trigger),
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
		return ordersdomain.StatusCompleted
	case "voided", "refunded":
		return ordersdomain.StatusCancelled
	case "pending", "authorized", "partially_paid":
		return ordersdomain.StatusPending
	}

	return ordersdomain.StatusCreated
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
			SKU:           strings.TrimSpace(value.SKU),
			AlternateName: buildAlternateName(value.Title, value.VariantTitle),
			Quantity:      value.Quantity,
			Value:         parseMoney(value.Price),
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
		CityCode: addressCity(address),
	}
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
	return metadata
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

func buildAlternateName(title string, variantTitle string) string {
	trimmedTitle := strings.TrimSpace(title)
	trimmedVariant := strings.TrimSpace(variantTitle)
	if trimmedTitle == "" {
		return trimmedVariant
	}
	if trimmedVariant == "" || strings.EqualFold(trimmedVariant, "default title") || strings.EqualFold(trimmedVariant, trimmedTitle) {
		return trimmedTitle
	}

	return trimmedTitle + " - " + trimmedVariant
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

func extractDocument(attributes []shopifyport.ShopifyNoteAttribute) (contactsdomain.DocumentType, string) {
	values := map[string]string{}
	for _, attribute := range attributes {
		key := normalizeAttributeKey(attribute.Name)
		if key == "" {
			continue
		}
		values[key] = strings.TrimSpace(attribute.Value)
	}

	documentType := normalizeDocumentType(preferString(values["document_type"], values["documenttype"], values["doc_type"]))
	documentNumber := preferString(values["document_number"], values["documentnumber"], values["doc_number"], values["document"])
	return documentType, strings.TrimSpace(documentNumber)
}

func normalizeDocumentType(value string) contactsdomain.DocumentType {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case string(contactsdomain.DocumentTypeCC):
		return contactsdomain.DocumentTypeCC
	case string(contactsdomain.DocumentTypeCE):
		return contactsdomain.DocumentTypeCE
	case string(contactsdomain.DocumentTypeTI):
		return contactsdomain.DocumentTypeTI
	case string(contactsdomain.DocumentTypePAS):
		return contactsdomain.DocumentTypePAS
	case string(contactsdomain.DocumentTypeNIT):
		return contactsdomain.DocumentTypeNIT
	case string(contactsdomain.DocumentTypeOther), "OTRO":
		return contactsdomain.DocumentTypeOther
	default:
		return ""
	}
}

func normalizeAttributeKey(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	trimmed = strings.ReplaceAll(trimmed, "-", "_")
	trimmed = strings.ReplaceAll(trimmed, " ", "_")
	return trimmed
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
