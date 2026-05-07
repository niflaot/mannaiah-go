package contacts

import (
	"context"
	"errors"
	"fmt"

	contactsapplication "mannaiah/module/contacts/application"
	contactsport "mannaiah/module/contacts/port"
	"mannaiah/module/exports/port"
)

const pageSize = 500

var (
	// ErrNilService is returned when contact services are nil.
	ErrNilService = errors.New("contacts service must not be nil")
)

// Source adapts contact application services to export source ports.
type Source struct {
	// service defines contact query dependencies.
	service contactsapplication.Service
}

var (
	// _ ensures Source satisfies export source ports.
	_ port.ContactSource = (*Source)(nil)
)

// NewSource creates contact export source adapters.
func NewSource(service contactsapplication.Service) (*Source, error) {
	if service == nil {
		return nil, ErrNilService
	}

	return &Source{service: service}, nil
}

// ListContacts returns all contacts to export.
func (s *Source) ListContacts(ctx context.Context) ([]port.ContactRow, error) {
	rows := []port.ContactRow{}
	for page := 1; ; page++ {
		result, err := s.service.List(ctx, contactsport.ListQuery{
			Page:     page,
			Limit:    pageSize,
			OrderBy:  "createdAt",
			OrderDir: "asc",
		})
		if err != nil {
			return nil, fmt.Errorf("list contacts page %d: %w", page, err)
		}
		for _, contact := range result.Data {
			rows = append(rows, port.ContactRow{
				ID:             contact.ID,
				DocumentType:   string(contact.DocumentType),
				DocumentNumber: contact.DocumentNumber,
				LegalName:      contact.LegalName,
				FirstName:      contact.FirstName,
				LastName:       contact.LastName,
				Email:          contact.Email,
				Phone:          contact.Phone,
				Address:        contact.Address,
				AddressExtra:   contact.AddressExtra,
				CityCode:       contact.CityCode,
				Metadata:       cloneMetadata(contact.Metadata),
				CreatedAt:      contact.CreatedAt,
				UpdatedAt:      contact.UpdatedAt,
			})
		}
		if result.TotalPages == 0 || page >= result.TotalPages || len(result.Data) == 0 {
			break
		}
	}

	return rows, nil
}

// cloneMetadata copies metadata maps.
func cloneMetadata(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
