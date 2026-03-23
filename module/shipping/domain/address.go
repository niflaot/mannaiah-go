package domain

import "strings"

// Address defines address values used for mark generation requests.
type Address struct {
	// Name defines receiver/sender display-name values.
	Name string `json:"name"`
	// ID defines receiver/sender identification values.
	ID string `json:"id"`
	// IDType defines identification-type values.
	IDType string `json:"idType"`
	// AddressLine defines full address-line values.
	AddressLine string `json:"addressLine"`
	// CityCode defines DANE city-code values.
	CityCode string `json:"cityCode"`
	// Phone defines phone-number values.
	Phone string `json:"phone"`
	// Email defines email-address values.
	Email string `json:"email"`
}

// Normalize normalizes address fields.
func (a Address) Normalize() Address {
	return Address{
		Name:        strings.TrimSpace(a.Name),
		ID:          strings.TrimSpace(a.ID),
		IDType:      strings.TrimSpace(a.IDType),
		AddressLine: strings.TrimSpace(a.AddressLine),
		CityCode:    strings.TrimSpace(a.CityCode),
		Phone:       strings.TrimSpace(a.Phone),
		Email:       strings.TrimSpace(a.Email),
	}
}
