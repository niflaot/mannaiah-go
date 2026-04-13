package orders

import (
	"context"
	"errors"
	"fmt"
	"strings"

	contactsapplication "mannaiah/module/contacts/application"
	contactsport "mannaiah/module/contacts/port"
	couponservice "mannaiah/module/coupons/application/coupon/service"
	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	contactsadapter "mannaiah/module/woocommerce/adapter/contacts"
	"mannaiah/module/woocommerce/port"
)

const (
	// defaultRealm defines default order realm values for WooCommerce syncs.
	defaultRealm = "woocommerce"
	// syncStatusAuthor defines status-entry author values used by WooCommerce sync updates.
	syncStatusAuthor = "woocommerce_sync"
	// syncStatusDescription defines default status-entry description values used by WooCommerce sync updates.
	syncStatusDescription = "WooCommerce Sync"
	// syncCommentAuthor defines default order-comment author values for WooCommerce synchronized comments.
	syncCommentAuthor = "system"
)

var (
	// ErrNilOrderService is returned when order service dependencies are nil.
	ErrNilOrderService = errors.New("woocommerce orders upserter order service must not be nil")
	// ErrNilContactService is returned when contact service dependencies are nil.
	ErrNilContactService = errors.New("woocommerce orders upserter contact service must not be nil")
	// ErrContactNotFound is returned when contact rows cannot be resolved after contact upsert behavior.
	ErrContactNotFound = errors.New("woocommerce sync contact not found by email")
)

// CouponUsageSyncService defines coupon-usage backfill behavior for synchronized orders.
type CouponUsageSyncService interface {
	// SyncUsageByCode backfills coupon usage matched by coupon code.
	SyncUsageByCode(ctx context.Context, cmd couponservice.SyncUsageByCodeCommand) error
}

// Upserter defines order upsert behavior backed by orders and contacts application services.
type Upserter struct {
	// orderService defines order application service dependencies.
	orderService ordersapplication.Service
	// contactService defines contact application service dependencies.
	contactService contactsapplication.Service
	// contactUpserter defines contact upsert behavior for fallback contact synchronization.
	contactUpserter port.ContactSyncTarget
	// couponUsageSyncService defines optional coupon-usage backfill behavior.
	couponUsageSyncService CouponUsageSyncService
}

var (
	// _ ensures Upserter satisfies order-sync target contracts.
	_ port.OrderSyncTarget = (*Upserter)(nil)
)

// NewUpserter creates WooCommerce order upsert adapters over order and contact services.
func NewUpserter(orderService ordersapplication.Service, contactService contactsapplication.Service) (*Upserter, error) {
	if orderService == nil {
		return nil, ErrNilOrderService
	}
	if contactService == nil {
		return nil, ErrNilContactService
	}

	contactUpserter, err := contactsadapter.NewUpserter(contactService)
	if err != nil {
		return nil, err
	}

	return &Upserter{
		orderService:    orderService,
		contactService:  contactService,
		contactUpserter: contactUpserter,
	}, nil
}

// SetCouponUsageSyncService configures optional coupon-usage synchronization behavior.
func (u *Upserter) SetCouponUsageSyncService(service CouponUsageSyncService) {
	if u == nil {
		return
	}

	u.couponUsageSyncService = service
}

// UpsertByIdentifier creates or updates orders by realm and identifier.
func (u *Upserter) UpsertByIdentifier(ctx context.Context, command port.OrderSyncCommand) (port.UpsertOutcome, error) {
	if _, err := u.contactUpserter.UpsertByEmail(ctx, command.Contact); err != nil {
		return "", fmt.Errorf("sync contact for woocommerce order: %w", err)
	}

	contactID, err := u.resolveContactIDByEmail(ctx, command.Contact.Email)
	if err != nil {
		return "", err
	}

	realm := normalizeRealm(command.Realm)
	status := mapOrderStatus(command.Status)

	existing, err := u.findExistingByIdentifier(ctx, realm, command.Identifier)
	if err != nil {
		return "", err
	}
	if existing == nil {
		return u.createOrder(ctx, command, contactID, realm, status)
	}

	return u.updateExisting(ctx, *existing, command, status)
}

// createOrder creates new orders and appends synchronized comments.
func (u *Upserter) createOrder(
	ctx context.Context,
	command port.OrderSyncCommand,
	contactID string,
	realm string,
	status ordersdomain.Status,
) (port.UpsertOutcome, error) {
	created, err := u.orderService.Create(ctx, toCreateCommand(command, contactID, realm, status))
	if err != nil {
		if !errors.Is(err, ordersport.ErrDuplicateIdentifier) {
			return "", fmt.Errorf("create order for woocommerce sync: %w", err)
		}

		existing, findErr := u.findExistingByIdentifier(ctx, realm, command.Identifier)
		if findErr != nil {
			return "", findErr
		}
		if existing == nil {
			return "", fmt.Errorf("create order for woocommerce sync: %w", err)
		}

		return u.updateExisting(ctx, *existing, command, status)
	}

	_, _, appendErr := u.appendComments(ctx, *created, command.Comments)
	if appendErr != nil {
		return "", appendErr
	}
	if err := u.syncCouponUsages(ctx, created.ID, command); err != nil {
		return "", err
	}

	return port.UpsertOutcomeCreated, nil
}

// updateExisting appends status and comment updates for existing order rows.
func (u *Upserter) updateExisting(
	ctx context.Context,
	existing ordersdomain.Order,
	command port.OrderSyncCommand,
	status ordersdomain.Status,
) (port.UpsertOutcome, error) {
	current := existing
	updated := false

	mutatedOrder, mutableUpdated, err := u.syncMutableState(ctx, current, command)
	if err != nil {
		return "", err
	}
	if mutableUpdated {
		current = mutatedOrder
		updated = true
	}

	if current.CurrentStatus != status {
		next, err := u.orderService.UpdateStatus(ctx, current.ID, ordersapplication.UpdateStatusCommand{
			Status:      status,
			Author:      syncStatusAuthor,
			Description: syncStatusDescription,
			OccurredAt:  resolveStatusOccurredAt(current, command.CreatedAt),
			Source:      syncStatusAuthor,
		})
		if err != nil {
			return "", fmt.Errorf("update order status for woocommerce sync: %w", err)
		}
		if hasStatusMutation(current, *next) {
			updated = true
		}
		current = *next
	}

	commentUpdated, next, err := u.appendComments(ctx, current, command.Comments)
	if err != nil {
		return "", err
	}
	if commentUpdated {
		current = next
		updated = true
	}
	if err := u.syncCouponUsages(ctx, current.ID, command); err != nil {
		return "", err
	}

	if updated {
		return port.UpsertOutcomeUpdated, nil
	}

	return port.UpsertOutcomeUnchanged, nil
}

// syncMutableState refreshes mutable order fields for synchronized WooCommerce orders.
func (u *Upserter) syncMutableState(ctx context.Context, existing ordersdomain.Order, command port.OrderSyncCommand) (ordersdomain.Order, bool, error) {
	updateCommand := toUpdateCommand(command, existing)
	next, err := u.orderService.Update(ctx, existing.ID, updateCommand)
	if err != nil {
		return ordersdomain.Order{}, false, fmt.Errorf("update order for woocommerce sync: %w", err)
	}
	if next == nil {
		return existing, false, nil
	}

	return *next, hasMutableOrderStateChanges(existing, *next), nil
}

// syncCouponUsages records WooCommerce coupon redemptions against matching local coupons.
func (u *Upserter) syncCouponUsages(ctx context.Context, orderID string, command port.OrderSyncCommand) error {
	if u == nil || u.couponUsageSyncService == nil || strings.TrimSpace(orderID) == "" || len(command.AppliedCoupons) == 0 {
		return nil
	}

	for _, appliedCoupon := range command.AppliedCoupons {
		code := strings.TrimSpace(appliedCoupon.Code)
		if code == "" {
			continue
		}

		if err := u.couponUsageSyncService.SyncUsageByCode(ctx, couponservice.SyncUsageByCodeCommand{
			Code:    code,
			OrderID: strings.TrimSpace(orderID),
			Email:   strings.TrimSpace(command.Contact.Email),
			UsedAt:  command.CreatedAt,
		}); err != nil {
			return fmt.Errorf("sync coupon usage for order %s coupon %s: %w", strings.TrimSpace(orderID), code, err)
		}
	}

	return nil
}

// resolveContactIDByEmail resolves contact identifiers by normalized email values.
func (u *Upserter) resolveContactIDByEmail(ctx context.Context, email string) (string, error) {
	result, err := u.contactService.List(ctx, contactsport.ListQuery{
		Page:  1,
		Limit: 1,
		Email: strings.TrimSpace(email),
	})
	if err != nil {
		return "", fmt.Errorf("resolve sync contact by email: %w", err)
	}
	if len(result.Data) == 0 {
		return "", ErrContactNotFound
	}

	return strings.TrimSpace(result.Data[0].ID), nil
}

// findExistingByIdentifier resolves optional existing order rows by realm and identifier.
func (u *Upserter) findExistingByIdentifier(ctx context.Context, realm string, identifier string) (*ordersdomain.Order, error) {
	result, err := u.orderService.List(ctx, ordersapplication.ListQuery{
		Page:       1,
		Limit:      1,
		Realm:      strings.TrimSpace(realm),
		Identifier: strings.TrimSpace(identifier),
	})
	if err != nil {
		return nil, fmt.Errorf("resolve existing order by identifier: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, nil
	}

	entity := result.Data[0]
	return &entity, nil
}
