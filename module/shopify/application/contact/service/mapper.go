package service

import (
	"strings"
	"unicode"

	contactsdomain "mannaiah/module/contacts/domain"
	shopifyport "mannaiah/module/shopify/port"
)

// BuildContactSyncCommand maps one Shopify customer into normalized mainstream contact values.
func BuildContactSyncCommand(customer shopifyport.ShopifyCustomer) shopifyport.ContactSyncCommand {
	// The checkout plugin relabels the Company field as the document ID input.
	// Strip formatting characters (dots, dashes, spaces) so "1.234.567" becomes "1234567".
	// All Shopify e-commerce customers present a CC (Cédula de Ciudadanía).
	documentNumber := extractCompanyDocumentNumber(customer.DefaultAddress)
	var documentType contactsdomain.DocumentType
	if documentNumber != "" {
		documentType = contactsdomain.DocumentTypeCC
	}

	firstName := strings.TrimSpace(customer.FirstName)
	lastName := strings.TrimSpace(customer.LastName)
	if customer.DefaultAddress != nil {
		if firstName == "" {
			firstName = strings.TrimSpace(customer.DefaultAddress.FirstName)
		}
		if lastName == "" {
			lastName = strings.TrimSpace(customer.DefaultAddress.LastName)
		}
	}
	if firstName == "" || lastName == "" {
		firstName = preferString(firstName, "Shopify")
		lastName = preferString(lastName, "Customer")
	}

	command := shopifyport.ContactSyncCommand{
		ShopDomain:     strings.TrimSpace(customer.ShopDomain),
		ShopifyID:      strings.TrimSpace(customer.ID),
		Email:          strings.TrimSpace(customer.Email),
		DocumentType:   documentType,
		DocumentNumber: documentNumber,
		FirstName:      firstName,
		LastName:       lastName,
		Phone:          strings.TrimSpace(customer.Phone),
		Metadata:       buildCustomerMetadata(customer),
	}
	if !customer.CreatedAt.IsZero() {
		createdAt := customer.CreatedAt.UTC()
		command.CreatedAt = &createdAt
	}
	if customer.DefaultAddress != nil {
		command.Address = strings.TrimSpace(customer.DefaultAddress.Address1)
		command.AddressExtra = strings.TrimSpace(customer.DefaultAddress.Address2)
		command.CityCode = strings.TrimSpace(customer.DefaultAddress.City)
		command.Phone = preferString(command.Phone, customer.DefaultAddress.Phone)
	}

	return command
}

// extractCompanyDocumentNumber reads the Company field from the default address and
// returns only its digit characters. Returns empty string when no digits are found.
func extractCompanyDocumentNumber(address *shopifyport.ShopifyAddress) string {
	if address == nil {
		return ""
	}
	var digits strings.Builder
	for _, r := range address.Company {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	return digits.String()
}

func buildCustomerMetadata(customer shopifyport.ShopifyCustomer) map[string]string {
	metadata := map[string]string{}
	if customer.ID != "" {
		metadata["shopify_customer_id"] = strings.TrimSpace(customer.ID)
	}
	if customer.Tags != "" {
		metadata["shopify_customer_tags"] = strings.TrimSpace(customer.Tags)
	}

	return metadata
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
