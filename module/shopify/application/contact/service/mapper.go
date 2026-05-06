package service

import (
	"strings"

	contactsdomain "mannaiah/module/contacts/domain"
	shopifyport "mannaiah/module/shopify/port"
)

// BuildContactSyncCommand maps one Shopify customer into normalized mainstream contact values.
func BuildContactSyncCommand(customer shopifyport.ShopifyCustomer) shopifyport.ContactSyncCommand {
	documentType, documentNumber := extractDocument(customer.NoteAttributes)
	firstName := strings.TrimSpace(customer.FirstName)
	lastName := strings.TrimSpace(customer.LastName)
	legalName := ""
	if customer.DefaultAddress != nil {
		if firstName == "" {
			firstName = strings.TrimSpace(customer.DefaultAddress.FirstName)
		}
		if lastName == "" {
			lastName = strings.TrimSpace(customer.DefaultAddress.LastName)
		}
	}
	if documentType == contactsdomain.DocumentTypeNIT && customer.DefaultAddress != nil {
		legalName = strings.TrimSpace(customer.DefaultAddress.Company)
		firstName = ""
		lastName = ""
	}
	if legalName == "" && (firstName == "" || lastName == "") {
		firstName = preferString(firstName, "Shopify")
		lastName = preferString(lastName, "Customer")
	}

	command := shopifyport.ContactSyncCommand{
		ShopifyID:      strings.TrimSpace(customer.ID),
		Email:          strings.TrimSpace(customer.Email),
		DocumentType:   documentType,
		DocumentNumber: documentNumber,
		LegalName:      legalName,
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
