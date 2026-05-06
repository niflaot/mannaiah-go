package service

import (
	"context"
	"testing"

	contactsapplication "mannaiah/module/contacts/application"
	contactsdomain "mannaiah/module/contacts/domain"
	contactsport "mannaiah/module/contacts/port"
	shopifyport "mannaiah/module/shopify/port"

	"go.uber.org/zap"
)

type contactsServiceStub struct {
	createResult *contactsdomain.Contact
	updateResult *contactsdomain.Contact
	listResult   *contactsapplication.ListResult
}

func (s *contactsServiceStub) Create(ctx context.Context, command contactsapplication.CreateCommand) (*contactsdomain.Contact, error) {
	_ = ctx
	_ = command
	return s.createResult, nil
}

func (s *contactsServiceStub) Get(ctx context.Context, id string) (*contactsdomain.Contact, error) {
	_ = ctx
	_ = id
	return nil, nil
}

func (s *contactsServiceStub) List(ctx context.Context, query contactsport.ListQuery) (*contactsapplication.ListResult, error) {
	_ = ctx
	_ = query
	return s.listResult, nil
}

func (s *contactsServiceStub) Update(ctx context.Context, id string, command contactsapplication.UpdateCommand) (*contactsdomain.Contact, error) {
	_ = ctx
	_ = id
	_ = command
	return s.updateResult, nil
}

func (s *contactsServiceStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	_ = id
	return nil
}

type contactLinkRepositoryStub struct {
	existing *shopifyport.SyncLink
	upserts  int
}

func (s *contactLinkRepositoryStub) GetLinkByShopifyID(ctx context.Context, kind shopifyport.SyncKind, shopDomain string, shopifyID string) (*shopifyport.SyncLink, error) {
	_ = ctx
	_ = kind
	_ = shopDomain
	_ = shopifyID
	return s.existing, nil
}

func (s *contactLinkRepositoryStub) GetLinkByMannaiahID(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string) (*shopifyport.SyncLink, error) {
	_ = ctx
	_ = kind
	_ = mannaiahID
	return nil, nil
}

func (s *contactLinkRepositoryStub) UpsertLink(ctx context.Context, input shopifyport.UpsertSyncLinkInput) (*shopifyport.SyncLink, error) {
	_ = ctx
	_ = input
	s.upserts++
	return &shopifyport.SyncLink{ShopifyID: input.ShopifyID, MannaiahID: input.MannaiahID}, nil
}

func (s *contactLinkRepositoryStub) UpdateLastKnownStatus(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string, status string) error {
	_ = ctx
	_ = kind
	_ = mannaiahID
	_ = status
	return nil
}

// TestContactUpserterUpsertsLinkAfterSuccessfulCreate verifies Shopify imports persist local sync links.
func TestContactUpserterUpsertsLinkAfterSuccessfulCreate(t *testing.T) {
	links := &contactLinkRepositoryStub{}
	upserter, err := NewUpserter(
		&contactsServiceStub{createResult: &contactsdomain.Contact{ID: "contact-1", FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}},
		links,
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	contact, err := upserter.UpsertContact(shopifyport.WithShopDomain(context.Background(), "flock-6591.myshopify.com"), shopifyport.ContactSyncCommand{
		ShopifyID: "123",
		Email:     "ada@example.com",
		FirstName: "Ada",
		LastName:  "Lovelace",
	})
	if err != nil {
		t.Fatalf("UpsertContact() error = %v", err)
	}
	if contact.ID != "contact-1" {
		t.Fatalf("contact ID = %q, want contact-1", contact.ID)
	}
	if links.upserts != 1 {
		t.Fatalf("link upserts = %d, want 1", links.upserts)
	}
}

// TestContactUpserterUpdatesExistingContact verifies Shopify imports deduplicate by email before updating.
func TestContactUpserterUpdatesExistingContact(t *testing.T) {
	existing := contactsdomain.Contact{ID: "contact-1", FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}
	links := &contactLinkRepositoryStub{existing: &shopifyport.SyncLink{ShopifyID: "123", MannaiahID: "contact-1"}}
	upserter, err := NewUpserter(
		&contactsServiceStub{
			updateResult: &contactsdomain.Contact{ID: "contact-1", FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"},
			listResult:   &contactsapplication.ListResult{Data: []contactsdomain.Contact{existing}},
		},
		links,
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	contact, err := upserter.UpsertContact(context.Background(), shopifyport.ContactSyncCommand{
		ShopifyID: "123",
		Email:     "ada@example.com",
		FirstName: "Ada",
		LastName:  "Lovelace",
	})
	if err != nil {
		t.Fatalf("UpsertContact() error = %v", err)
	}
	if contact.ID != "contact-1" {
		t.Fatalf("contact ID = %q, want contact-1", contact.ID)
	}
	if links.upserts != 1 {
		t.Fatalf("link upserts = %d, want 1", links.upserts)
	}
}
