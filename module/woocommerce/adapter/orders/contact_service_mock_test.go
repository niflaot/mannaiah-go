package orders

import (
	"context"
	"fmt"
	"strings"

	contactsapplication "mannaiah/module/contacts/application"
	contactdomain "mannaiah/module/contacts/domain"
	contactport "mannaiah/module/contacts/port"
)

// contactServiceMock defines contact service behavior for order upserter tests.
type contactServiceMock struct {
	// listErr defines list-operation errors.
	listErr error
	// contactsByID stores contacts keyed by id.
	contactsByID map[string]contactdomain.Contact
	// contactIDByEmail stores contact ids keyed by normalized email.
	contactIDByEmail map[string]string
	// created stores create command values.
	created []contactsapplication.CreateCommand
	// updated stores update command values.
	updated []contactsapplication.UpdateCommand
}

// Create stores created contacts and returns persisted rows.
func (m *contactServiceMock) Create(ctx context.Context, command contactsapplication.CreateCommand) (*contactdomain.Contact, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(command.Email))
	if normalizedEmail == "" {
		return nil, fmt.Errorf("email is required")
	}
	if _, exists := m.contactIDByEmail[normalizedEmail]; exists {
		return nil, contactport.ErrDuplicateEmail
	}

	identifier := fmt.Sprintf("contact-%d", len(m.created)+1)
	entity := contactdomain.Contact{
		ID:             identifier,
		Email:          normalizedEmail,
		FirstName:      strings.TrimSpace(command.FirstName),
		LastName:       strings.TrimSpace(command.LastName),
		LegalName:      strings.TrimSpace(command.LegalName),
		Phone:          strings.TrimSpace(command.Phone),
		Address:        strings.TrimSpace(command.Address),
		AddressExtra:   strings.TrimSpace(command.AddressExtra),
		CityCode:       strings.TrimSpace(command.CityCode),
		DocumentType:   command.DocumentType,
		DocumentNumber: strings.TrimSpace(command.DocumentNumber),
		Metadata:       cloneMetadata(command.Metadata),
	}
	if command.CreatedAt != nil {
		entity.CreatedAt = command.CreatedAt.UTC()
	}

	m.created = append(m.created, command)
	m.contactsByID[identifier] = entity
	m.contactIDByEmail[normalizedEmail] = identifier

	return &entity, nil
}

// Get returns contacts by id.
func (m *contactServiceMock) Get(ctx context.Context, id string) (*contactdomain.Contact, error) {
	entity, ok := m.contactsByID[strings.TrimSpace(id)]
	if !ok {
		return nil, contactport.ErrNotFound
	}

	copied := entity
	return &copied, nil
}

// List returns optional contact rows filtered by email.
func (m *contactServiceMock) List(ctx context.Context, query contactport.ListQuery) (*contactsapplication.ListResult, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}

	email := strings.ToLower(strings.TrimSpace(query.Email))
	if email == "" {
		return &contactsapplication.ListResult{}, nil
	}
	id, ok := m.contactIDByEmail[email]
	if !ok {
		return &contactsapplication.ListResult{}, nil
	}

	entity := m.contactsByID[id]
	return &contactsapplication.ListResult{
		Data: []contactdomain.Contact{entity},
	}, nil
}

// Update updates contacts by id.
func (m *contactServiceMock) Update(ctx context.Context, id string, command contactsapplication.UpdateCommand) (*contactdomain.Contact, error) {
	entity, ok := m.contactsByID[strings.TrimSpace(id)]
	if !ok {
		return nil, contactport.ErrNotFound
	}

	if command.Email != nil {
		entity.Email = strings.ToLower(strings.TrimSpace(*command.Email))
	}
	if command.FirstName != nil {
		entity.FirstName = strings.TrimSpace(*command.FirstName)
	}
	if command.LastName != nil {
		entity.LastName = strings.TrimSpace(*command.LastName)
	}
	if command.LegalName != nil {
		entity.LegalName = strings.TrimSpace(*command.LegalName)
	}
	if command.Phone != nil {
		entity.Phone = strings.TrimSpace(*command.Phone)
	}
	if command.Address != nil {
		entity.Address = strings.TrimSpace(*command.Address)
	}
	if command.AddressExtra != nil {
		entity.AddressExtra = strings.TrimSpace(*command.AddressExtra)
	}
	if command.CityCode != nil {
		entity.CityCode = strings.TrimSpace(*command.CityCode)
	}
	if command.DocumentType != nil {
		entity.DocumentType = *command.DocumentType
	}
	if command.DocumentNumber != nil {
		entity.DocumentNumber = strings.TrimSpace(*command.DocumentNumber)
	}
	if command.CreatedAt != nil {
		entity.CreatedAt = command.CreatedAt.UTC()
	}
	if command.Metadata != nil {
		entity.Metadata = cloneMetadata(*command.Metadata)
	}

	m.updated = append(m.updated, command)
	m.contactsByID[entity.ID] = entity
	m.contactIDByEmail[strings.ToLower(entity.Email)] = entity.ID

	copied := entity
	return &copied, nil
}

// Delete deletes contact rows by id.
func (m *contactServiceMock) Delete(ctx context.Context, id string) error {
	delete(m.contactsByID, strings.TrimSpace(id))
	return nil
}
