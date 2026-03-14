package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"mannaiah/module/contacts/domain"
	"mannaiah/module/contacts/port"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("contacts db must not be nil")
)

// Repository implements contact persistence using GORM.
type Repository struct {
	// db is the underlying GORM handle.
	db *gorm.DB
}

// contactRecord defines contact persistence schema.
type contactRecord struct {
	// ID is the primary key identifier.
	ID string `gorm:"primaryKey;size:64"`
	// DocumentType defines document category.
	DocumentType string `gorm:"index;size:16"`
	// DocumentNumber defines document number value.
	DocumentNumber string `gorm:"index;size:128"`
	// DocumentKey defines a normalized document identity key used for uniqueness checks.
	DocumentKey *string `gorm:"uniqueIndex:idx_contacts_document_key;size:191"`
	// LegalName defines legal contact names.
	LegalName string `gorm:"size:255"`
	// FirstName defines personal first names.
	FirstName string `gorm:"size:255"`
	// LastName defines personal last names.
	LastName string `gorm:"size:255"`
	// Email defines contact email values.
	Email string `gorm:"uniqueIndex:idx_contacts_email;size:255;not null"`
	// Phone defines contact phone values.
	Phone string `gorm:"size:64"`
	// Address defines physical addresses.
	Address string `gorm:"size:512"`
	// AddressExtra defines optional address details.
	AddressExtra string `gorm:"size:512"`
	// CityCode defines city code values.
	CityCode string `gorm:"size:64"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
	// DeletedAt defines soft-delete timestamps.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName defines storage table name.
func (contactRecord) TableName() string {
	return "contacts"
}

var (
	// _ ensures Repository satisfies contact repository contracts.
	_ port.Repository = (*Repository)(nil)
)

// NewRepository creates a contact repository over GORM.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// EnsureSchema is a no-op because schema evolution is managed by SQL migrations.
func (r *Repository) EnsureSchema(ctx context.Context) error {
	_ = ctx

	return nil
}

// Create persists a new contact entity.
func (r *Repository) Create(ctx context.Context, contact *domain.Contact) error {
	record := toRecord(*contact)
	if strings.TrimSpace(record.ID) == "" {
		record.ID = generateID()
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return wrapWriteError("create", err)
		}
		if err := replaceContactMetadata(tx, record.ID, contact.Metadata); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	latest, err := r.GetByID(ctx, record.ID)
	if err != nil {
		return err
	}
	*contact = *latest

	return nil
}

// GetByID retrieves a contact entity by id.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Contact, error) {
	trimmedID := strings.TrimSpace(id)

	var record contactRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", trimmedID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, port.ErrNotFound
		}
		return nil, fmt.Errorf("get contact record: %w", err)
	}

	metadataMap, err := loadMetadataByContactIDs(ctx, r.db, []string{trimmedID})
	if err != nil {
		return nil, err
	}

	entity := toDomain(record, metadataMap[trimmedID])
	return &entity, nil
}

// List retrieves contacts and total count using query-side filters.
func (r *Repository) List(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) {
	resolvedPage, resolvedLimit := normalizePagination(query.Page, query.Limit)

	base := r.db.WithContext(ctx).Model(&contactRecord{})
	base = applyListQuery(base, query)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count contact records: %w", err)
	}

	orderBy, orderDir := normalizeOrder(query.OrderBy, query.OrderDir)
	var records []contactRecord
	if err := base.Order(fmt.Sprintf("%s %s", orderBy, orderDir)).Offset((resolvedPage - 1) * resolvedLimit).Limit(resolvedLimit).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list contact records: %w", err)
	}

	contactIDs := make([]string, 0, len(records))
	for _, record := range records {
		contactIDs = append(contactIDs, record.ID)
	}
	metadataMap, err := loadMetadataByContactIDs(ctx, r.db, contactIDs)
	if err != nil {
		return nil, 0, err
	}

	result := make([]domain.Contact, 0, len(records))
	for _, record := range records {
		result = append(result, toDomain(record, metadataMap[record.ID]))
	}

	return result, total, nil
}

// Update persists modifications for an existing contact.
func (r *Repository) Update(ctx context.Context, contact *domain.Contact) error {
	if strings.TrimSpace(contact.ID) == "" {
		return port.ErrNotFound
	}

	record := toRecord(*contact)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updateTx := tx.Model(&contactRecord{}).Where("id = ?", record.ID).Updates(map[string]any{
			"document_type":   record.DocumentType,
			"document_number": record.DocumentNumber,
			"document_key":    buildDocumentKey(record.DocumentType, record.DocumentNumber),
			"legal_name":      record.LegalName,
			"first_name":      record.FirstName,
			"last_name":       record.LastName,
			"email":           record.Email,
			"phone":           record.Phone,
			"address":         record.Address,
			"address_extra":   record.AddressExtra,
			"city_code":       record.CityCode,
			"created_at":      record.CreatedAt,
		})
		if updateTx.Error != nil {
			return wrapWriteError("update", updateTx.Error)
		}
		if updateTx.RowsAffected == 0 {
			var existingRows int64
			existsTx := tx.Model(&contactRecord{}).Where("id = ?", record.ID).Count(&existingRows)
			if existsTx.Error != nil {
				return fmt.Errorf("check contact record existence: %w", existsTx.Error)
			}
			if existingRows == 0 {
				return port.ErrNotFound
			}
		}
		if err := replaceContactMetadata(tx, record.ID, contact.Metadata); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	latest, err := r.GetByID(ctx, record.ID)
	if err != nil {
		return err
	}
	*contact = *latest

	return nil
}

// Delete soft-deletes a contact by id.
func (r *Repository) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deleteTx := tx.Delete(&contactRecord{}, "id = ?", trimmedID)
		if deleteTx.Error != nil {
			return fmt.Errorf("delete contact record: %w", deleteTx.Error)
		}
		if deleteTx.RowsAffected == 0 {
			return port.ErrNotFound
		}
		if err := tx.Where("contact_id = ?", trimmedID).Delete(&contactMetadataRecord{}).Error; err != nil {
			return fmt.Errorf("delete contact metadata records: %w", err)
		}

		return nil
	})
}

// applyListQuery applies list filtering and exclusion options.
func applyListQuery(tx *gorm.DB, query port.ListQuery) *gorm.DB {
	next := tx
	if strings.TrimSpace(query.Email) != "" {
		next = next.Where("email = ?", strings.TrimSpace(query.Email))
	}
	trimmedDocumentType := strings.TrimSpace(query.DocumentType)
	trimmedDocumentNumber := strings.TrimSpace(query.DocumentNumber)
	if trimmedDocumentType != "" {
		next = next.Where("document_type = ?", trimmedDocumentType)
	}
	if trimmedDocumentNumber != "" {
		next = next.Where("document_number = ?", trimmedDocumentNumber)
	}
	trimmedMetadataKey := strings.TrimSpace(query.MetadataKey)
	trimmedMetadataValue := strings.TrimSpace(query.MetadataValue)
	if trimmedMetadataKey != "" || trimmedMetadataValue != "" {
		subQuery := tx.Session(&gorm.Session{NewDB: true}).Model(&contactMetadataRecord{}).Select("contact_id")
		if trimmedMetadataKey != "" {
			subQuery = subQuery.Where("`key` = ?", trimmedMetadataKey)
		}
		if trimmedMetadataValue != "" {
			subQuery = subQuery.Where("`value` = ?", trimmedMetadataValue)
		}
		next = next.Where("id IN (?)", subQuery)
	}
	if len(query.ExcludeIDs) > 0 {
		next = next.Where("id NOT IN ?", query.ExcludeIDs)
	}

	return next
}

// normalizePagination resolves list pagination defaults.
func normalizePagination(page int, limit int) (int, int) {
	resolvedPage := page
	if resolvedPage <= 0 {
		resolvedPage = 1
	}

	resolvedLimit := limit
	if resolvedLimit <= 0 {
		resolvedLimit = 10
	}

	return resolvedPage, resolvedLimit
}

// normalizeOrder resolves list order-by field and direction values.
func normalizeOrder(orderBy string, orderDir string) (string, string) {
	columns := map[string]string{
		"createdAt":      "created_at",
		"updatedAt":      "updated_at",
		"legalName":      "legal_name",
		"firstName":      "first_name",
		"lastName":       "last_name",
		"email":          "email",
		"documentType":   "document_type",
		"documentNumber": "document_number",
	}

	column, ok := columns[strings.TrimSpace(orderBy)]
	if !ok {
		column = "created_at"
	}

	direction := strings.ToLower(strings.TrimSpace(orderDir))
	if direction != "asc" {
		direction = "desc"
	}

	return column, direction
}

// toRecord maps domain contact entities to persistence records.
func toRecord(contact domain.Contact) contactRecord {
	normalized := contact
	normalized.Normalize()

	documentType := strings.TrimSpace(string(normalized.DocumentType))
	documentNumber := strings.TrimSpace(normalized.DocumentNumber)

	return contactRecord{
		ID:             strings.TrimSpace(normalized.ID),
		DocumentType:   documentType,
		DocumentNumber: documentNumber,
		DocumentKey:    buildDocumentKey(documentType, documentNumber),
		LegalName:      strings.TrimSpace(normalized.LegalName),
		FirstName:      strings.TrimSpace(normalized.FirstName),
		LastName:       strings.TrimSpace(normalized.LastName),
		Email:          strings.TrimSpace(normalized.Email),
		Phone:          strings.TrimSpace(normalized.Phone),
		Address:        strings.TrimSpace(normalized.Address),
		AddressExtra:   strings.TrimSpace(normalized.AddressExtra),
		CityCode:       strings.TrimSpace(normalized.CityCode),
		CreatedAt:      normalized.CreatedAt,
		UpdatedAt:      normalized.UpdatedAt,
	}
}

// toDomain maps persistence records to domain contact entities.
func toDomain(record contactRecord, metadata map[string]string) domain.Contact {
	entity := domain.Contact{
		ID:             record.ID,
		DocumentType:   domain.DocumentType(record.DocumentType),
		DocumentNumber: record.DocumentNumber,
		LegalName:      record.LegalName,
		FirstName:      record.FirstName,
		LastName:       record.LastName,
		Email:          record.Email,
		Phone:          record.Phone,
		Address:        record.Address,
		AddressExtra:   record.AddressExtra,
		CityCode:       record.CityCode,
		Metadata:       metadata,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
	entity.Normalize()

	return entity
}

// generateID creates a random contact identifier.
func generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(bytes)
}

// wrapWriteError normalizes persistence write errors to stable repository errors.
func wrapWriteError(operation string, err error) error {
	if mapped := mapDuplicateError(err); mapped != nil {
		return fmt.Errorf("%s contact record: %w", operation, mapped)
	}

	return fmt.Errorf("%s contact record: %w", operation, err)
}

// mapDuplicateError maps duplicate-key persistence failures into repository-level conflict errors.
func mapDuplicateError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	if !errors.Is(err, gorm.ErrDuplicatedKey) && !isUniqueConstraintError(message) {
		return nil
	}

	if isDocumentConflictError(message) {
		return port.ErrDuplicateDocument
	}
	if isEmailConflictError(message) {
		return port.ErrDuplicateEmail
	}

	return port.ErrDuplicateContact
}

// isUniqueConstraintError reports whether a persistence error message represents uniqueness conflicts.
func isUniqueConstraintError(message string) bool {
	return strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "duplicated key") ||
		strings.Contains(message, "unique constraint failed") ||
		strings.Contains(message, "unique failed")
}

// isDocumentConflictError reports whether a uniqueness conflict references the document composite key.
func isDocumentConflictError(message string) bool {
	return strings.Contains(message, "idx_contacts_document_key") ||
		strings.Contains(message, "contacts.document_key")
}

// isEmailConflictError reports whether a uniqueness conflict references the email key.
func isEmailConflictError(message string) bool {
	return strings.Contains(message, "idx_contacts_email") ||
		strings.Contains(message, "contacts.email")
}

// buildDocumentKey resolves a stable document key for uniqueness checks.
func buildDocumentKey(documentType string, documentNumber string) *string {
	normalizedType := strings.ToUpper(strings.TrimSpace(documentType))
	normalizedNumber := strings.ToUpper(strings.TrimSpace(documentNumber))
	if normalizedType == "" || normalizedNumber == "" {
		return nil
	}

	key := normalizedType + "|" + normalizedNumber
	return &key
}
