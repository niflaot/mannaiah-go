package orders

import (
	"context"
	"errors"
	"fmt"
	"strings"

	contactsapplication "mannaiah/module/contacts/application"
	"mannaiah/module/exports/port"
	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
)

const pageSize = 500

var (
	// ErrNilOrderService is returned when order services are nil.
	ErrNilOrderService = errors.New("orders service must not be nil")
	// ErrNilContactService is returned when contact services are nil.
	ErrNilContactService = errors.New("contacts service must not be nil")
)

// Source adapts order application services to export source ports.
type Source struct {
	// orders defines order query dependencies.
	orders ordersapplication.Service
	// contacts defines contact lookup dependencies for customer emails.
	contacts contactsapplication.Service
}

var (
	// _ ensures Source satisfies export source ports.
	_ port.OrderSource = (*Source)(nil)
)

// NewSource creates order export source adapters.
func NewSource(orders ordersapplication.Service, contacts contactsapplication.Service) (*Source, error) {
	if orders == nil {
		return nil, ErrNilOrderService
	}
	if contacts == nil {
		return nil, ErrNilContactService
	}

	return &Source{orders: orders, contacts: contacts}, nil
}

// ListOrders returns all orders to export.
func (s *Source) ListOrders(ctx context.Context) ([]port.OrderRow, error) {
	rows := []port.OrderRow{}
	emailCache := map[string]string{}
	for page := 1; ; page++ {
		result, err := s.orders.List(ctx, ordersapplication.ListQuery{Page: page, Limit: pageSize})
		if err != nil {
			return nil, fmt.Errorf("list orders page %d: %w", page, err)
		}
		for _, order := range result.Data {
			email := emailCache[order.ContactID]
			if email == "" && strings.TrimSpace(order.ContactID) != "" {
				contact, contactErr := s.contacts.Get(ctx, order.ContactID)
				if contactErr == nil && contact != nil {
					email = contact.Email
					emailCache[order.ContactID] = email
				}
			}

			rows = append(rows, port.OrderRow{
				ID:            order.ID,
				Identifier:    order.Identifier,
				Realm:         order.Realm,
				ContactID:     order.ContactID,
				ContactEmail:  email,
				Address:       order.ShippingAddress.Address,
				Address2:      order.ShippingAddress.Address2,
				Phone:         order.ShippingAddress.Phone,
				CityName:      resolveCityName(order.ShippingAddress.CityCode),
				CityCode:      order.ShippingAddress.CityCode,
				Status:        string(order.CurrentStatus),
				Items:         mapItems(order.Items),
				PaymentMethod: order.PaymentMethod,
				Metadata:      cloneMetadata(order.Metadata),
				CreatedAt:     order.CreatedAt,
				UpdatedAt:     order.UpdatedAt,
			})
		}
		if result.TotalPages == 0 || page >= result.TotalPages || len(result.Data) == 0 {
			break
		}
	}

	return rows, nil
}

// mapItems maps order item aggregates to export rows.
func mapItems(items []ordersdomain.Item) []port.OrderItemRow {
	rows := make([]port.OrderItemRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, port.OrderItemRow{
			SKU:           item.SKU,
			AlternateName: item.AlternateName,
			Quantity:      item.Quantity,
			Value:         item.Value,
			ProductID:     item.ProductID,
		})
	}
	return rows
}

// resolveCityName resolves city display names from available order values.
func resolveCityName(cityCode string) string {
	return strings.TrimSpace(cityCode)
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
