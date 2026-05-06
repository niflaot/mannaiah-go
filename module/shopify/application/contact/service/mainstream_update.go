package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	contactsapplication "mannaiah/module/contacts/application"
	shopifyport "mannaiah/module/shopify/port"

	"go.uber.org/zap"
)

var (
	// ErrNilMainstreamSource is returned when a nil Shopify customer source is provided.
	ErrNilMainstreamSource = errors.New("shopify customer source must not be nil")
	// ErrNilMainstreamDestination is returned when a nil Shopify customer destination is provided.
	ErrNilMainstreamDestination = errors.New("shopify customer destination must not be nil")
	// ErrNilMainstreamLinkRepository is returned when a nil sync-link repository is provided.
	ErrNilMainstreamLinkRepository = errors.New("shopify contact link repository must not be nil")
)

const (
	metadataKeyShopifyCustomerID = "shopify_customer_id"
	metadataKeyShopifyShopDomain = "shopify_shop_domain"
	outboundSyncedTag            = "mannaiah:synced"
)

// ContactEventHandler defines mainstream contact integration event handling behavior.
type ContactEventHandler interface {
	// HandleContactEvent pushes one mainstream contact event back to Shopify when appropriate.
	HandleContactEvent(ctx context.Context, payload contactsapplication.ContactEventPayload) error
}

// MainstreamContactUpdateService defines outbound Shopify contact update behavior.
type MainstreamContactUpdateService struct {
	// source defines Shopify lookup dependencies.
	source shopifyport.CustomerSource
	// destination defines Shopify write dependencies.
	destination shopifyport.CustomerDestination
	// links defines Shopify sync-link persistence dependencies.
	links shopifyport.SyncLinkRepository
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// sourceBreaker defines optional source breaker behavior.
	sourceBreaker CircuitBreaker
	// destinationBreaker defines optional destination breaker behavior.
	destinationBreaker CircuitBreaker
}

var (
	// _ ensures MainstreamContactUpdateService satisfies handler contracts.
	_ ContactEventHandler = (*MainstreamContactUpdateService)(nil)
)

// NewMainstreamUpdateService creates outbound Shopify contact update services.
func NewMainstreamUpdateService(source shopifyport.CustomerSource, destination shopifyport.CustomerDestination, links shopifyport.SyncLinkRepository, providedLogger *zap.Logger, breakers ...CircuitBreakers) (*MainstreamContactUpdateService, error) {
	if source == nil {
		return nil, ErrNilMainstreamSource
	}
	if destination == nil {
		return nil, ErrNilMainstreamDestination
	}
	if links == nil {
		return nil, ErrNilMainstreamLinkRepository
	}

	resolvedBreaker := CircuitBreakers{}
	if len(breakers) > 0 {
		resolvedBreaker = breakers[0]
	}
	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &MainstreamContactUpdateService{
		source:             source,
		destination:        destination,
		links:              links,
		logger:             logger,
		sourceBreaker:      resolvedBreaker.Source,
		destinationBreaker: resolvedBreaker.Destination,
	}, nil
}

// HandleContactEvent pushes one mainstream contact event back to Shopify when appropriate.
func (s *MainstreamContactUpdateService) HandleContactEvent(ctx context.Context, payload contactsapplication.ContactEventPayload) error {
	contactID := strings.TrimSpace(payload.ID)
	if contactID == "" {
		s.logger.Info("skip shopify outbound contact sync: empty contact id")
		return nil
	}
	s.logger.Info(
		"shopify outbound contact sync started",
		zap.String("contact_id", contactID),
		zap.Bool("has_email", strings.TrimSpace(payload.Email) != ""),
		zap.Bool("has_shopify_customer_metadata", strings.TrimSpace(payload.Metadata[metadataKeyShopifyCustomerID]) != ""),
	)

	link, err := s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindContact, contactID)
	if err != nil {
		s.logger.Warn("shopify outbound contact link lookup failed", zap.String("contact_id", contactID), zap.Error(err))
		return err
	}

	if link == nil {
		s.logger.Info("shopify outbound contact link not found; checking metadata", zap.String("contact_id", contactID))
		linked, linkErr := s.linkFromPayloadMetadata(ctx, payload)
		if linkErr != nil {
			s.logger.Warn("shopify outbound contact metadata link failed", zap.String("contact_id", contactID), zap.Error(linkErr))
			return linkErr
		}
		if linked {
			s.logger.Info("shopify outbound contact linked from metadata; skipping write-back", zap.String("contact_id", contactID))
			return nil
		}
	}

	if link == nil {
		link, err = s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindContact, contactID)
		if err != nil {
			s.logger.Warn("shopify outbound contact link reload failed", zap.String("contact_id", contactID), zap.Error(err))
			return err
		}
	}

	command := buildMainstreamCustomerUpsertCommand(payload)
	if link != nil {
		s.logger.Info("shopify outbound contact using existing link", zap.String("contact_id", contactID), zap.String("shopify_id", link.ShopifyID), zap.String("shop_domain", link.ShopDomain))
		return s.syncLinkedCustomer(ctx, payload, command, link)
	}
	if strings.TrimSpace(command.Email) == "" {
		s.logger.Info("skip shopify outbound contact sync: no link and no email", zap.String("contact_id", contactID))
		return nil
	}

	s.logger.Info("shopify outbound contact has no link; searching by email", zap.String("contact_id", contactID))
	return s.syncUnlinkedCustomer(ctx, payload, command)
}

func (s *MainstreamContactUpdateService) linkFromPayloadMetadata(ctx context.Context, payload contactsapplication.ContactEventPayload) (bool, error) {
	shopifyID := strings.TrimSpace(payload.Metadata[metadataKeyShopifyCustomerID])
	if shopifyID == "" {
		return false, nil
	}

	shopDomain := strings.TrimSpace(payload.Metadata[metadataKeyShopifyShopDomain])
	if err := s.upsertLink(ctx, shopDomain, shopifyID, payload.ID); err != nil {
		return false, err
	}
	s.logger.Info("shopify outbound contact link persisted from metadata", zap.String("contact_id", payload.ID), zap.String("shopify_id", shopifyID), zap.String("shop_domain", shopDomain))

	return true, nil
}

func (s *MainstreamContactUpdateService) syncLinkedCustomer(ctx context.Context, payload contactsapplication.ContactEventPayload, command shopifyport.MainstreamCustomerUpsertCommand, link *shopifyport.SyncLink) error {
	linkedCtx := shopifyport.WithShopDomain(ctx, link.ShopDomain)
	customer, err := s.loadCustomer(linkedCtx, link.ShopifyID)
	if err != nil {
		if errors.Is(err, shopifyport.ErrCustomerNotFound) {
			s.logger.Warn("shopify linked customer missing; creating replacement", zap.String("contact_id", payload.ID), zap.String("shopify_id", link.ShopifyID))
			return s.createAndLinkCustomer(linkedCtx, command)
		}
		s.logger.Warn("shopify linked customer load failed", zap.String("contact_id", payload.ID), zap.String("shopify_id", link.ShopifyID), zap.Error(err))
		return err
	}

	if customerMatchesPayload(customer, payload) {
		s.logger.Info("skip shopify outbound contact update: linked customer already matches", zap.String("contact_id", payload.ID), zap.String("shopify_id", customer.ID))
		return s.upsertLink(ctx, customer.ShopDomain, customer.ID, payload.ID)
	}

	s.logger.Info("updating linked shopify customer from contact event", zap.String("contact_id", payload.ID), zap.String("shopify_id", customer.ID), zap.String("shop_domain", customer.ShopDomain))
	err = s.executeWithBreaker(s.destinationBreaker, ErrIntegrationUnavailable, func() error {
		return s.destination.UpdateCustomerFromMainstream(linkedCtx, customer.ID, command)
	})
	if err != nil {
		s.logger.Warn("update linked shopify customer failed", zap.String("contact_id", payload.ID), zap.String("shopify_id", customer.ID), zap.Error(err))
		if !errors.Is(err, ErrIntegrationUnavailable) {
			return fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
		}
		return err
	}
	s.logger.Info("updated linked shopify customer from contact event", zap.String("contact_id", payload.ID), zap.String("shopify_id", customer.ID))

	return s.upsertLink(ctx, customer.ShopDomain, customer.ID, payload.ID)
}

func (s *MainstreamContactUpdateService) syncUnlinkedCustomer(ctx context.Context, payload contactsapplication.ContactEventPayload, command shopifyport.MainstreamCustomerUpsertCommand) error {
	customer, err := s.findCustomerByEmail(ctx, command.Email)
	if err != nil && !errors.Is(err, shopifyport.ErrCustomerNotFound) {
		s.logger.Warn("shopify customer email lookup failed", zap.String("contact_id", payload.ID), zap.Error(err))
		return err
	}
	if err == nil {
		if !customerMatchesPayload(customer, payload) {
			s.logger.Info("updating email-matched shopify customer from contact event", zap.String("contact_id", payload.ID), zap.String("shopify_id", customer.ID), zap.String("shop_domain", customer.ShopDomain))
			updateErr := s.executeWithBreaker(s.destinationBreaker, ErrIntegrationUnavailable, func() error {
				return s.destination.UpdateCustomerFromMainstream(ctx, customer.ID, command)
			})
			if updateErr != nil {
				s.logger.Warn("update email-matched shopify customer failed", zap.String("contact_id", payload.ID), zap.String("shopify_id", customer.ID), zap.Error(updateErr))
				if !errors.Is(updateErr, ErrIntegrationUnavailable) {
					return fmt.Errorf("%w: %v", ErrIntegrationUnavailable, updateErr)
				}
				return updateErr
			}
		} else {
			s.logger.Info("skip shopify outbound contact update: email-matched customer already matches", zap.String("contact_id", payload.ID), zap.String("shopify_id", customer.ID))
		}
		return s.upsertLink(ctx, customer.ShopDomain, customer.ID, payload.ID)
	}

	s.logger.Info("creating shopify customer from contact event", zap.String("contact_id", payload.ID))
	return s.createAndLinkCustomer(ctx, command)
}

func (s *MainstreamContactUpdateService) createAndLinkCustomer(ctx context.Context, command shopifyport.MainstreamCustomerUpsertCommand) error {
	var customer shopifyport.ShopifyCustomer
	err := s.executeWithBreaker(s.destinationBreaker, ErrIntegrationUnavailable, func() error {
		var createErr error
		customer, createErr = s.destination.CreateCustomerFromMainstream(ctx, command)
		return createErr
	})
	if err != nil {
		s.logger.Warn("create shopify customer from contact event failed", zap.String("contact_id", command.ContactID), zap.Error(err))
		if !errors.Is(err, ErrIntegrationUnavailable) {
			return fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
		}
		return err
	}
	s.logger.Info("created shopify customer from contact event", zap.String("contact_id", command.ContactID), zap.String("shopify_id", customer.ID), zap.String("shop_domain", customer.ShopDomain))

	return s.upsertLink(ctx, customer.ShopDomain, customer.ID, command.ContactID)
}

func (s *MainstreamContactUpdateService) upsertLink(ctx context.Context, shopDomain string, shopifyID string, contactID string) error {
	lastSyncedAt := time.Now().UTC()
	_, err := s.links.UpsertLink(ctx, shopifyport.UpsertSyncLinkInput{
		Kind:         shopifyport.SyncKindContact,
		ShopDomain:   strings.TrimSpace(shopDomain),
		ShopifyID:    strings.TrimSpace(shopifyID),
		MannaiahID:   strings.TrimSpace(contactID),
		LastSyncedAt: &lastSyncedAt,
	})
	if err != nil {
		s.logger.Warn("persist shopify contact link failed", zap.String("contact_id", contactID), zap.String("shopify_id", shopifyID), zap.String("shop_domain", shopDomain), zap.Error(err))
		return err
	}
	s.logger.Info("persisted shopify contact link", zap.String("contact_id", contactID), zap.String("shopify_id", shopifyID), zap.String("shop_domain", shopDomain))
	return err
}

func (s *MainstreamContactUpdateService) loadCustomer(ctx context.Context, shopifyID string) (shopifyport.ShopifyCustomer, error) {
	var customer shopifyport.ShopifyCustomer
	err := s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
		var sourceErr error
		customer, sourceErr = s.source.GetCustomer(ctx, shopifyID)
		return sourceErr
	})
	if err != nil {
		if errors.Is(err, shopifyport.ErrCustomerNotFound) || errors.Is(err, ErrIntegrationUnavailable) {
			return shopifyport.ShopifyCustomer{}, err
		}
		return shopifyport.ShopifyCustomer{}, fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
	}

	return customer, nil
}

func (s *MainstreamContactUpdateService) findCustomerByEmail(ctx context.Context, email string) (shopifyport.ShopifyCustomer, error) {
	var customer shopifyport.ShopifyCustomer
	err := s.executeWithBreaker(s.sourceBreaker, ErrIntegrationUnavailable, func() error {
		var sourceErr error
		customer, sourceErr = s.source.FindCustomerByEmail(ctx, email)
		return sourceErr
	})
	if err != nil {
		if errors.Is(err, shopifyport.ErrCustomerNotFound) || errors.Is(err, ErrIntegrationUnavailable) {
			return shopifyport.ShopifyCustomer{}, err
		}
		return shopifyport.ShopifyCustomer{}, fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
	}

	return customer, nil
}

func (s *MainstreamContactUpdateService) executeWithBreaker(breaker CircuitBreaker, unavailableErr error, fn func() error) error {
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

func buildMainstreamCustomerUpsertCommand(payload contactsapplication.ContactEventPayload) shopifyport.MainstreamCustomerUpsertCommand {
	return shopifyport.MainstreamCustomerUpsertCommand{
		ContactID:      strings.TrimSpace(payload.ID),
		Email:          strings.TrimSpace(payload.Email),
		LegalName:      strings.TrimSpace(payload.LegalName),
		FirstName:      strings.TrimSpace(payload.FirstName),
		LastName:       strings.TrimSpace(payload.LastName),
		Phone:          strings.TrimSpace(payload.Phone),
		DocumentNumber: strings.TrimSpace(payload.DocumentNumber),
		Address:        strings.TrimSpace(payload.Address),
		AddressExtra:   strings.TrimSpace(payload.AddressExtra),
		CityCode:       strings.TrimSpace(payload.CityCode),
	}
}

func customerMatchesPayload(customer shopifyport.ShopifyCustomer, payload contactsapplication.ContactEventPayload) bool {
	if !strings.EqualFold(strings.TrimSpace(customer.Email), strings.TrimSpace(payload.Email)) {
		return false
	}
	if strings.TrimSpace(customer.FirstName) != outboundCustomerFirstName(payload) {
		return false
	}
	if strings.TrimSpace(customer.LastName) != outboundCustomerLastName(payload) {
		return false
	}
	if strings.TrimSpace(customer.Phone) != strings.TrimSpace(payload.Phone) {
		return false
	}
	if !customerTagsContain(customer.Tags, outboundSyncedTag) {
		return false
	}
	if note := outboundCustomerContactNote(payload.ID); note != "" && !strings.Contains(strings.TrimSpace(customer.Note), note) {
		return false
	}

	address := customer.DefaultAddress
	if address == nil {
		return strings.TrimSpace(payload.DocumentNumber) == "" &&
			strings.TrimSpace(payload.Address) == "" &&
			strings.TrimSpace(payload.AddressExtra) == "" &&
			strings.TrimSpace(payload.CityCode) == ""
	}

	return strings.TrimSpace(address.Company) == strings.TrimSpace(payload.DocumentNumber) &&
		strings.TrimSpace(address.Address1) == strings.TrimSpace(payload.Address) &&
		strings.TrimSpace(address.Address2) == strings.TrimSpace(payload.AddressExtra) &&
		strings.TrimSpace(address.City) == strings.TrimSpace(payload.CityCode)
}

func outboundCustomerFirstName(payload contactsapplication.ContactEventPayload) string {
	for _, value := range []string{payload.FirstName, payload.LegalName, "Mannaiah"} {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return "Mannaiah"
}

func outboundCustomerLastName(payload contactsapplication.ContactEventPayload) string {
	for _, value := range []string{payload.LastName, "Contact"} {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return "Contact"
}

func outboundCustomerContactNote(contactID string) string {
	trimmedID := strings.TrimSpace(contactID)
	if trimmedID == "" {
		return ""
	}
	return fmt.Sprintf("[Mannaiah] contact_id=%s", trimmedID)
}

func customerTagsContain(existing string, want string) bool {
	trimmedWant := strings.TrimSpace(want)
	if trimmedWant == "" {
		return true
	}
	for _, value := range strings.Split(existing, ",") {
		if strings.TrimSpace(value) == trimmedWant {
			return true
		}
	}
	return false
}
