package orders

import (
	"strings"

	contactdomain "mannaiah/module/contacts/domain"
	ordersdomain "mannaiah/module/orders/domain"
)

// newContactServiceMock creates initialized contact service mock values.
func newContactServiceMock() *contactServiceMock {
	return &contactServiceMock{
		contactsByID:     map[string]contactdomain.Contact{},
		contactIDByEmail: map[string]string{},
	}
}

// newOrdersServiceMock creates initialized order service mock values.
func newOrdersServiceMock() *ordersServiceMock {
	return &ordersServiceMock{
		orders: map[string]ordersdomain.Order{},
	}
}

// seedContact seeds contact rows into contact service mock values.
func seedContact(service *contactServiceMock, id string, email string) {
	entity := contactdomain.Contact{
		ID:        strings.TrimSpace(id),
		Email:     strings.ToLower(strings.TrimSpace(email)),
		FirstName: "Woo",
		LastName:  "One",
	}

	service.contactsByID[entity.ID] = entity
	service.contactIDByEmail[entity.Email] = entity.ID
}

// cloneMetadata clones metadata map values.
func cloneMetadata(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}

	return cloned
}
