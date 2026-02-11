package contacts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	contactapplication "mannaiah/module/contacts/application"
	contactdomain "mannaiah/module/contacts/domain"
	contactport "mannaiah/module/contacts/port"
	"mannaiah/module/woocommerce/port"
)

var (
	// ErrNilService is returned when a nil contact service dependency is provided.
	ErrNilService = errors.New("contacts upserter service must not be nil")
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

	if existing == nil {
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
		}); createErr != nil {
			if !isDuplicateCreateError(createErr) {
				return "", fmt.Errorf("create contact for woocommerce sync: %w", createErr)
			}

			latest, findErr := u.findByEmail(ctx, command.Email)
			if findErr != nil {
				return "", findErr
			}
			if latest == nil {
				return "", fmt.Errorf("create contact for woocommerce sync: %w", createErr)
			}
			if !hasMeaningfulChange(*latest, command) {
				return port.UpsertOutcomeUnchanged, nil
			}
			if err := u.updateExisting(ctx, latest.ID, command); err != nil {
				return "", err
			}

			return port.UpsertOutcomeUpdated, nil
		}

		return port.UpsertOutcomeCreated, nil
	}

	if !hasMeaningfulChange(*existing, command) {
		return port.UpsertOutcomeUnchanged, nil
	}

	if err := u.updateExisting(ctx, existing.ID, command); err != nil {
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

// updateExisting applies update payload values to existing contacts.
func (u *Upserter) updateExisting(ctx context.Context, id string, command port.ContactSyncCommand) error {
	_, err := u.service.Update(ctx, id, contactapplication.UpdateCommand{
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
	})
	if err != nil {
		return fmt.Errorf("update contact for woocommerce sync: %w", err)
	}

	return nil
}

// isDuplicateCreateError reports duplicate-create failures handled as retryable lookup flows.
func isDuplicateCreateError(err error) bool {
	return errors.Is(err, contactport.ErrDuplicateEmail) ||
		errors.Is(err, contactport.ErrDuplicateContact) ||
		errors.Is(err, contactport.ErrDuplicateDocument)
}

// hasMeaningfulChange reports whether non-document fields changed between existing contacts and sync commands.
func hasMeaningfulChange(existing contactdomain.Contact, command port.ContactSyncCommand) bool {
	if strings.ToLower(strings.TrimSpace(existing.Email)) != strings.ToLower(strings.TrimSpace(command.Email)) {
		return true
	}
	if strings.TrimSpace(existing.FirstName) != strings.TrimSpace(command.FirstName) {
		return true
	}
	if strings.TrimSpace(existing.LastName) != strings.TrimSpace(command.LastName) {
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
