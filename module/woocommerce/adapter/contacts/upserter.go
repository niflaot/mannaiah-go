package contacts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	contactapplication "mannaiah/module/contacts/application"
	contactdomain "mannaiah/module/contacts/domain"
	contactport "mannaiah/module/contacts/port"
	"mannaiah/module/woocommerce/port"
)

var (
	// ErrNilService is returned when a nil contact service dependency is provided.
	ErrNilService = errors.New("contacts upserter service must not be nil")
)

const (
	// circleOptInMetadataKey defines metadata key used for circle opt-in decisions.
	circleOptInMetadataKey = "flock_checker_circle_optin"
)

// Upserter defines contact upsert behavior backed by contacts application services.
type Upserter struct {
	// service defines contacts application service dependencies.
	service contactapplication.Service
}

var (
	// _ ensures Upserter satisfies WooCommerce contact sync target contracts.
	_ port.ContactSyncTarget = (*Upserter)(nil)
)

// NewUpserter creates contact upsert adapters over contacts application services.
func NewUpserter(service contactapplication.Service) (*Upserter, error) {
	if service == nil {
		return nil, ErrNilService
	}

	return &Upserter{service: service}, nil
}

// UpsertByEmail creates or updates contacts keyed by email and reports upsert outcomes.
func (u *Upserter) UpsertByEmail(ctx context.Context, command port.ContactSyncCommand) (port.UpsertOutcome, error) {
	existing, err := u.findByEmail(ctx, command.Email)
	if err != nil {
		return "", err
	}
	normalizedMetadata := normalizeSyncMetadata(command.Metadata)

	if existing == nil {
		createdAt := cloneTimePointer(command.CreatedAt)
		if _, createErr := u.service.Create(ctx, contactapplication.CreateCommand{
			Email:          strings.TrimSpace(command.Email),
			FirstName:      strings.TrimSpace(command.FirstName),
			LastName:       strings.TrimSpace(command.LastName),
			LegalName:      strings.TrimSpace(command.LegalName),
			Phone:          strings.TrimSpace(command.Phone),
			Address:        strings.TrimSpace(command.Address),
			AddressExtra:   strings.TrimSpace(command.AddressExtra),
			CityCode:       strings.TrimSpace(command.CityCode),
			DocumentType:   contactdomain.DocumentType(strings.TrimSpace(command.DocumentType)),
			DocumentNumber: strings.TrimSpace(command.DocumentNumber),
			CreatedAt:      createdAt,
			Metadata:       normalizedMetadata,
		}); createErr != nil {
			if !isDuplicateCreateError(createErr) {
				return "", fmt.Errorf("create contact for woocommerce sync: %w", createErr)
			}

			latest, findErr := u.findByEmail(ctx, command.Email)
			if findErr != nil {
				return "", findErr
			}
			if latest == nil {
				latest, findErr = u.findByDocument(ctx, command.DocumentType, command.DocumentNumber)
				if findErr != nil {
					return "", findErr
				}
			}
			if latest == nil {
				return "", fmt.Errorf("create contact for woocommerce sync: %w", createErr)
			}
			if !hasMeaningfulChange(*latest, command, normalizedMetadata) {
				return port.UpsertOutcomeUnchanged, nil
			}
			if err := u.updateExisting(ctx, *latest, command, normalizedMetadata); err != nil {
				return "", err
			}

			return port.UpsertOutcomeUpdated, nil
		}

		return port.UpsertOutcomeCreated, nil
	}

	if !hasMeaningfulChange(*existing, command, normalizedMetadata) {
		return port.UpsertOutcomeUnchanged, nil
	}

	if err := u.updateExisting(ctx, *existing, command, normalizedMetadata); err != nil {
		return "", err
	}
	return port.UpsertOutcomeUpdated, nil
}

// findByEmail retrieves an optional contact by email.
func (u *Upserter) findByEmail(ctx context.Context, email string) (*contactdomain.Contact, error) {
	result, err := u.service.List(ctx, contactport.ListQuery{
		Page:  1,
		Limit: 1,
		Email: email,
	})
	if err != nil {
		return nil, fmt.Errorf("list contacts by email: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, nil
	}

	contact := result.Data[0]
	return &contact, nil
}

// findByDocument retrieves an optional contact by document identity values.
func (u *Upserter) findByDocument(ctx context.Context, documentType string, documentNumber string) (*contactdomain.Contact, error) {
	normalizedType := strings.TrimSpace(documentType)
	normalizedNumber := strings.TrimSpace(documentNumber)
	if normalizedType == "" || normalizedNumber == "" {
		return nil, nil
	}

	result, err := u.service.List(ctx, contactport.ListQuery{
		Page:           1,
		Limit:          1,
		DocumentType:   normalizedType,
		DocumentNumber: normalizedNumber,
	})
	if err != nil {
		return nil, fmt.Errorf("list contacts by document: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, nil
	}
	contact := result.Data[0]
	if !strings.EqualFold(strings.TrimSpace(string(contact.DocumentType)), normalizedType) {
		return nil, nil
	}
	if strings.TrimSpace(contact.DocumentNumber) != normalizedNumber {
		return nil, nil
	}

	return &contact, nil
}

// updateExisting applies update payload values to existing contacts.
func (u *Upserter) updateExisting(ctx context.Context, existing contactdomain.Contact, command port.ContactSyncCommand, normalizedMetadata map[string]string) error {
	mergedMetadata := mergeMetadata(existing.Metadata, normalizedMetadata)
	updateCommand := contactapplication.UpdateCommand{
		Email:          pointer(strings.TrimSpace(command.Email)),
		FirstName:      pointer(strings.TrimSpace(command.FirstName)),
		LastName:       pointer(strings.TrimSpace(command.LastName)),
		LegalName:      pointer(strings.TrimSpace(command.LegalName)),
		Phone:          pointer(strings.TrimSpace(command.Phone)),
		Address:        pointer(strings.TrimSpace(command.Address)),
		AddressExtra:   pointer(strings.TrimSpace(command.AddressExtra)),
		CityCode:       pointer(strings.TrimSpace(command.CityCode)),
		DocumentType:   documentTypePointer(command.DocumentType),
		DocumentNumber: pointer(strings.TrimSpace(command.DocumentNumber)),
	}
	if len(mergedMetadata) > 0 {
		updateCommand.Metadata = &mergedMetadata
	}
	if shouldUpdateCreatedAt(existing.CreatedAt, command.CreatedAt) {
		updateCommand.CreatedAt = cloneTimePointer(command.CreatedAt)
	}

	_, err := u.service.Update(ctx, existing.ID, updateCommand)
	if err != nil {
		return fmt.Errorf("update contact for woocommerce sync: %w", err)
	}

	return nil
}

// isDuplicateCreateError reports duplicate-create failures handled as retryable lookup flows.
func isDuplicateCreateError(err error) bool {
	return errors.Is(err, contactport.ErrDuplicateEmail) ||
		errors.Is(err, contactport.ErrDuplicateContact) ||
		errors.Is(err, contactport.ErrDuplicateDocument) ||
		isDuplicateKeyErrorMessage(err)
}

// isDuplicateKeyErrorMessage reports duplicate-key create errors by SQL driver message fallback.
func isDuplicateKeyErrorMessage(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	if message == "" {
		return false
	}
	if strings.Contains(message, "error 1062") || strings.Contains(message, "duplicate entry") {
		return true
	}

	return strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "duplicated key") ||
		strings.Contains(message, "unique constraint failed")
}

// hasMeaningfulChange reports whether non-document fields changed between existing contacts and sync commands.
func hasMeaningfulChange(existing contactdomain.Contact, command port.ContactSyncCommand, normalizedMetadata map[string]string) bool {
	if strings.ToLower(strings.TrimSpace(existing.Email)) != strings.ToLower(strings.TrimSpace(command.Email)) {
		return true
	}
	if strings.TrimSpace(existing.FirstName) != strings.TrimSpace(command.FirstName) {
		return true
	}
	if strings.TrimSpace(existing.LastName) != strings.TrimSpace(command.LastName) {
		return true
	}
	if strings.TrimSpace(existing.LegalName) != strings.TrimSpace(command.LegalName) {
		return true
	}
	if strings.TrimSpace(existing.Phone) != strings.TrimSpace(command.Phone) {
		return true
	}
	if strings.TrimSpace(existing.Address) != strings.TrimSpace(command.Address) {
		return true
	}
	if strings.TrimSpace(existing.AddressExtra) != strings.TrimSpace(command.AddressExtra) {
		return true
	}
	if strings.TrimSpace(existing.CityCode) != strings.TrimSpace(command.CityCode) {
		return true
	}
	if shouldUpdateCreatedAt(existing.CreatedAt, command.CreatedAt) {
		return true
	}
	if !metadataEqual(existing.Metadata, mergeMetadata(existing.Metadata, normalizedMetadata)) {
		return true
	}

	return false
}

// documentTypePointer maps string document-type values to contact domain document type pointers.
func documentTypePointer(value string) *contactdomain.DocumentType {
	trimmed := strings.TrimSpace(value)
	resolved := contactdomain.DocumentType(trimmed)
	return &resolved
}

// pointer returns a pointer for value.
func pointer(value string) *string {
	return &value
}

// normalizeSyncMetadata normalizes sync metadata maps.
func normalizeSyncMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}

	normalized := make(map[string]string, len(metadata))
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		normalized[trimmedKey] = strings.TrimSpace(value)
	}
	if len(normalized) == 0 {
		return nil
	}
	stripCircleOptInMetadata(normalized)

	return normalized
}

// mergeMetadata merges sync metadata into existing contact metadata values.
func mergeMetadata(existing map[string]string, syncMetadata map[string]string) map[string]string {
	if len(existing) == 0 && len(syncMetadata) == 0 {
		return nil
	}

	merged := make(map[string]string, len(existing)+len(syncMetadata))
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range syncMetadata {
		merged[key] = value
	}
	stripCircleOptInMetadata(merged)

	return merged
}

// stripCircleOptInMetadata removes circle opt-in metadata keys from contact metadata payload values.
func stripCircleOptInMetadata(metadata map[string]string) {
	if len(metadata) == 0 {
		return
	}

	for key := range metadata {
		if strings.HasPrefix(strings.TrimSpace(key), circleOptInMetadataKey) {
			delete(metadata, key)
		}
	}
}

// metadataEqual compares metadata maps.
func metadataEqual(left map[string]string, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}

	for key, value := range left {
		if right[key] != value {
			return false
		}
	}

	return true
}

// shouldUpdateCreatedAt reports whether sync commands should update existing creation timestamps.
func shouldUpdateCreatedAt(existing time.Time, candidate *time.Time) bool {
	if candidate == nil || candidate.IsZero() {
		return false
	}
	if existing.IsZero() {
		return true
	}

	return candidate.UTC().Before(existing.UTC())
}

// cloneTimePointer clones time pointer values.
func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	copied := value.UTC()
	return &copied
}
