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
	// addCommentErr defines add-comment operation errors.
	addCommentErr error
	// orders stores order rows keyed by id.
	orders map[string]ordersdomain.Order
	// createCommands stores create command values.
	createCommands []ordersapplication.CreateCommand
	// updateCommands stores mutable update command values.
	updateCommands []ordersapplication.UpdateCommand
	// updateStatusCommands stores update-status command values.
	updateStatusCommands []ordersapplication.UpdateStatusCommand
	// addCommentCommands stores add-comment command values.
	addCommentCommands []ordersapplication.AddCommentCommand
	// ignoreWooSourceMutations reports whether Woo-origin status/comment mutations should be ignored.
	ignoreWooSourceMutations bool
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
	entity.AppliedCoupons = toDomainAppliedCoupons(command.AppliedCoupons)

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

// Update updates mutable order rows.
func (m *ordersServiceMock) Update(ctx context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error) {
	entity, ok := m.orders[strings.TrimSpace(id)]
	if !ok {
		return nil, ordersport.ErrNotFound
	}
	m.updateCommands = append(m.updateCommands, command)
	if m.ignoreWooSourceMutations && strings.HasPrefix(strings.ToLower(strings.TrimSpace(command.Source)), "woocommerce") && strings.EqualFold(strings.TrimSpace(entity.Realm), "woocommerce") {
		copied := entity
		return &copied, nil
	}

	if command.Items != nil {
		entity.Items = make([]ordersdomain.Item, 0, len(*command.Items))
		for _, item := range *command.Items {
			entity.Items = append(entity.Items, ordersdomain.Item{
				SKU:           strings.TrimSpace(item.SKU),
				AlternateName: strings.TrimSpace(item.AlternateName),
				Quantity:      item.Quantity,
				Value:         item.Value,
			})
		}
	}
	if command.ShippingAddress != nil {
		entity.HasCustomShippingAddress = true
		entity.ShippingAddress = ordersdomain.ShippingAddress{
			Address:  strings.TrimSpace(command.ShippingAddress.Address),
			Address2: strings.TrimSpace(command.ShippingAddress.Address2),
			Phone:    strings.TrimSpace(command.ShippingAddress.Phone),
			CityCode: strings.TrimSpace(command.ShippingAddress.CityCode),
		}
	}
	if command.ShippingCharges != nil {
		entity.ShippingCharges = make([]ordersdomain.ShippingCharge, 0, len(*command.ShippingCharges))
		for _, charge := range *command.ShippingCharges {
			entity.ShippingCharges = append(entity.ShippingCharges, ordersdomain.ShippingCharge{
				MethodID:    strings.TrimSpace(charge.MethodID),
				MethodTitle: strings.TrimSpace(charge.MethodTitle),
				Price:       charge.Price,
			})
		}
	}
	if command.AppliedCoupons != nil {
		entity.AppliedCoupons = toDomainAppliedCoupons(*command.AppliedCoupons)
	}

	m.orders[entity.ID] = entity
	copied := entity
	return &copied, nil
}

// toDomainAppliedCoupons maps application coupon commands to domain coupon values.
func toDomainAppliedCoupons(values []ordersapplication.AppliedCouponCommand) []ordersdomain.AppliedCoupon {
	if len(values) == 0 {
		return nil
	}

	rows := make([]ordersdomain.AppliedCoupon, 0, len(values))
	for _, value := range values {
		appliedAt := time.Now().UTC()
		if value.AppliedAt != nil && !value.AppliedAt.IsZero() {
			appliedAt = value.AppliedAt.UTC()
		}
		rows = append(rows, ordersdomain.AppliedCoupon{
			CouponID:       strings.TrimSpace(value.CouponID),
			Code:           strings.TrimSpace(value.Code),
			DiscountType:   strings.TrimSpace(value.DiscountType),
			DiscountAmount: value.DiscountAmount,
			AppliedAt:      appliedAt,
		})
	}

	return rows
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
	m.updateStatusCommands = append(m.updateStatusCommands, command)
	if m.ignoreWooSourceMutations && strings.HasPrefix(strings.ToLower(strings.TrimSpace(command.Source)), "woocommerce") && strings.EqualFold(strings.TrimSpace(entity.Realm), "woocommerce") {
		copied := entity
		return &copied, nil
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
		NoteOwner:   strings.TrimSpace(command.NoteOwner),
		Note:        strings.TrimSpace(command.Note),
		OccurredAt:  occurredAt,
	})

	m.orders[entity.ID] = entity

	copied := entity
	return &copied, nil
}

// AddComment appends order comment rows.
func (m *ordersServiceMock) AddComment(ctx context.Context, id string, command ordersapplication.AddCommentCommand) (*ordersdomain.Order, error) {
	if m.addCommentErr != nil {
		return nil, m.addCommentErr
	}

	entity, ok := m.orders[strings.TrimSpace(id)]
	if !ok {
		return nil, ordersport.ErrNotFound
	}
	m.addCommentCommands = append(m.addCommentCommands, command)
	if m.ignoreWooSourceMutations && strings.HasPrefix(strings.ToLower(strings.TrimSpace(command.Source)), "woocommerce") && strings.EqualFold(strings.TrimSpace(entity.Realm), "woocommerce") {
		copied := entity
		return &copied, nil
	}

	occurredAt := time.Now().UTC()
	if command.OccurredAt != nil && !command.OccurredAt.IsZero() {
		occurredAt = command.OccurredAt.UTC()
	}
	entity.Comments = append(entity.Comments, ordersdomain.Comment{
		Author:     strings.TrimSpace(command.Author),
		Comment:    strings.TrimSpace(command.Comment),
		Internal:   command.Internal,
		OccurredAt: occurredAt,
	})

	m.orders[entity.ID] = entity

	copied := entity
	return &copied, nil
}

// UpdateComment updates order comment rows.
func (m *ordersServiceMock) UpdateComment(ctx context.Context, id string, commentID string, command ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error) {
	entity, ok := m.orders[strings.TrimSpace(id)]
	if !ok {
		return nil, ordersport.ErrNotFound
	}
	if len(entity.Comments) == 0 {
		return nil, ordersport.ErrCommentNotFound
	}

	index := -1
	for valueIndex, value := range entity.Comments {
		if strings.TrimSpace(value.ID) == strings.TrimSpace(commentID) {
			index = valueIndex
			break
		}
	}
	if index < 0 {
		return nil, ordersport.ErrCommentNotFound
	}

	if command.Author != nil {
		entity.Comments[index].Author = strings.TrimSpace(*command.Author)
	}
	if command.Comment != nil {
		entity.Comments[index].Comment = strings.TrimSpace(*command.Comment)
	}
	if command.Internal != nil {
		entity.Comments[index].Internal = *command.Internal
	}

	m.orders[entity.ID] = entity
	copied := entity
	return &copied, nil
}

// DeleteComment deletes order comment rows.
func (m *ordersServiceMock) DeleteComment(ctx context.Context, id string, commentID string, command ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error) {
	entity, ok := m.orders[strings.TrimSpace(id)]
	if !ok {
		return nil, ordersport.ErrNotFound
	}
	if len(entity.Comments) == 0 {
		return nil, ordersport.ErrCommentNotFound
	}

	nextComments := make([]ordersdomain.Comment, 0, len(entity.Comments))
	deleted := false
	for _, value := range entity.Comments {
		if strings.TrimSpace(value.ID) == strings.TrimSpace(commentID) {
			deleted = true
			continue
		}
		nextComments = append(nextComments, value)
	}
	if !deleted {
		return nil, ordersport.ErrCommentNotFound
	}

	entity.Comments = nextComments
	m.orders[entity.ID] = entity
	copied := entity
	return &copied, nil
}
