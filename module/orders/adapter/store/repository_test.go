package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
	coredb "mannaiah/module/core/database"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

// TestNewRepositoryRejectsNilDB verifies constructor validation for nil DB dependencies.
func TestNewRepositoryRejectsNilDB(t *testing.T) {
	if _, err := NewRepository(nil); !errors.Is(err, ErrNilDB) {
		t.Fatalf("NewRepository() error = %v, want ErrNilDB", err)
	}
}

// TestRepositoryCreateGetAppendStatusAndList verifies core repository order lifecycle behavior.
func TestRepositoryCreateGetAppendStatusAndList(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	entity := &ordersdomain.Order{
		Identifier: "woo-100",
		Realm:      "woocommerce",
		ContactID:  "contact-1",
		Items: []ordersdomain.Item{
			{
				SKU:              "SKU-1",
				Quantity:         2,
				Value:            25000,
				ProductID:        "product-1",
				ResolutionSource: ordersdomain.ItemResolutionSourceSKU,
			},
			{SKU: "SKU-2", AlternateName: "Fallback", Quantity: 1, Value: 10000, ResolutionSource: ordersdomain.ItemResolutionSourceAlternateName},
		},
		CurrentStatus: ordersdomain.StatusCreated,
		StatusHistory: []ordersdomain.StatusEntry{
			{Status: ordersdomain.StatusCreated, Author: "system", Description: "created", OccurredAt: time.Now().UTC().Add(-2 * time.Minute)},
		},
		HasCustomShippingAddress: true,
		ShippingAddress: ordersdomain.ShippingAddress{
			Address:  "Street 1",
			Address2: "Apt 2",
			Phone:    "3000000",
			CityCode: "11001",
		},
		ShippingCharges: []ordersdomain.ShippingCharge{
			{MethodID: "flat_rate", MethodTitle: "Flat Rate", Price: 15000},
		},
		Metadata: map[string]string{
			"integration.source": "woocommerce",
		},
	}
	if err := repository.Create(ctx, entity); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if entity.ID == "" {
		t.Fatalf("expected generated order id")
	}

	stored, err := repository.GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.Identifier != "woo-100" {
		t.Fatalf("stored.Identifier = %q, want %q", stored.Identifier, "woo-100")
	}
	if len(stored.Items) != 2 {
		t.Fatalf("len(stored.Items) = %d, want %d", len(stored.Items), 2)
	}
	if !stored.HasCustomShippingAddress {
		t.Fatalf("expected custom shipping address")
	}
	if stored.ShippingAddress.CityCode != "11001" {
		t.Fatalf("stored.ShippingAddress.CityCode = %q, want %q", stored.ShippingAddress.CityCode, "11001")
	}
	if stored.Metadata["integration.source"] != "woocommerce" {
		t.Fatalf("stored.Metadata[integration.source] = %q, want %q", stored.Metadata["integration.source"], "woocommerce")
	}
	if stored.Items[0].Value != 25000 {
		t.Fatalf("stored.Items[0].Value = %v, want %v", stored.Items[0].Value, 25000.0)
	}
	if len(stored.ShippingCharges) != 1 || stored.ShippingCharges[0].MethodID != "flat_rate" {
		t.Fatalf("stored.ShippingCharges = %+v, want one flat_rate row", stored.ShippingCharges)
	}

	nextStatus := ordersdomain.StatusEntry{
		Status:      ordersdomain.StatusPending,
		Author:      "user-1",
		Description: "pending review",
		OccurredAt:  time.Now().UTC(),
	}
	updated, err := repository.AppendStatus(ctx, entity.ID, nextStatus)
	if err != nil {
		t.Fatalf("AppendStatus() error = %v", err)
	}
	if updated.CurrentStatus != ordersdomain.StatusPending {
		t.Fatalf("updated.CurrentStatus = %q, want %q", updated.CurrentStatus, ordersdomain.StatusPending)
	}
	if len(updated.StatusHistory) != 2 {
		t.Fatalf("len(updated.StatusHistory) = %d, want %d", len(updated.StatusHistory), 2)
	}

	commentAt := time.Now().UTC().Add(time.Minute)
	updated, err = repository.AppendComment(ctx, entity.ID, ordersdomain.Comment{
		Author:     "agent-1",
		Comment:    "reviewed with customer",
		Internal:   true,
		OccurredAt: commentAt,
	})
	if err != nil {
		t.Fatalf("AppendComment() error = %v", err)
	}
	if len(updated.Comments) != 1 {
		t.Fatalf("len(updated.Comments) = %d, want 1", len(updated.Comments))
	}
	if updated.Comments[0].Author != "agent-1" || updated.Comments[0].Comment != "reviewed with customer" || !updated.Comments[0].Internal {
		t.Fatalf("updated.Comments[0] = %+v, want author/comment/internal values", updated.Comments[0])
	}
	if !updated.Comments[0].OccurredAt.UTC().Equal(commentAt) {
		t.Fatalf("updated.Comments[0].OccurredAt = %v, want %v", updated.Comments[0].OccurredAt, commentAt)
	}

	rows, total, err := repository.List(ctx, ordersport.ListQuery{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("List() result = total:%d len:%d, want total:1 len:1", total, len(rows))
	}

	entity.Items = []ordersdomain.Item{
		{SKU: "SKU-3", Quantity: 3, Value: 30000, ResolutionSource: ordersdomain.ItemResolutionSourceSKU},
	}
	entity.ShippingAddress = ordersdomain.ShippingAddress{
		Address:  "Street Updated",
		Address2: "Apt Updated",
		Phone:    "3111111",
		CityCode: "05001",
	}
	entity.ShippingCharges = []ordersdomain.ShippingCharge{
		{MethodID: "pickup", MethodTitle: "Pickup", Price: 0},
	}
	if err := repository.Update(ctx, entity); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	updatedEntity, err := repository.GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID(after update) error = %v", err)
	}
	if len(updatedEntity.Items) != 1 || updatedEntity.Items[0].SKU != "SKU-3" {
		t.Fatalf("updatedEntity.Items = %+v, want one SKU-3 item", updatedEntity.Items)
	}
	if updatedEntity.ShippingAddress.CityCode != "05001" {
		t.Fatalf("updatedEntity.ShippingAddress.CityCode = %q, want %q", updatedEntity.ShippingAddress.CityCode, "05001")
	}
}

// TestRepositoryCreateRejectsDuplicateIdentifier verifies realm+identifier uniqueness behavior.
func TestRepositoryCreateRejectsDuplicateIdentifier(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	first := newOrderForTest("order-1", "woo", "contact-1")
	if err := repository.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	second := newOrderForTest("order-1", "woo", "contact-2")
	if err := repository.Create(ctx, second); !errors.Is(err, ordersport.ErrDuplicateIdentifier) {
		t.Fatalf("Create(second) error = %v, want ErrDuplicateIdentifier", err)
	}
}

// TestRepositoryNotFoundPaths verifies missing-row retrieval and update behavior.
func TestRepositoryNotFoundPaths(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	if _, err := repository.GetByID(ctx, "missing"); !errors.Is(err, ordersport.ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
	if err := repository.Update(ctx, &ordersdomain.Order{ID: "missing", Identifier: "id", Realm: "realm", ContactID: "contact", Items: []ordersdomain.Item{{SKU: "SKU", Quantity: 1}}, CurrentStatus: ordersdomain.StatusCreated, StatusHistory: []ordersdomain.StatusEntry{{Status: ordersdomain.StatusCreated, Author: "system", OccurredAt: time.Now().UTC()}}}); !errors.Is(err, ordersport.ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
	if _, err := repository.AppendStatus(ctx, "missing", ordersdomain.StatusEntry{
		Status:     ordersdomain.StatusCreated,
		Author:     "system",
		OccurredAt: time.Now().UTC(),
	}); !errors.Is(err, ordersport.ErrNotFound) {
		t.Fatalf("AppendStatus() error = %v, want ErrNotFound", err)
	}
	if _, err := repository.AppendComment(ctx, "missing", ordersdomain.Comment{
		Author:     "agent-1",
		Comment:    "missing",
		OccurredAt: time.Now().UTC(),
	}); !errors.Is(err, ordersport.ErrNotFound) {
		t.Fatalf("AppendComment() error = %v, want ErrNotFound", err)
	}
}

// TestRepositoryListFilters verifies list filtering by realm, contact, identifier, and status.
func TestRepositoryListFilters(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	seed := []*ordersdomain.Order{
		newOrderForTest("identifier-a", "woocommerce", "contact-a"),
		newOrderForTest("identifier-b", "website", "contact-b"),
	}
	seed[1].CurrentStatus = ordersdomain.StatusCompleted
	seed[1].StatusHistory = []ordersdomain.StatusEntry{
		{Status: ordersdomain.StatusCreated, Author: "system", OccurredAt: time.Now().UTC().Add(-time.Minute)},
		{Status: ordersdomain.StatusCompleted, Author: "system", OccurredAt: time.Now().UTC()},
	}
	for _, row := range seed {
		if err := repository.Create(ctx, row); err != nil {
			t.Fatalf("Create(seed) error = %v", err)
		}
	}

	rows, total, err := repository.List(ctx, ordersport.ListQuery{Realm: "woocommerce"})
	if err != nil {
		t.Fatalf("List(realm) error = %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Realm != "woocommerce" {
		t.Fatalf("unexpected realm result: total=%d len=%d rows=%+v", total, len(rows), rows)
	}

	rows, total, err = repository.List(ctx, ordersport.ListQuery{ContactID: "contact-b"})
	if err != nil {
		t.Fatalf("List(contact) error = %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ContactID != "contact-b" {
		t.Fatalf("unexpected contact result: total=%d len=%d rows=%+v", total, len(rows), rows)
	}

	rows, total, err = repository.List(ctx, ordersport.ListQuery{Identifier: "identifier-b"})
	if err != nil {
		t.Fatalf("List(identifier) error = %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Identifier != "identifier-b" {
		t.Fatalf("unexpected identifier result: total=%d len=%d rows=%+v", total, len(rows), rows)
	}

	rows, total, err = repository.List(ctx, ordersport.ListQuery{Status: ordersdomain.StatusCompleted})
	if err != nil {
		t.Fatalf("List(status) error = %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].CurrentStatus != ordersdomain.StatusCompleted {
		t.Fatalf("unexpected status result: total=%d len=%d rows=%+v", total, len(rows), rows)
	}
}

// TestRepositoryCurrentStatusUsesLatestOccurredAt verifies current-status resolution from latest occurred-at values.
func TestRepositoryCurrentStatusUsesLatestOccurredAt(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	latest := time.Now().UTC()
	older := latest.Add(-2 * time.Hour)
	entity := newOrderForTest("identifier-status-time", "woocommerce", "contact-status-time")
	entity.StatusHistory = []ordersdomain.StatusEntry{
		{Status: ordersdomain.StatusCompleted, Author: "system", OccurredAt: latest},
		{Status: ordersdomain.StatusCreated, Author: "system", OccurredAt: older},
	}
	entity.CurrentStatus = ordersdomain.StatusCreated
	if err := repository.Create(ctx, entity); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stored, err := repository.GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.CurrentStatus != ordersdomain.StatusCompleted {
		t.Fatalf("stored.CurrentStatus = %q, want %q", stored.CurrentStatus, ordersdomain.StatusCompleted)
	}

	rows, total, err := repository.List(ctx, ordersport.ListQuery{Status: ordersdomain.StatusCompleted})
	if err != nil {
		t.Fatalf("List(StatusCompleted) error = %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("List(StatusCompleted) result = total:%d len:%d, want total:1 len:1", total, len(rows))
	}

	rows, total, err = repository.List(ctx, ordersport.ListQuery{Status: ordersdomain.StatusCreated})
	if err != nil {
		t.Fatalf("List(StatusCreated) error = %v", err)
	}
	if total != 0 || len(rows) != 0 {
		t.Fatalf("List(StatusCreated) result = total:%d len:%d, want total:0 len:0", total, len(rows))
	}
}

// TestRepositoryCreateWithoutCustomShipping verifies optional shipping-row behavior.
func TestRepositoryCreateWithoutCustomShipping(t *testing.T) {
	repository := newRepositoryForTest(t)
	ctx := context.Background()

	entity := newOrderForTest("identifier-no-shipping", "website", "contact-c")
	entity.HasCustomShippingAddress = false
	entity.ShippingAddress = ordersdomain.ShippingAddress{
		Address:  "Ignored",
		Address2: "Ignored",
		Phone:    "Ignored",
		CityCode: "Ignored",
	}
	if err := repository.Create(ctx, entity); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stored, err := repository.GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.HasCustomShippingAddress {
		t.Fatalf("stored.HasCustomShippingAddress = true, want false")
	}
}

// TestRepositoryErrorPathsOnClosedDB verifies wrapped DB errors after SQL handle closure.
func TestRepositoryErrorPathsOnClosedDB(t *testing.T) {
	db := newDBForTest(t)
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := repository.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("sqlDB.Close() error = %v", err)
	}

	if err := repository.EnsureSchema(context.Background()); err == nil {
		t.Fatalf("expected EnsureSchema() error on closed db")
	}
	if err := repository.Create(context.Background(), newOrderForTest("x", "y", "c")); err == nil {
		t.Fatalf("expected Create() error on closed db")
	}
	if err := repository.Update(context.Background(), newOrderForTest("x", "y", "c")); err == nil {
		t.Fatalf("expected Update() error on closed db")
	}
	if _, err := repository.GetByID(context.Background(), "x"); err == nil {
		t.Fatalf("expected GetByID() error on closed db")
	}
	if _, _, err := repository.List(context.Background(), ordersport.ListQuery{}); err == nil {
		t.Fatalf("expected List() error on closed db")
	}
	if _, err := repository.AppendStatus(context.Background(), "x", ordersdomain.StatusEntry{
		Status:     ordersdomain.StatusCreated,
		Author:     "system",
		OccurredAt: time.Now().UTC(),
	}); err == nil {
		t.Fatalf("expected AppendStatus() error on closed db")
	}
	if _, err := repository.AppendComment(context.Background(), "x", ordersdomain.Comment{
		Author:     "system",
		Comment:    "closed db",
		OccurredAt: time.Now().UTC(),
	}); err == nil {
		t.Fatalf("expected AppendComment() error on closed db")
	}
}

// TestHelpers verifies helper behavior coverage for normalization and duplicate detection.
func TestHelpers(t *testing.T) {
	values := normalizeOrderIDs([]string{" a ", "b", "a", "", " b "})
	if len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("normalizeOrderIDs() = %#v, want %#v", values, []string{"a", "b"})
	}

	duplicateMessage := errors.New("UNIQUE constraint failed: orders.realm, orders.identifier")
	if mapped := mapDuplicateError(duplicateMessage); !errors.Is(mapped, ordersport.ErrDuplicateIdentifier) {
		t.Fatalf("mapDuplicateError() = %v, want ErrDuplicateIdentifier", mapped)
	}
	if mapped := mapDuplicateError(errors.New("timeout")); mapped != nil {
		t.Fatalf("mapDuplicateError() = %v, want nil", mapped)
	}
}

// newOrderForTest creates valid test order values.
func newOrderForTest(identifier string, realm string, contactID string) *ordersdomain.Order {
	return &ordersdomain.Order{
		Identifier:    identifier,
		Realm:         realm,
		ContactID:     contactID,
		Items:         []ordersdomain.Item{{SKU: "SKU-1", Quantity: 1, Value: 10000, ResolutionSource: ordersdomain.ItemResolutionSourceUnresolved}},
		CurrentStatus: ordersdomain.StatusCreated,
		StatusHistory: []ordersdomain.StatusEntry{
			{Status: ordersdomain.StatusCreated, Author: "system", OccurredAt: time.Now().UTC()},
		},
	}
}

// newRepositoryForTest creates schema-ready repositories for tests.
func newRepositoryForTest(t *testing.T) *Repository {
	t.Helper()

	db := newDBForTest(t)
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := repository.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	return repository
}

// newDBForTest creates an in-memory sqlite DB for repository tests.
func newDBForTest(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared", MaxOpenConns: 1}, nil)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}
