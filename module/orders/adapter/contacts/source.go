package contacts

import (
	"context"
	"errors"
	"strings"

	contactsapplication "mannaiah/module/contacts/application"
	contactsport "mannaiah/module/contacts/port"
	ordersport "mannaiah/module/orders/port"
)

var (
	// ErrNilService is returned when contact service dependencies are nil.
	ErrNilService = errors.New("orders contacts source service must not be nil")
)

// Source defines contact-backed customer lookup behavior.
type Source struct {
	// service defines contacts application service dependencies.
	service contactsapplication.Service
}

var (
	// _ ensures Source satisfies customer-source contracts.
	_ ordersport.CustomerSource = (*Source)(nil)
)

// NewSource creates customer source adapters over contacts application services.
func NewSource(service contactsapplication.Service) (*Source, error) {
	if service == nil {
		return nil, ErrNilService
	}

	return &Source{service: service}, nil
}

// GetByID resolves customer values by identifiers.
func (s *Source) GetByID(ctx context.Context, id string) (*ordersport.Customer, error) {
	entity, err := s.service.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, contactsport.ErrNotFound) {
			return nil, ordersport.ErrCustomerNotFound
		}
		return nil, err
	}

	return &ordersport.Customer{
		ID:           entity.ID,
		Address:      entity.Address,
		AddressExtra: entity.AddressExtra,
		Phone:        entity.Phone,
		CityCode:     entity.CityCode,
	}, nil
}
