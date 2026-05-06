package service

import (
	"context"
	"errors"
	"strings"
	"time"

	contactsapplication "mannaiah/module/contacts/application"
	contactsdomain "mannaiah/module/contacts/domain"
	contactsport "mannaiah/module/contacts/port"
	shopifyport "mannaiah/module/shopify/port"
)

var (
	// ErrNilContactsService is returned when a nil contacts service is provided.
	ErrNilContactsService = errors.New("shopify contacts application service must not be nil")
	// ErrNilSyncLinkRepository is returned when a nil sync-link repository is provided.
	ErrNilSyncLinkRepository = errors.New("shopify sync link repository must not be nil")
)

// ContactUpserter defines mainstream contact upsert behavior for Shopify sync flows.
type ContactUpserter struct {
	// service defines mainstream contact application dependencies.
	service contactsapplication.Service
	// links defines Shopify sync-link persistence dependencies.
	links shopifyport.SyncLinkRepository
}

var (
	// _ ensures ContactUpserter satisfies Shopify contact target contracts.
	_ shopifyport.ContactSyncTarget = (*ContactUpserter)(nil)
)

// NewUpserter creates mainstream contact upsert adapters for Shopify sync flows.
func NewUpserter(service contactsapplication.Service, links shopifyport.SyncLinkRepository) (*ContactUpserter, error) {
	if service == nil {
		return nil, ErrNilContactsService
	}
	if links == nil {
		return nil, ErrNilSyncLinkRepository
	}

	return &ContactUpserter{service: service, links: links}, nil
}

// UpsertContact creates or updates one mainstream contact from Shopify values.
func (u *ContactUpserter) UpsertContact(ctx context.Context, command shopifyport.ContactSyncCommand) (*contactsdomain.Contact, error) {
	existing, err := u.findExisting(ctx, command)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		updated, updateErr := u.service.Update(ctx, existing.ID, buildContactUpdateCommand(existing, command))
		if updateErr != nil {
			return nil, updateErr
		}
		if linkErr := u.upsertLink(ctx, command.ShopifyID, updated.ID); linkErr != nil {
			return nil, linkErr
		}
		return updated, nil
	}

	created, createErr := u.service.Create(ctx, buildContactCreateCommand(command))
	if createErr != nil {
		if !errors.Is(createErr, contactsport.ErrDuplicateContact) && !errors.Is(createErr, contactsport.ErrDuplicateEmail) && !errors.Is(createErr, contactsport.ErrDuplicateDocument) {
			return nil, createErr
		}
		existing, err = u.findExisting(ctx, command)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, createErr
		}
		updated, updateErr := u.service.Update(ctx, existing.ID, buildContactUpdateCommand(existing, command))
		if updateErr != nil {
			return nil, updateErr
		}
		if linkErr := u.upsertLink(ctx, command.ShopifyID, updated.ID); linkErr != nil {
			return nil, linkErr
		}
		return updated, nil
	}
	if linkErr := u.upsertLink(ctx, command.ShopifyID, created.ID); linkErr != nil {
		return nil, linkErr
	}

	return created, nil
}

func (u *ContactUpserter) findExisting(ctx context.Context, command shopifyport.ContactSyncCommand) (*contactsdomain.Contact, error) {
	if entity, err := u.findByEmail(ctx, command.Email); err != nil || entity != nil {
		return entity, err
	}
	if strings.TrimSpace(command.DocumentNumber) == "" || strings.TrimSpace(string(command.DocumentType)) == "" {
		return nil, nil
	}

	return u.findByDocument(ctx, command.DocumentType, command.DocumentNumber)
}

func (u *ContactUpserter) findByEmail(ctx context.Context, email string) (*contactsdomain.Contact, error) {
	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" {
		return nil, nil
	}

	result, err := u.service.List(ctx, contactsport.ListQuery{Page: 1, Limit: 1, Email: trimmedEmail})
	if err != nil || result == nil || len(result.Data) == 0 {
		return nil, err
	}
	entity := result.Data[0]
	return &entity, nil
}

func (u *ContactUpserter) findByDocument(ctx context.Context, documentType contactsdomain.DocumentType, documentNumber string) (*contactsdomain.Contact, error) {
	result, err := u.service.List(ctx, contactsport.ListQuery{
		Page:           1,
		Limit:          1,
		DocumentType:   string(documentType),
		DocumentNumber: strings.TrimSpace(documentNumber),
	})
	if err != nil || result == nil || len(result.Data) == 0 {
		return nil, err
	}
	entity := result.Data[0]
	return &entity, nil
}

func (u *ContactUpserter) upsertLink(ctx context.Context, shopifyID string, contactID string) error {
	if strings.TrimSpace(shopifyID) == "" || strings.TrimSpace(contactID) == "" {
		return nil
	}
	lastSyncedAt := time.Now().UTC()
	_, err := u.links.UpsertLink(ctx, shopifyport.UpsertSyncLinkInput{
		Kind:         shopifyport.SyncKindContact,
		ShopifyID:    strings.TrimSpace(shopifyID),
		MannaiahID:   strings.TrimSpace(contactID),
		LastSyncedAt: &lastSyncedAt,
	})
	return err
}

func buildContactCreateCommand(command shopifyport.ContactSyncCommand) contactsapplication.CreateCommand {
	return contactsapplication.CreateCommand{
		DocumentType:   command.DocumentType,
		DocumentNumber: strings.TrimSpace(command.DocumentNumber),
		LegalName:      strings.TrimSpace(command.LegalName),
		FirstName:      strings.TrimSpace(command.FirstName),
		LastName:       strings.TrimSpace(command.LastName),
		Email:          strings.TrimSpace(command.Email),
		Phone:          strings.TrimSpace(command.Phone),
		Address:        strings.TrimSpace(command.Address),
		AddressExtra:   strings.TrimSpace(command.AddressExtra),
		CityCode:       strings.TrimSpace(command.CityCode),
		Metadata:       cloneMetadata(command.Metadata),
		CreatedAt:      command.CreatedAt,
	}
}

func buildContactUpdateCommand(existing *contactsdomain.Contact, command shopifyport.ContactSyncCommand) contactsapplication.UpdateCommand {
	metadata := mergeMetadata(nil, command.Metadata)
	if existing != nil {
		metadata = mergeMetadata(existing.Metadata, command.Metadata)
	}
	return contactsapplication.UpdateCommand{
		DocumentType:   pointerDocumentType(command.DocumentType),
		DocumentNumber: pointerString(command.DocumentNumber),
		LegalName:      pointerString(command.LegalName),
		FirstName:      pointerString(command.FirstName),
		LastName:       pointerString(command.LastName),
		Email:          pointerString(command.Email),
		Phone:          pointerString(command.Phone),
		Address:        pointerString(command.Address),
		AddressExtra:   pointerString(command.AddressExtra),
		CityCode:       pointerString(command.CityCode),
		Metadata:       &metadata,
		CreatedAt:      command.CreatedAt,
	}
}

func mergeMetadata(base map[string]string, updates map[string]string) map[string]string {
	result := cloneMetadata(base)
	if result == nil {
		result = map[string]string{}
	}
	for key, value := range updates {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = strings.TrimSpace(value)
	}
	return result
}

func cloneMetadata(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = strings.TrimSpace(value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func pointerString(value string) *string {
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func pointerDocumentType(value contactsdomain.DocumentType) *contactsdomain.DocumentType {
	resolved := contactsdomain.DocumentType(strings.TrimSpace(string(value)))
	return &resolved
}
