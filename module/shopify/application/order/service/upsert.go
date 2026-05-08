package service

import (
	"context"
	"errors"
	"strings"
	"time"

	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	shopifyport "mannaiah/module/shopify/port"
)

var (
	// ErrNilOrdersService is returned when a nil orders service is provided.
	ErrNilOrdersService = errors.New("shopify orders application service must not be nil")
	// ErrNilSyncLinks is returned when a nil sync-link repository is provided.
	ErrNilSyncLinks = errors.New("shopify order sync link repository must not be nil")
)

const (
	syncMutationSource = "shopify_sync"
	statusAuthor       = syncMutationSource
)

// OrderUpserter defines mainstream order upsert behavior for Shopify sync flows.
type OrderUpserter struct {
	// service defines mainstream order application dependencies.
	service ordersapplication.Service
	// links defines Shopify sync-link persistence dependencies.
	links shopifyport.SyncLinkRepository
}

var (
	// _ ensures OrderUpserter satisfies Shopify order target contracts.
	_ shopifyport.OrderSyncTarget = (*OrderUpserter)(nil)
)

// NewUpserter creates mainstream order upsert adapters for Shopify sync flows.
func NewUpserter(service ordersapplication.Service, links shopifyport.SyncLinkRepository) (*OrderUpserter, error) {
	if service == nil {
		return nil, ErrNilOrdersService
	}
	if links == nil {
		return nil, ErrNilSyncLinks
	}

	return &OrderUpserter{service: service, links: links}, nil
}

// UpsertOrder creates or updates one mainstream order from Shopify values.
func (u *OrderUpserter) UpsertOrder(ctx context.Context, command shopifyport.OrderSyncCommand) (*ordersdomain.Order, error) {
	existing, err := u.findExisting(ctx, command)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		created, createErr := u.service.Create(ctx, buildOrderCreateCommand(command))
		if createErr != nil {
			if !errors.Is(createErr, ordersport.ErrDuplicateIdentifier) {
				return nil, createErr
			}
			existing, err = u.findExisting(ctx, command)
			if err != nil || existing == nil {
				return nil, createErr
			}
		} else {
			if linkErr := u.upsertLink(ctx, command, created.ID, created.CurrentStatus); linkErr != nil {
				return nil, linkErr
			}
			if linkErr := u.upsertProductLinks(ctx, command); linkErr != nil {
				return nil, linkErr
			}
			return created, nil
		}
	}

	updated, err := u.service.Update(ctx, existing.ID, buildOrderUpdateCommand(command))
	if err != nil {
		return nil, err
	}
	if updated.CurrentStatus != command.Status && strings.TrimSpace(string(command.Status)) != "" {
		updated, err = u.service.UpdateStatus(ctx, updated.ID, ordersapplication.UpdateStatusCommand{
			Status:      command.Status,
			Author:      statusAuthor,
			Description: strings.TrimSpace(command.StatusDescription),
			Source:      syncMutationSource,
		})
		if err != nil {
			return nil, err
		}
	}
	if linkErr := u.upsertLink(ctx, command, updated.ID, updated.CurrentStatus); linkErr != nil {
		return nil, linkErr
	}
	if linkErr := u.upsertProductLinks(ctx, command); linkErr != nil {
		return nil, linkErr
	}

	return updated, nil
}

func (u *OrderUpserter) findExisting(ctx context.Context, command shopifyport.OrderSyncCommand) (*ordersdomain.Order, error) {
	if linked, err := u.findLinkedOrder(ctx, command); err != nil || linked != nil {
		return linked, err
	}
	if exact, err := u.findByIdentifier(ctx, command.Realm, command.Identifier); err != nil || exact != nil {
		return exact, err
	}
	return u.findByIdentifier(ctx, "", command.Identifier)
}

func (u *OrderUpserter) findLinkedOrder(ctx context.Context, command shopifyport.OrderSyncCommand) (*ordersdomain.Order, error) {
	shopifyID := strings.TrimSpace(command.ShopifyID)
	if shopifyID == "" {
		return nil, nil
	}
	link, err := u.links.GetLinkByShopifyID(ctx, shopifyport.SyncKindOrder, command.ShopDomain, shopifyID)
	if err != nil || link == nil || strings.TrimSpace(link.MannaiahID) == "" {
		return nil, err
	}
	entity, err := u.service.Get(ctx, link.MannaiahID)
	if errors.Is(err, ordersport.ErrNotFound) {
		return nil, nil
	}
	return entity, err
}

func (u *OrderUpserter) findByIdentifier(ctx context.Context, realm string, identifier string) (*ordersdomain.Order, error) {
	result, err := u.service.List(ctx, ordersapplication.ListQuery{
		Page:       1,
		Limit:      1,
		Realm:      strings.TrimSpace(realm),
		Identifier: strings.TrimSpace(identifier),
	})
	if err != nil || result == nil || len(result.Data) == 0 {
		return nil, err
	}
	entity := result.Data[0]
	return &entity, nil
}

func (u *OrderUpserter) upsertLink(ctx context.Context, command shopifyport.OrderSyncCommand, orderID string, currentStatus ordersdomain.Status) error {
	lastSyncedAt := time.Now().UTC()
	_, err := u.links.UpsertLink(ctx, shopifyport.UpsertSyncLinkInput{
		Kind:            shopifyport.SyncKindOrder,
		ShopDomain:      strings.TrimSpace(command.ShopDomain),
		ShopifyID:       strings.TrimSpace(command.ShopifyID),
		MannaiahID:      strings.TrimSpace(orderID),
		LastKnownStatus: strings.TrimSpace(string(currentStatus)),
		LastSyncedAt:    &lastSyncedAt,
	})
	return err
}

func (u *OrderUpserter) upsertProductLinks(ctx context.Context, command shopifyport.OrderSyncCommand) error {
	for _, item := range command.Items {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" {
			continue
		}
		lastSyncedAt := time.Now().UTC()
		if strings.TrimSpace(item.ShopifyProductID) != "" {
			if _, err := u.links.UpsertLink(ctx, shopifyport.UpsertSyncLinkInput{
				Kind:         shopifyport.SyncKindProduct,
				ShopDomain:   strings.TrimSpace(command.ShopDomain),
				ShopifyID:    strings.TrimSpace(item.ShopifyProductID),
				MannaiahID:   productID,
				LastSyncedAt: &lastSyncedAt,
			}); err != nil {
				return err
			}
		}
		if strings.TrimSpace(item.ShopifyVariantID) != "" {
			if _, err := u.links.UpsertLink(ctx, shopifyport.UpsertSyncLinkInput{
				Kind:         shopifyport.SyncKindVariant,
				ShopDomain:   strings.TrimSpace(command.ShopDomain),
				ShopifyID:    strings.TrimSpace(item.ShopifyVariantID),
				MannaiahID:   productID,
				LastSyncedAt: &lastSyncedAt,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildOrderCreateCommand(command shopifyport.OrderSyncCommand) ordersapplication.CreateCommand {
	return ordersapplication.CreateCommand{
		Identifier:      strings.TrimSpace(command.Identifier),
		Realm:           strings.TrimSpace(command.Realm),
		ContactID:       strings.TrimSpace(command.ContactID),
		Items:           buildCreateItems(command.Items),
		InitialStatus:   pointerStatus(command.Status),
		Author:          statusAuthor,
		Description:     strings.TrimSpace(command.StatusDescription),
		ShippingAddress: buildCreateShippingAddress(command.ShippingAddress),
		ShippingCharges: buildCreateShippingCharges(command.ShippingCharges),
		PaymentMethod:   strings.TrimSpace(command.PaymentMethod),
		AppliedCoupons:  buildCreateCoupons(command.AppliedCoupons),
		Metadata:        cloneMetadata(command.Metadata),
		CreatedAt:       command.CreatedAt,
		Source:          syncMutationSource,
	}
}

func buildOrderUpdateCommand(command shopifyport.OrderSyncCommand) ordersapplication.UpdateCommand {
	items := buildCreateItems(command.Items)
	shippingCharges := buildCreateShippingCharges(command.ShippingCharges)
	appliedCoupons := buildCreateCoupons(command.AppliedCoupons)
	return ordersapplication.UpdateCommand{
		Items:           &items,
		ShippingAddress: buildCreateShippingAddress(command.ShippingAddress),
		ShippingCharges: &shippingCharges,
		AppliedCoupons:  &appliedCoupons,
		Source:          syncMutationSource,
	}
}

func buildCreateItems(values []shopifyport.OrderSyncItemCommand) []ordersapplication.CreateItemCommand {
	items := make([]ordersapplication.CreateItemCommand, 0, len(values))
	for _, value := range values {
		items = append(items, ordersapplication.CreateItemCommand{
			SKU:           strings.TrimSpace(value.SKU),
			AlternateName: strings.TrimSpace(value.AlternateName),
			ProductID:     strings.TrimSpace(value.ProductID),
			Quantity:      value.Quantity,
			Value:         value.Value,
		})
	}
	return items
}

func buildCreateShippingAddress(value *shopifyport.OrderSyncShippingAddressCommand) *ordersapplication.ShippingAddressCommand {
	if value == nil {
		return nil
	}
	return &ordersapplication.ShippingAddressCommand{
		Address:  strings.TrimSpace(value.Address),
		Address2: strings.TrimSpace(value.Address2),
		Phone:    strings.TrimSpace(value.Phone),
		CityCode: strings.TrimSpace(value.CityCode),
	}
}

func buildCreateShippingCharges(values []shopifyport.OrderSyncShippingChargeCommand) []ordersapplication.ShippingChargeCommand {
	charges := make([]ordersapplication.ShippingChargeCommand, 0, len(values))
	for _, value := range values {
		charges = append(charges, ordersapplication.ShippingChargeCommand{
			MethodID:    strings.TrimSpace(value.MethodID),
			MethodTitle: strings.TrimSpace(value.MethodTitle),
			Price:       value.Price,
		})
	}
	return charges
}

func buildCreateCoupons(values []shopifyport.OrderSyncAppliedCouponCommand) []ordersapplication.AppliedCouponCommand {
	coupons := make([]ordersapplication.AppliedCouponCommand, 0, len(values))
	for _, value := range values {
		coupons = append(coupons, ordersapplication.AppliedCouponCommand{
			Code:           strings.TrimSpace(value.Code),
			DiscountType:   strings.TrimSpace(value.DiscountType),
			DiscountAmount: value.DiscountAmount,
			AppliedAt:      value.AppliedAt,
		})
	}
	return coupons
}

func cloneMetadata(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = strings.TrimSpace(value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func pointerStatus(value ordersdomain.Status) *ordersdomain.Status {
	resolved := ordersdomain.Status(strings.TrimSpace(string(value)))
	if resolved == "" {
		return nil
	}
	return &resolved
}
