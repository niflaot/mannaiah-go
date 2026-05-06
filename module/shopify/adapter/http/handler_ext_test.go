package http

import (
	"context"
	"testing"
	"time"

	contactsapplication "mannaiah/module/contacts/application"
	contactsdomain "mannaiah/module/contacts/domain"
	contactsport "mannaiah/module/contacts/port"
	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	shopifyport "mannaiah/module/shopify/port"
)

type extensionTestSyncLinkRepository struct {
	link          *shopifyport.SyncLink
	err           error
	requestedKind shopifyport.SyncKind
	requestedShop string
	requestedID   string
}

// GetLinkByShopifyID returns the configured sync link for extension-route tests.
func (r *extensionTestSyncLinkRepository) GetLinkByShopifyID(ctx context.Context, kind shopifyport.SyncKind, shopDomain string, shopifyID string) (*shopifyport.SyncLink, error) {
	r.requestedKind = kind
	r.requestedShop = shopDomain
	r.requestedID = shopifyID
	return r.link, r.err
}

// GetLinkByMannaiahID satisfies the sync-link repository interface for tests.
func (r *extensionTestSyncLinkRepository) GetLinkByMannaiahID(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string) (*shopifyport.SyncLink, error) {
	return nil, r.err
}

// UpsertLink satisfies the sync-link repository interface for tests.
func (r *extensionTestSyncLinkRepository) UpsertLink(ctx context.Context, input shopifyport.UpsertSyncLinkInput) (*shopifyport.SyncLink, error) {
	return r.link, r.err
}

// UpdateLastKnownStatus satisfies the sync-link repository interface for tests.
func (r *extensionTestSyncLinkRepository) UpdateLastKnownStatus(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string, status string) error {
	return r.err
}

type extensionTestContactsLookup struct {
	contact      *contactsdomain.Contact
	err          error
	requestedID  string
	listResult   *contactsapplication.ListResult
	listError    error
	updateError  error
	deleteError  error
	createError  error
	updateResult *contactsdomain.Contact
	createResult *contactsdomain.Contact
}

// Create satisfies the contacts application service interface for extension-route tests.
func (s *extensionTestContactsLookup) Create(ctx context.Context, command contactsapplication.CreateCommand) (*contactsdomain.Contact, error) {
	return s.createResult, s.createError
}

// Get returns the configured contact for extension-route tests.
func (s *extensionTestContactsLookup) Get(ctx context.Context, id string) (*contactsdomain.Contact, error) {
	s.requestedID = id
	return s.contact, s.err
}

// List satisfies the contacts application service interface for extension-route tests.
func (s *extensionTestContactsLookup) List(ctx context.Context, query contactsport.ListQuery) (*contactsapplication.ListResult, error) {
	return s.listResult, s.listError
}

// Update satisfies the contacts application service interface for extension-route tests.
func (s *extensionTestContactsLookup) Update(ctx context.Context, id string, command contactsapplication.UpdateCommand) (*contactsdomain.Contact, error) {
	return s.updateResult, s.updateError
}

// Delete satisfies the contacts application service interface for extension-route tests.
func (s *extensionTestContactsLookup) Delete(ctx context.Context, id string) error {
	return s.deleteError
}

type extensionTestOrdersLookup struct {
	order          *ordersdomain.Order
	err            error
	requestedID    string
	listResult     *ordersapplication.ListResult
	listError      error
	createError    error
	updateError    error
	statusError    error
	commentError   error
	deleteError    error
	createResult   *ordersdomain.Order
	updateResult   *ordersdomain.Order
	statusResult   *ordersdomain.Order
	commentResult  *ordersdomain.Order
	commentList    []ordersdomain.Comment
	commentListErr error
}

// Create satisfies the orders application service interface for extension-route tests.
func (s *extensionTestOrdersLookup) Create(ctx context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error) {
	return s.createResult, s.createError
}

// Update satisfies the orders application service interface for extension-route tests.
func (s *extensionTestOrdersLookup) Update(ctx context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error) {
	return s.updateResult, s.updateError
}

// Get returns the configured order for extension-route tests.
func (s *extensionTestOrdersLookup) Get(ctx context.Context, id string) (*ordersdomain.Order, error) {
	s.requestedID = id
	return s.order, s.err
}

// List satisfies the orders application service interface for extension-route tests.
func (s *extensionTestOrdersLookup) List(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
	return s.listResult, s.listError
}

// UpdateStatus satisfies the orders application service interface for extension-route tests.
func (s *extensionTestOrdersLookup) UpdateStatus(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
	return s.statusResult, s.statusError
}

// AddComment satisfies the orders application service interface for extension-route tests.
func (s *extensionTestOrdersLookup) AddComment(ctx context.Context, id string, command ordersapplication.AddCommentCommand) (*ordersdomain.Order, error) {
	return s.commentResult, s.commentError
}

// UpdateComment satisfies the orders application service interface for extension-route tests.
func (s *extensionTestOrdersLookup) UpdateComment(ctx context.Context, id string, commentID string, command ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error) {
	return s.commentResult, s.commentError
}

// DeleteComment satisfies the orders application service interface for extension-route tests.
func (s *extensionTestOrdersLookup) DeleteComment(ctx context.Context, id string, commentID string, command ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error) {
	return s.commentResult, s.commentError
}

// ListComments satisfies the orders application service interface for extension-route tests.
func (s *extensionTestOrdersLookup) ListComments(ctx context.Context, id string) ([]ordersdomain.Comment, error) {
	return s.commentList, s.commentListErr
}

// Delete satisfies the orders application service interface for extension-route tests.
func (s *extensionTestOrdersLookup) Delete(ctx context.Context, id string) error {
	return s.deleteError
}

// TestGetExtensionOrderIncludesCreatedAt verifies extension order summaries expose the mainstream order creation timestamp.
func TestGetExtensionOrderIncludesCreatedAt(t *testing.T) {
	createdAt := time.Date(2026, time.May, 6, 12, 30, 0, 0, time.UTC)
	lastSyncedAt := createdAt.Add(5 * time.Minute)
	links := &extensionTestSyncLinkRepository{
		link: &shopifyport.SyncLink{
			Kind:         shopifyport.SyncKindOrder,
			ShopDomain:   "2axh5c-b1.myshopify.com",
			ShopifyID:    "1234567890",
			MannaiahID:   "order-123",
			LastSyncedAt: &lastSyncedAt,
		},
	}
	handler := &Handler{
		links:          links,
		ordersLookup:   &extensionTestOrdersLookup{order: &ordersdomain.Order{ID: "order-123", CurrentStatus: ordersdomain.StatusCompleted, CreatedAt: createdAt}},
		contactsLookup: &extensionTestContactsLookup{},
	}
	requestContext := &launchTestContext{
		headers: map[string]string{},
		pathParams: map[string]string{
			"shopifyOrderId": "1234567890",
		},
		locals: map[string]any{
			extensionShopDomainLocal: "2axh5c-b1.myshopify.com",
		},
	}

	if err := handler.getExtensionOrder(requestContext); err != nil {
		t.Fatalf("getExtensionOrder() error = %v", err)
	}
	summary, ok := requestContext.jsonBody.(ExtensionOrderSummary)
	if !ok {
		t.Fatalf("getExtensionOrder() json body type = %T, want ExtensionOrderSummary", requestContext.jsonBody)
	}
	if !summary.Linked {
		t.Fatalf("getExtensionOrder() linked = false, want true")
	}
	if summary.CreatedAt == nil || !summary.CreatedAt.Equal(createdAt) {
		t.Fatalf("getExtensionOrder() createdAt = %v, want %v", summary.CreatedAt, createdAt)
	}
	if summary.LastSyncedAt == nil || !summary.LastSyncedAt.Equal(lastSyncedAt) {
		t.Fatalf("getExtensionOrder() lastSyncedAt = %v, want %v", summary.LastSyncedAt, lastSyncedAt)
	}
	if links.requestedShop != "2axh5c-b1.myshopify.com" {
		t.Fatalf("getExtensionOrder() requested shop = %q, want %q", links.requestedShop, "2axh5c-b1.myshopify.com")
	}
}

// TestGetExtensionContactIncludesCreatedAt verifies extension contact summaries expose the mainstream contact creation timestamp.
func TestGetExtensionContactIncludesCreatedAt(t *testing.T) {
	createdAt := time.Date(2026, time.May, 3, 9, 0, 0, 0, time.UTC)
	lastSyncedAt := createdAt.Add(30 * time.Minute)
	links := &extensionTestSyncLinkRepository{
		link: &shopifyport.SyncLink{
			Kind:         shopifyport.SyncKindContact,
			ShopDomain:   "2axh5c-b1.myshopify.com",
			ShopifyID:    "9988776655",
			MannaiahID:   "contact-123",
			LastSyncedAt: &lastSyncedAt,
		},
	}
	handler := &Handler{
		links: links,
		contactsLookup: &extensionTestContactsLookup{
			contact: &contactsdomain.Contact{
				ID:        "contact-123",
				LegalName: "Juan Perez",
				CreatedAt: createdAt,
			},
		},
		ordersLookup: &extensionTestOrdersLookup{},
	}
	requestContext := &launchTestContext{
		headers: map[string]string{},
		pathParams: map[string]string{
			"shopifyCustomerId": "9988776655",
		},
		locals: map[string]any{
			extensionShopDomainLocal: "2axh5c-b1.myshopify.com",
		},
	}

	if err := handler.getExtensionContact(requestContext); err != nil {
		t.Fatalf("getExtensionContact() error = %v", err)
	}
	summary, ok := requestContext.jsonBody.(ExtensionContactSummary)
	if !ok {
		t.Fatalf("getExtensionContact() json body type = %T, want ExtensionContactSummary", requestContext.jsonBody)
	}
	if !summary.Linked {
		t.Fatalf("getExtensionContact() linked = false, want true")
	}
	if summary.CreatedAt == nil || !summary.CreatedAt.Equal(createdAt) {
		t.Fatalf("getExtensionContact() createdAt = %v, want %v", summary.CreatedAt, createdAt)
	}
	if summary.LastSyncedAt == nil || !summary.LastSyncedAt.Equal(lastSyncedAt) {
		t.Fatalf("getExtensionContact() lastSyncedAt = %v, want %v", summary.LastSyncedAt, lastSyncedAt)
	}
	if links.requestedShop != "2axh5c-b1.myshopify.com" {
		t.Fatalf("getExtensionContact() requested shop = %q, want %q", links.requestedShop, "2axh5c-b1.myshopify.com")
	}
}