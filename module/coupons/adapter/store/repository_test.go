package store

import (
	"context"
	"strings"
	"testing"
	"time"

	coredb "mannaiah/module/core/database"
	"mannaiah/module/coupons/domain"
	"mannaiah/module/coupons/port"

	"gorm.io/gorm"
)

// couponSearchContactRecord defines the minimal contacts schema needed by coupon search tests.
type couponSearchContactRecord struct {
	// ID defines contact identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// LegalName defines legal contact names.
	LegalName string `gorm:"size:255"`
	// FirstName defines personal first names.
	FirstName string `gorm:"size:255"`
	// LastName defines personal last names.
	LastName string `gorm:"size:255"`
	// Email defines contact email values.
	Email string `gorm:"size:255;not null"`
	// DeletedAt defines soft-delete timestamps.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName defines the contacts table name used by coupon search joins.
func (couponSearchContactRecord) TableName() string {
	return "contacts"
}

// TestRepositorySearchMatchesAssignedContactName verifies unified term search across linked contact names.
func TestRepositorySearchMatchesAssignedContactName(t *testing.T) {
	repository := newRepositoryForTest(t)

	seedContactForTest(t, repository.db, couponSearchContactRecord{
		ID:        "contact-1",
		FirstName: "Ian",
		LastName:  "Fedev",
		Email:     "ian@example.com",
	})

	if err := repository.Create(context.Background(), newCouponForTest("coupon-1", "WELCOME10", domain.DiscountTypeFixed, []string{"contact-1"})); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	rows, total, err := repository.Search(context.Background(), port.SearchQuery{
		Term:     "ian fedev",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want %d", total, 1)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want %d", len(rows), 1)
	}
	if rows[0].ID != "coupon-1" {
		t.Fatalf("rows[0].ID = %q, want %q", rows[0].ID, "coupon-1")
	}
}

// TestRepositorySearchFiltersByDiscountType verifies exact discount-type filtering with unified pagination.
func TestRepositorySearchFiltersByDiscountType(t *testing.T) {
	repository := newRepositoryForTest(t)

	if err := repository.Create(context.Background(), newCouponForTest("coupon-fixed", "FIXED10", domain.DiscountTypeFixed, nil)); err != nil {
		t.Fatalf("Create(fixed) error = %v", err)
	}
	if err := repository.Create(context.Background(), newCouponForTest("coupon-percentage", "PERCENT10", domain.DiscountTypePercentage, nil)); err != nil {
		t.Fatalf("Create(percentage) error = %v", err)
	}

	rows, total, err := repository.Search(context.Background(), port.SearchQuery{
		DiscountType: string(domain.DiscountTypeFixed),
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want %d", total, 1)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want %d", len(rows), 1)
	}
	if rows[0].DiscountType != domain.DiscountTypeFixed {
		t.Fatalf("rows[0].DiscountType = %q, want %q", rows[0].DiscountType, domain.DiscountTypeFixed)
	}
}

// TestRepositoryCreatePersistsWooCommerceID verifies WooCommerce identifier persistence and lookup behavior.
func TestRepositoryCreatePersistsWooCommerceID(t *testing.T) {
	repository := newRepositoryForTest(t)
	wooCommerceID := 1024365

	coupon := newCouponForTest("coupon-woo", "WOO10", domain.DiscountTypePercentage, nil)
	coupon.WooCommerceID = &wooCommerceID

	if err := repository.Create(context.Background(), coupon); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stored, err := repository.GetByWooCommerceID(context.Background(), wooCommerceID)
	if err != nil {
		t.Fatalf("GetByWooCommerceID() error = %v", err)
	}
	if stored == nil {
		t.Fatalf("GetByWooCommerceID() = nil, want coupon")
	}
	if stored.WooCommerceID == nil || *stored.WooCommerceID != wooCommerceID {
		t.Fatalf("stored.WooCommerceID = %v, want %d", stored.WooCommerceID, wooCommerceID)
	}
}

// newCouponForTest builds a minimal coupon aggregate for repository tests.
func newCouponForTest(id string, code string, discountType domain.DiscountType, assignedContactIDs []string) *domain.Coupon {
	now := time.Now().UTC()

	return &domain.Coupon{
		ID:                 id,
		Code:               code,
		Origin:             "manual",
		DiscountType:       discountType,
		DiscountAmount:     10,
		Active:             true,
		AssignedContactIDs: assignedContactIDs,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// seedContactForTest inserts a linked contact row for coupon term search tests.
func seedContactForTest(t *testing.T, db *gorm.DB, contact couponSearchContactRecord) {
	t.Helper()

	if err := db.WithContext(context.Background()).Create(&contact).Error; err != nil {
		t.Fatalf("Create(contact) error = %v", err)
	}
}

// newRepositoryForTest creates a schema-ready coupon repository for tests.
func newRepositoryForTest(t *testing.T) *Repository {
	t.Helper()

	db := newDBForTest(t)
	if err := ensureSchemaForTest(db); err != nil {
		t.Fatalf("ensureSchemaForTest() error = %v", err)
	}

	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	return repository
}

// ensureSchemaForTest creates the minimal schema required by coupon repository tests.
func ensureSchemaForTest(db *gorm.DB) error {
	statements := []string{
		`CREATE TABLE contacts (
			id TEXT PRIMARY KEY,
			legal_name TEXT,
			first_name TEXT,
			last_name TEXT,
			email TEXT NOT NULL,
			deleted_at DATETIME
		)`,
		`CREATE TABLE coupons (
			id TEXT PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			origin TEXT NOT NULL DEFAULT '',
			discount_type TEXT NOT NULL,
			discount_amount REAL NOT NULL DEFAULT 0,
			max_usages_global INTEGER,
			max_usages_per_email INTEGER,
			active BOOLEAN NOT NULL DEFAULT 1,
			expires_at DATETIME,
			woocommerce_id INTEGER,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME
		)`,
		`CREATE TABLE coupon_assigned_emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			coupon_id TEXT NOT NULL,
			email TEXT NOT NULL,
			UNIQUE(coupon_id, email)
		)`,
		`CREATE TABLE coupon_assigned_contact_ids (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			coupon_id TEXT NOT NULL,
			contact_id TEXT NOT NULL,
			UNIQUE(coupon_id, contact_id)
		)`,
		`CREATE TABLE coupon_included_product_ids (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			coupon_id TEXT NOT NULL,
			product_id TEXT NOT NULL,
			UNIQUE(coupon_id, product_id)
		)`,
		`CREATE TABLE coupon_included_category_ids (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			coupon_id TEXT NOT NULL,
			category_id TEXT NOT NULL,
			UNIQUE(coupon_id, category_id)
		)`,
		`CREATE TABLE coupon_included_tag_ids (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			coupon_id TEXT NOT NULL,
			tag_id TEXT NOT NULL,
			UNIQUE(coupon_id, tag_id)
		)`,
		`CREATE TABLE coupon_usages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			coupon_id TEXT NOT NULL,
			order_id TEXT NOT NULL,
			email TEXT NOT NULL,
			used_at DATETIME NOT NULL,
			UNIQUE(coupon_id, order_id)
		)`,
	}

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}

	return nil
}

// newDBForTest creates an in-memory sqlite DB for coupon repository tests.
func newDBForTest(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := coredb.Open(coredb.Config{Driver: "sqlite", DSN: dsn}, nil)
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
