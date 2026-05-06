package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	contactsapplication "mannaiah/module/contacts/application"
	contactsdomain "mannaiah/module/contacts/domain"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	shopifyport "mannaiah/module/shopify/port"

	"go.uber.org/zap"
)

var (
	// ErrNilDestination is returned when a nil Shopify destination is provided.
	ErrNilDestination = errors.New("shopify order destination must not be nil")
	// ErrNilLinkRepository is returned when a nil sync-link repository is provided.
	ErrNilLinkRepository = errors.New("shopify link repository must not be nil")
	// ErrCreatedOrderIDRequired is returned when Shopify creates an order without an id.
	ErrCreatedOrderIDRequired = errors.New("created shopify order id is required")
)

// OrderEventHandler defines mainstream order integration event handling behavior.
type OrderEventHandler interface {
	// HandleOrderEvent pushes one mainstream order event back to Shopify when appropriate.
	HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error
}

// ContactSource defines local contact lookup behavior needed before Shopify order creation.
type ContactSource interface {
	// Get handles contact retrieval by id.
	Get(ctx context.Context, id string) (*contactsdomain.Contact, error)
}

// ContactEventHandler defines local contact outbound synchronization behavior.
type ContactEventHandler interface {
	// HandleContactEvent pushes one local contact to Shopify when needed.
	HandleContactEvent(ctx context.Context, payload contactsapplication.ContactEventPayload) error
}

// MainstreamUpdateService defines outbound Shopify order update behavior.
type MainstreamUpdateService struct {
	// destination defines Shopify destination dependencies.
	destination shopifyport.OrderDestination
	// links defines Shopify sync-link persistence dependencies.
	links shopifyport.SyncLinkRepository
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// destinationBreaker defines optional outbound breaker behavior.
	destinationBreaker CircuitBreaker
	// contactSource defines optional local contact lookup dependencies.
	contactSource ContactSource
	// contactHandler defines optional local-to-Shopify contact sync dependencies.
	contactHandler ContactEventHandler
}

var (
	// _ ensures MainstreamUpdateService satisfies handler contracts.
	_ OrderEventHandler = (*MainstreamUpdateService)(nil)
)

// NewMainstreamUpdateService creates outbound Shopify order update services.
func NewMainstreamUpdateService(destination shopifyport.OrderDestination, links shopifyport.SyncLinkRepository, providedLogger *zap.Logger, breakers ...CircuitBreakers) (*MainstreamUpdateService, error) {
	if destination == nil {
		return nil, ErrNilDestination
	}
	if links == nil {
		return nil, ErrNilLinkRepository
	}

	resolvedBreaker := CircuitBreakers{}
	if len(breakers) > 0 {
		resolvedBreaker = breakers[0]
	}
	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &MainstreamUpdateService{
		destination:        destination,
		links:              links,
		logger:             logger,
		destinationBreaker: resolvedBreaker.Destination,
	}, nil
}

// SetContactResolver configures contact creation/linking before missing Shopify order creation.
func (s *MainstreamUpdateService) SetContactResolver(source ContactSource, handler ContactEventHandler) {
	if s == nil {
		return
	}
	s.contactSource = source
	s.contactHandler = handler
}

// HandleOrderEvent pushes one mainstream order event back to Shopify when appropriate.
func (s *MainstreamUpdateService) HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error {
	if !strings.EqualFold(strings.TrimSpace(payload.Realm), "shopify") {
		s.logger.Debug("skip shopify outbound order sync: non-shopify realm", zap.String("realm", strings.TrimSpace(payload.Realm)), zap.String("order_id", strings.TrimSpace(payload.ID)))
		return nil
	}
	if strings.HasPrefix(strings.TrimSpace(payload.Source), "shopify_") {
		s.logger.Debug("skip shopify outbound order sync: shopify-originated event", zap.String("source", strings.TrimSpace(payload.Source)), zap.String("order_id", strings.TrimSpace(payload.ID)))
		return nil
	}
	status := resolvePayloadStatus(payload)
	if strings.TrimSpace(string(status)) == "" || strings.TrimSpace(payload.ID) == "" {
		s.logger.Info("skip shopify outbound order sync: missing order id or status", zap.String("order_id", strings.TrimSpace(payload.ID)), zap.String("status", strings.TrimSpace(string(status))))
		return nil
	}
	s.logger.Info(
		"shopify outbound order sync started",
		zap.String("order_id", strings.TrimSpace(payload.ID)),
		zap.String("identifier", strings.TrimSpace(payload.Identifier)),
		zap.String("contact_id", strings.TrimSpace(payload.ContactID)),
		zap.String("status", strings.TrimSpace(string(status))),
		zap.Int("items", len(payload.Items)),
	)

	link, err := s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindOrder, payload.ID)
	if err != nil {
		s.logger.Warn("shopify outbound order link lookup failed", zap.String("order_id", payload.ID), zap.Error(err))
		return err
	}
	if link == nil {
		s.logger.Info("shopify outbound order link not found; creating order if customer link exists", zap.String("order_id", payload.ID), zap.String("contact_id", payload.ContactID))
		return s.createAndLinkOrder(ctx, payload, status)
	}

	if strings.EqualFold(strings.TrimSpace(link.LastKnownStatus), strings.TrimSpace(string(status))) {
		s.logger.Info("skip shopify outbound order update: linked order already has status", zap.String("order_id", payload.ID), zap.String("shopify_id", link.ShopifyID), zap.String("status", string(status)))
		return nil
	}

	linkedCtx := shopifyport.WithShopDomain(ctx, link.ShopDomain)
	err = s.executeWithBreaker(s.destinationBreaker, ErrIntegrationUnavailable, func() error {
		return s.destination.UpdateOrderFromMainstream(linkedCtx, link.ShopifyID, shopifyport.MainstreamOrderUpdateCommand{Status: status})
	})
	if err != nil {
		if errors.Is(err, shopifyport.ErrOrderNotFound) {
			s.logger.Warn("linked shopify order missing; creating replacement", zap.String("order_id", payload.ID), zap.String("shopify_id", link.ShopifyID))
			return s.createAndLinkOrder(ctx, payload, status)
		}
		s.logger.Warn("shopify outbound order update failed", zap.String("order_id", payload.ID), zap.String("shopify_id", link.ShopifyID), zap.Error(err))
		if !errors.Is(err, ErrIntegrationUnavailable) {
			return fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
		}
		return err
	}

	if updateErr := s.links.UpdateLastKnownStatus(ctx, shopifyport.SyncKindOrder, payload.ID, string(status)); updateErr != nil {
		s.logger.Warn("update shopify last known status failed", zap.Error(updateErr))
		return updateErr
	}
	s.logger.Info("updated linked shopify order from order event", zap.String("order_id", payload.ID), zap.String("shopify_id", link.ShopifyID), zap.String("status", string(status)))

	return nil
}

func (s *MainstreamUpdateService) createAndLinkOrder(ctx context.Context, payload ordersport.OrderEventPayload, status ordersdomain.Status) error {
	contactID := strings.TrimSpace(payload.ContactID)
	if contactID == "" {
		s.logger.Warn("skip shopify outbound order create: missing contact id", zap.String("order_id", payload.ID))
		return nil
	}

	contactLink, err := s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindContact, contactID)
	if err != nil {
		s.logger.Warn("shopify outbound order customer link lookup failed", zap.String("order_id", payload.ID), zap.String("contact_id", contactID), zap.Error(err))
		return err
	}
	if contactLink == nil || strings.TrimSpace(contactLink.ShopifyID) == "" {
		if ensureErr := s.ensureContactLinked(ctx, contactID); ensureErr != nil {
			return ensureErr
		}
		contactLink, err = s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindContact, contactID)
		if err != nil {
			s.logger.Warn("shopify outbound order customer link reload failed", zap.String("order_id", payload.ID), zap.String("contact_id", contactID), zap.Error(err))
			return err
		}
		if contactLink == nil || strings.TrimSpace(contactLink.ShopifyID) == "" {
			s.logger.Warn("skip shopify outbound order create: contact is not linked to a shopify customer", zap.String("order_id", payload.ID), zap.String("contact_id", contactID))
			return nil
		}
	}

	command := buildMainstreamOrderCreateCommand(payload, status, contactLink.ShopifyID)
	linkedCtx := shopifyport.WithShopDomain(ctx, contactLink.ShopDomain)
	var created shopifyport.ShopifyOrder
	err = s.executeWithBreaker(s.destinationBreaker, ErrIntegrationUnavailable, func() error {
		var createErr error
		created, createErr = s.destination.CreateOrderFromMainstream(linkedCtx, command)
		return createErr
	})
	if err != nil {
		s.logger.Warn("create shopify order from order event failed", zap.String("order_id", payload.ID), zap.String("contact_id", contactID), zap.String("shopify_customer_id", contactLink.ShopifyID), zap.Error(err))
		if !errors.Is(err, ErrIntegrationUnavailable) {
			return fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
		}
		return err
	}
	if strings.TrimSpace(created.ID) == "" {
		return ErrCreatedOrderIDRequired
	}

	lastSyncedAt := time.Now().UTC()
	_, err = s.links.UpsertLink(ctx, shopifyport.UpsertSyncLinkInput{
		Kind:            shopifyport.SyncKindOrder,
		ShopDomain:      preferNonEmpty(created.ShopDomain, contactLink.ShopDomain),
		ShopifyID:       created.ID,
		MannaiahID:      payload.ID,
		LastKnownStatus: string(status),
		LastSyncedAt:    &lastSyncedAt,
	})
	if err != nil {
		s.logger.Warn("persist shopify order link failed", zap.String("order_id", payload.ID), zap.String("shopify_id", created.ID), zap.Error(err))
		return err
	}
	s.logger.Info("created shopify order from order event", zap.String("order_id", payload.ID), zap.String("shopify_id", created.ID), zap.String("shop_domain", preferNonEmpty(created.ShopDomain, contactLink.ShopDomain)), zap.String("shopify_customer_id", contactLink.ShopifyID))

	return nil
}

func (s *MainstreamUpdateService) ensureContactLinked(ctx context.Context, contactID string) error {
	if s.contactSource == nil || s.contactHandler == nil {
		s.logger.Warn("skip shopify contact pre-sync before order create: contact resolver is unavailable", zap.String("contact_id", contactID))
		return nil
	}
	contact, err := s.contactSource.Get(ctx, contactID)
	if err != nil {
		s.logger.Warn("shopify contact pre-sync lookup failed before order create", zap.String("contact_id", contactID), zap.Error(err))
		return err
	}
	if contact == nil {
		s.logger.Warn("skip shopify contact pre-sync before order create: contact not found", zap.String("contact_id", contactID))
		return nil
	}
	if err := s.contactHandler.HandleContactEvent(ctx, buildContactEventPayload(*contact)); err != nil {
		s.logger.Warn("shopify contact pre-sync failed before order create", zap.String("contact_id", contactID), zap.Error(err))
		return err
	}
	return nil
}

func (s *MainstreamUpdateService) executeWithBreaker(breaker CircuitBreaker, unavailableErr error, fn func() error) error {
	if breaker == nil {
		return fn()
	}

	var operationErr error
	err := breaker.Execute(func() error {
		operationErr = fn()
		return operationErr
	})
	if err == nil {
		return nil
	}
	if operationErr != nil {
		return operationErr
	}

	return unavailableErr
}

func resolvePayloadStatus(payload ordersport.OrderEventPayload) ordersdomain.Status {
	if strings.TrimSpace(payload.LatestStatus.Status) != "" {
		return ordersdomain.Status(strings.TrimSpace(payload.LatestStatus.Status))
	}
	if strings.TrimSpace(payload.CurrentStatus) != "" {
		return ordersdomain.Status(strings.TrimSpace(payload.CurrentStatus))
	}
	return ""
}

func buildMainstreamOrderCreateCommand(payload ordersport.OrderEventPayload, status ordersdomain.Status, customerID string) shopifyport.MainstreamOrderCreateCommand {
	items := make([]shopifyport.MainstreamOrderCreateItem, 0, len(payload.Items))
	for _, row := range payload.Items {
		items = append(items, shopifyport.MainstreamOrderCreateItem{
			SKU:      strings.TrimSpace(row.SKU),
			Title:    strings.TrimSpace(row.AlternateName),
			Quantity: row.Quantity,
			Price:    row.Value,
		})
	}

	charges := make([]shopifyport.MainstreamOrderCreateShippingCharge, 0, len(payload.ShippingCharges))
	for _, row := range payload.ShippingCharges {
		charges = append(charges, shopifyport.MainstreamOrderCreateShippingCharge{
			Code:  strings.TrimSpace(row.MethodID),
			Title: strings.TrimSpace(row.MethodTitle),
			Price: row.Price,
		})
	}

	return shopifyport.MainstreamOrderCreateCommand{
		OrderID:         strings.TrimSpace(payload.ID),
		Identifier:      strings.TrimSpace(payload.Identifier),
		CustomerID:      strings.TrimSpace(customerID),
		Status:          status,
		Items:           items,
		ShippingCharges: charges,
		CreatedAt:       payload.CreatedAt,
	}
}

func preferNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func buildContactEventPayload(contact contactsdomain.Contact) contactsapplication.ContactEventPayload {
	return contactsapplication.ContactEventPayload{
		ID:             strings.TrimSpace(contact.ID),
		DocumentType:   contact.DocumentType,
		DocumentNumber: strings.TrimSpace(contact.DocumentNumber),
		LegalName:      strings.TrimSpace(contact.LegalName),
		FirstName:      strings.TrimSpace(contact.FirstName),
		LastName:       strings.TrimSpace(contact.LastName),
		Email:          strings.TrimSpace(contact.Email),
		Phone:          strings.TrimSpace(contact.Phone),
		Address:        strings.TrimSpace(contact.Address),
		AddressExtra:   strings.TrimSpace(contact.AddressExtra),
		CityCode:       strings.TrimSpace(contact.CityCode),
		Metadata:       cloneStringMap(contact.Metadata),
		CreatedAt:      contact.CreatedAt,
		UpdatedAt:      contact.UpdatedAt,
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
