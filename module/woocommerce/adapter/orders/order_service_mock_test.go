package orders

import (
	"context"
	"strings"
	"time"

	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

// ordersServiceMock defines order service behavior for order upserter tests.
type ordersServiceMock struct {
	// createErr defines create-operation errors.
	createErr error
	// listErr defines list-operation errors.
	listErr error
	// updateStatusErr defines update-status operation errors.
	updateStatusErr error
	// orders stores order rows keyed by id.
	orders map[string]ordersdomain.Order
	// createCommands stores create command values.
	createCommands []ordersapplication.CreateCommand
	// updateStatusCommands stores update-status command values.
	updateStatusCommands []ordersapplication.UpdateStatusCommand
}

// Create stores created order rows.
func (m *ordersServiceMock) Create(ctx context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	for _, entity := range m.orders {
		if entity.Realm == strings.TrimSpace(command.Realm) && entity.Identifier == strings.TrimSpace(command.Identifier) {
			return nil, ordersport.ErrDuplicateIdentifier
		}
	}

	identifier := "order-" + strings.TrimSpace(command.Identifier)
	status := ordersdomain.StatusCreated
	if command.InitialStatus != nil {
		status = *command.InitialStatus
	}
	occurredAt := time.Now().UTC()
	if command.CreatedAt != nil && !command.CreatedAt.IsZero() {
		occurredAt = command.CreatedAt.UTC()
	}
	entity := ordersdomain.Order{
		ID:                       identifier,
		Identifier:               strings.TrimSpace(command.Identifier),
		Realm:                    strings.TrimSpace(command.Realm),
		ContactID:                strings.TrimSpace(command.ContactID),
		CurrentStatus:            status,
		StatusHistory:            []ordersdomain.StatusEntry{{Status: status, Author: strings.TrimSpace(command.Author), Description: strings.TrimSpace(command.Description), OccurredAt: occurredAt}},
		Metadata:                 cloneMetadata(command.Metadata),
		HasCustomShippingAddress: command.ShippingAddress != nil,
		CreatedAt:                occurredAt,
		Items:                    make([]ordersdomain.Item, 0, len(command.Items)),
	}
	for _, item := range command.Items {
		entity.Items = append(entity.Items, ordersdomain.Item{
			SKU:           strings.TrimSpace(item.SKU),
			AlternateName: strings.TrimSpace(item.AlternateName),
			Quantity:      item.Quantity,
			Value:         item.Value,
		})
	}
	if command.ShippingAddress != nil {
		entity.ShippingAddress = ordersdomain.ShippingAddress{
			Address:  strings.TrimSpace(command.ShippingAddress.Address),
			Address2: strings.TrimSpace(command.ShippingAddress.Address2),
			Phone:    strings.TrimSpace(command.ShippingAddress.Phone),
			CityCode: strings.TrimSpace(command.ShippingAddress.CityCode),
		}
	}
	entity.ShippingCharges = make([]ordersdomain.ShippingCharge, 0, len(command.ShippingCharges))
	for _, charge := range command.ShippingCharges {
		entity.ShippingCharges = append(entity.ShippingCharges, ordersdomain.ShippingCharge{
			MethodID:    strings.TrimSpace(charge.MethodID),
			MethodTitle: strings.TrimSpace(charge.MethodTitle),
			Price:       charge.Price,
		})
	}

	m.createCommands = append(m.createCommands, command)
	m.orders[identifier] = entity

	copied := entity
	return &copied, nil
}

// Get resolves orders by id.
func (m *ordersServiceMock) Get(ctx context.Context, id string) (*ordersdomain.Order, error) {
	entity, ok := m.orders[strings.TrimSpace(id)]
	if !ok {
		return nil, ordersport.ErrNotFound
	}

	copied := entity
	return &copied, nil
}

// List resolves orders by realm and identifier filters.
func (m *ordersServiceMock) List(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}

	rows := make([]ordersdomain.Order, 0)
	for _, entity := range m.orders {
		if query.Realm != "" && strings.TrimSpace(entity.Realm) != strings.TrimSpace(query.Realm) {
			continue
		}
		if query.Identifier != "" && strings.TrimSpace(entity.Identifier) != strings.TrimSpace(query.Identifier) {
			continue
		}

		rows = append(rows, entity)
	}

	return &ordersapplication.ListResult{
		Data:  rows,
		Total: int64(len(rows)),
		Page:  1,
		Limit: 1,
	}, nil
}

// UpdateStatus appends status history rows.
func (m *ordersServiceMock) UpdateStatus(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
	if m.updateStatusErr != nil {
		return nil, m.updateStatusErr
	}

	entity, ok := m.orders[strings.TrimSpace(id)]
	if !ok {
		return nil, ordersport.ErrNotFound
	}

	occurredAt := time.Now().UTC()
	if command.OccurredAt != nil && !command.OccurredAt.IsZero() {
		occurredAt = command.OccurredAt.UTC()
	}
	entity.CurrentStatus = command.Status
	entity.StatusHistory = append(entity.StatusHistory, ordersdomain.StatusEntry{
		Status:      command.Status,
		Author:      strings.TrimSpace(command.Author),
		Description: strings.TrimSpace(command.Description),
		OccurredAt:  occurredAt,
	})

	m.updateStatusCommands = append(m.updateStatusCommands, command)
	m.orders[entity.ID] = entity

	copied := entity
	return &copied, nil
}
