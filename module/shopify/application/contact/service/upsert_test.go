package service

import (
	"context"
	"errors"
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

type customerDestinationStub struct {
	tagsCalls int
	noteCalls int
	err       error
}

func (s *customerDestinationStub) Validate(ctx context.Context) error {
	_ = ctx
	return nil
}

func (s *customerDestinationStub) UpdateCustomerTags(ctx context.Context, id string, tags []string) error {
	_ = ctx
	_ = id
	_ = tags
	s.tagsCalls++
	return s.err
}

func (s *customerDestinationStub) AppendCustomerNote(ctx context.Context, id string, note string) error {
	_ = ctx
	_ = id
	_ = note
	s.noteCalls++
	return s.err
}

// TestContactUpserterWritesBackAfterSuccessfulUpsert verifies customer sync markers are pushed after mainstream success.
func TestContactUpserterWritesBackAfterSuccessfulUpsert(t *testing.T) {
	destination := &customerDestinationStub{}
	upserter, err := NewUpserter(
		&contactsServiceStub{createResult: &contactsdomain.Contact{ID: "contact-1", FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}},
		&contactLinkRepositoryStub{},
		destination,
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
	if destination.tagsCalls != 1 {
		t.Fatalf("tag write-back calls = %d, want 1", destination.tagsCalls)
	}
	if destination.noteCalls != 1 {
		t.Fatalf("note write-back calls = %d, want 1", destination.noteCalls)
	}
}

// TestContactUpserterDoesNotAppendNoteForExistingLink verifies contact notes are only appended on new links.
func TestContactUpserterDoesNotAppendNoteForExistingLink(t *testing.T) {
	destination := &customerDestinationStub{}
	upserter, err := NewUpserter(
		&contactsServiceStub{createResult: &contactsdomain.Contact{ID: "contact-1", FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}},
		&contactLinkRepositoryStub{existing: &shopifyport.SyncLink{ShopifyID: "123", MannaiahID: "contact-1"}},
		destination,
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	if _, err := upserter.UpsertContact(context.Background(), shopifyport.ContactSyncCommand{
		ShopifyID: "123",
		Email:     "ada@example.com",
		FirstName: "Ada",
		LastName:  "Lovelace",
	}); err != nil {
		t.Fatalf("UpsertContact() error = %v", err)
	}
	if destination.tagsCalls != 1 {
		t.Fatalf("tag write-back calls = %d, want 1", destination.tagsCalls)
	}
	if destination.noteCalls != 0 {
		t.Fatalf("note write-back calls = %d, want 0", destination.noteCalls)
	}
}

// TestContactUpserterWriteBackFailureIsWarnOnly verifies Shopify write-back failures do not fail sync.
func TestContactUpserterWriteBackFailureIsWarnOnly(t *testing.T) {
	upserter, err := NewUpserter(
		&contactsServiceStub{createResult: &contactsdomain.Contact{ID: "contact-1", FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}},
		&contactLinkRepositoryStub{},
		&customerDestinationStub{err: errors.New("shopify unavailable")},
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	if _, err := upserter.UpsertContact(context.Background(), shopifyport.ContactSyncCommand{
		ShopifyID: "123",
		Email:     "ada@example.com",
		FirstName: "Ada",
		LastName:  "Lovelace",
	}); err != nil {
		t.Fatalf("UpsertContact() error = %v", err)
	}
}
