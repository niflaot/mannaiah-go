package service

import (
	"context"
	"testing"

	contactsapplication "mannaiah/module/contacts/application"
	shopifyport "mannaiah/module/shopify/port"
)

type mainstreamCustomerSourceStub struct {
	customerByID    map[string]shopifyport.ShopifyCustomer
	customerByEmail map[string]shopifyport.ShopifyCustomer
}

func (s *mainstreamCustomerSourceStub) Validate(ctx context.Context) error {
	_ = ctx
	return nil
}

func (s *mainstreamCustomerSourceStub) GetCustomer(ctx context.Context, id string) (shopifyport.ShopifyCustomer, error) {
	_ = ctx
	customer, ok := s.customerByID[id]
	if !ok {
		return shopifyport.ShopifyCustomer{}, shopifyport.ErrCustomerNotFound
	}
	return customer, nil
}

func (s *mainstreamCustomerSourceStub) FindCustomerByEmail(ctx context.Context, email string) (shopifyport.ShopifyCustomer, error) {
	_ = ctx
	customer, ok := s.customerByEmail[email]
	if !ok {
		return shopifyport.ShopifyCustomer{}, shopifyport.ErrCustomerNotFound
	}
	return customer, nil
}

func (s *mainstreamCustomerSourceStub) ListCustomers(ctx context.Context, sinceID string, limit int) ([]shopifyport.ShopifyCustomer, bool, error) {
	_ = ctx
	_ = sinceID
	_ = limit
	return nil, false, nil
}

type mainstreamCustomerDestinationStub struct {
	createCalls int
	updateCalls int
	created     shopifyport.ShopifyCustomer
}

func (s *mainstreamCustomerDestinationStub) Validate(ctx context.Context) error {
	_ = ctx
	return nil
}

func (s *mainstreamCustomerDestinationStub) CreateCustomerFromMainstream(ctx context.Context, command shopifyport.MainstreamCustomerUpsertCommand) (shopifyport.ShopifyCustomer, error) {
	_ = ctx
	s.createCalls++
	if s.created.ID != "" {
		return s.created, nil
	}
	return shopifyport.ShopifyCustomer{
		ID:         "created-shopify-id",
		ShopDomain: "flock-6591.myshopify.com",
		Email:      command.Email,
		FirstName:  command.FirstName,
		LastName:   command.LastName,
		Phone:      command.Phone,
	}, nil
}

func (s *mainstreamCustomerDestinationStub) UpdateCustomerFromMainstream(ctx context.Context, id string, command shopifyport.MainstreamCustomerUpsertCommand) error {
	_ = ctx
	_ = id
	_ = command
	s.updateCalls++
	return nil
}

func (s *mainstreamCustomerDestinationStub) UpdateCustomerTags(ctx context.Context, id string, tags []string) error {
	_ = ctx
	_ = id
	_ = tags
	return nil
}

func (s *mainstreamCustomerDestinationStub) AppendCustomerNote(ctx context.Context, id string, note string) error {
	_ = ctx
	_ = id
	_ = note
	return nil
}

type mainstreamLinkRepositoryStub struct {
	byMannaiah map[string]*shopifyport.SyncLink
	upserts    []shopifyport.UpsertSyncLinkInput
}

func (s *mainstreamLinkRepositoryStub) GetLinkByShopifyID(ctx context.Context, kind shopifyport.SyncKind, shopDomain string, shopifyID string) (*shopifyport.SyncLink, error) {
	_ = ctx
	_ = kind
	_ = shopDomain
	_ = shopifyID
	return nil, nil
}

func (s *mainstreamLinkRepositoryStub) GetLinkByMannaiahID(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string) (*shopifyport.SyncLink, error) {
	_ = ctx
	_ = kind
	if s.byMannaiah == nil {
		return nil, nil
	}
	return s.byMannaiah[mannaiahID], nil
}

func (s *mainstreamLinkRepositoryStub) UpsertLink(ctx context.Context, input shopifyport.UpsertSyncLinkInput) (*shopifyport.SyncLink, error) {
	_ = ctx
	s.upserts = append(s.upserts, input)
	if s.byMannaiah == nil {
		s.byMannaiah = map[string]*shopifyport.SyncLink{}
	}
	link := &shopifyport.SyncLink{
		Kind:       input.Kind,
		ShopDomain: input.ShopDomain,
		ShopifyID:  input.ShopifyID,
		MannaiahID: input.MannaiahID,
	}
	s.byMannaiah[input.MannaiahID] = link
	return link, nil
}

func (s *mainstreamLinkRepositoryStub) UpdateLastKnownStatus(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string, status string) error {
	_ = ctx
	_ = kind
	_ = mannaiahID
	_ = status
	return nil
}

// TestMainstreamContactUpdateServiceLinksInboundCreatedContacts verifies inbound-created events stitch links instead of creating duplicates.
func TestMainstreamContactUpdateServiceLinksInboundCreatedContacts(t *testing.T) {
	links := &mainstreamLinkRepositoryStub{}
	destination := &mainstreamCustomerDestinationStub{}
	service, err := NewMainstreamUpdateService(&mainstreamCustomerSourceStub{}, destination, links, nil)
	if err != nil {
		t.Fatalf("NewMainstreamUpdateService() error = %v", err)
	}

	err = service.HandleContactEvent(context.Background(), contactsapplication.ContactEventPayload{
		ID:        "contact-1",
		Email:     "shop@example.com",
		FirstName: "Shop",
		LastName:  "Customer",
		Metadata: map[string]string{
			metadataKeyShopifyCustomerID: "10451279282474",
			metadataKeyShopifyShopDomain: "flock-6591.myshopify.com",
		},
	})
	if err != nil {
		t.Fatalf("HandleContactEvent() error = %v", err)
	}
	if destination.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", destination.createCalls)
	}
	if destination.updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", destination.updateCalls)
	}
	if len(links.upserts) != 1 {
		t.Fatalf("link upserts = %d, want 1", len(links.upserts))
	}
	if links.upserts[0].ShopifyID != "10451279282474" {
		t.Fatalf("upsert ShopifyID = %q, want 10451279282474", links.upserts[0].ShopifyID)
	}
}

// TestMainstreamContactUpdateServiceCreatesMissingLinkedCustomers verifies local-only contacts are created in Shopify and linked.
func TestMainstreamContactUpdateServiceCreatesMissingLinkedCustomers(t *testing.T) {
	links := &mainstreamLinkRepositoryStub{}
	destination := &mainstreamCustomerDestinationStub{
		created: shopifyport.ShopifyCustomer{ID: "created-1", ShopDomain: "flock-6591.myshopify.com"},
	}
	service, err := NewMainstreamUpdateService(&mainstreamCustomerSourceStub{}, destination, links, nil)
	if err != nil {
		t.Fatalf("NewMainstreamUpdateService() error = %v", err)
	}

	err = service.HandleContactEvent(context.Background(), contactsapplication.ContactEventPayload{
		ID:        "contact-2",
		Email:     "new@example.com",
		FirstName: "New",
		LastName:  "Contact",
	})
	if err != nil {
		t.Fatalf("HandleContactEvent() error = %v", err)
	}
	if destination.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", destination.createCalls)
	}
	if len(links.upserts) != 1 {
		t.Fatalf("link upserts = %d, want 1", len(links.upserts))
	}
	if links.upserts[0].MannaiahID != "contact-2" {
		t.Fatalf("upsert MannaiahID = %q, want contact-2", links.upserts[0].MannaiahID)
	}
}

// TestMainstreamContactUpdateServiceSkipsEquivalentLinkedCustomers verifies webhook echoes do not loop back out.
func TestMainstreamContactUpdateServiceSkipsEquivalentLinkedCustomers(t *testing.T) {
	links := &mainstreamLinkRepositoryStub{
		byMannaiah: map[string]*shopifyport.SyncLink{
			"contact-3": {ShopifyID: "shopify-3", MannaiahID: "contact-3"},
		},
	}
	source := &mainstreamCustomerSourceStub{
		customerByID: map[string]shopifyport.ShopifyCustomer{
			"shopify-3": {
				ID:         "shopify-3",
				ShopDomain: "flock-6591.myshopify.com",
				Email:      "same@example.com",
				FirstName:  "Same",
				LastName:   "Customer",
				Phone:      "3001234567",
				Tags:       "vip, mannaiah:synced",
				Note:       "[Mannaiah] contact_id=contact-3",
				DefaultAddress: &shopifyport.ShopifyAddress{
					Company:  "12345678",
					Address1: "Street 1",
					Address2: "Apt 2",
					City:     "Bogota",
				},
			},
		},
	}
	destination := &mainstreamCustomerDestinationStub{}
	service, err := NewMainstreamUpdateService(source, destination, links, nil)
	if err != nil {
		t.Fatalf("NewMainstreamUpdateService() error = %v", err)
	}

	err = service.HandleContactEvent(context.Background(), contactsapplication.ContactEventPayload{
		ID:             "contact-3",
		Email:          "same@example.com",
		FirstName:      "Same",
		LastName:       "Customer",
		Phone:          "3001234567",
		DocumentNumber: "12345678",
		Address:        "Street 1",
		AddressExtra:   "Apt 2",
		CityCode:       "Bogota",
	})
	if err != nil {
		t.Fatalf("HandleContactEvent() error = %v", err)
	}
	if destination.updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", destination.updateCalls)
	}
}

// TestMainstreamContactUpdateServiceUpdatesDivergedLinkedCustomers verifies local edits are pushed back to Shopify.
func TestMainstreamContactUpdateServiceUpdatesDivergedLinkedCustomers(t *testing.T) {
	links := &mainstreamLinkRepositoryStub{
		byMannaiah: map[string]*shopifyport.SyncLink{
			"contact-4": {ShopifyID: "shopify-4", MannaiahID: "contact-4"},
		},
	}
	source := &mainstreamCustomerSourceStub{
		customerByID: map[string]shopifyport.ShopifyCustomer{
			"shopify-4": {
				ID:         "shopify-4",
				ShopDomain: "flock-6591.myshopify.com",
				Email:      "old@example.com",
				FirstName:  "Old",
				LastName:   "Customer",
				Tags:       "mannaiah:synced",
				Note:       "[Mannaiah] contact_id=contact-4",
			},
		},
	}
	destination := &mainstreamCustomerDestinationStub{}
	service, err := NewMainstreamUpdateService(source, destination, links, nil)
	if err != nil {
		t.Fatalf("NewMainstreamUpdateService() error = %v", err)
	}

	err = service.HandleContactEvent(context.Background(), contactsapplication.ContactEventPayload{
		ID:        "contact-4",
		Email:     "new@example.com",
		FirstName: "New",
		LastName:  "Customer",
	})
	if err != nil {
		t.Fatalf("HandleContactEvent() error = %v", err)
	}
	if destination.updateCalls != 1 {
		t.Fatalf("update calls = %d, want 1", destination.updateCalls)
	}
}
