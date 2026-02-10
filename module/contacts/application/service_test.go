package application

import (
	"context"
	"errors"
	"testing"

	"mannaiah/module/contacts/domain"
	"mannaiah/module/contacts/port"
)

// repositoryMock defines repository behavior for application tests.
type repositoryMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, contact *domain.Contact) error
	// getByIDFn defines get-by-id behavior.
	getByIDFn func(ctx context.Context, id string) (*domain.Contact, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error)
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, contact *domain.Contact) error
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, id string) error
}

// Create executes configured create behavior.
func (m repositoryMock) Create(ctx context.Context, contact *domain.Contact) error {
	return m.createFn(ctx, contact)
}

// GetByID executes configured get-by-id behavior.
func (m repositoryMock) GetByID(ctx context.Context, id string) (*domain.Contact, error) {
	return m.getByIDFn(ctx, id)
}

// List executes configured list behavior.
func (m repositoryMock) List(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) {
	return m.listFn(ctx, query)
}

// Update executes configured update behavior.
func (m repositoryMock) Update(ctx context.Context, contact *domain.Contact) error {
	return m.updateFn(ctx, contact)
}

// Delete executes configured delete behavior.
func (m repositoryMock) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

// TestNewServiceRejectsNilRepository verifies constructor validation for nil repositories.
func TestNewServiceRejectsNilRepository(t *testing.T) {
	_, err := NewService(nil)
	if !errors.Is(err, ErrNilRepository) {
		t.Fatalf("NewService() error = %v, want ErrNilRepository", err)
	}
}

// TestCreateValidatesDomainRules verifies create validation before persistence.
func TestCreateValidatesDomainRules(t *testing.T) {
	svc, err := NewService(repositoryMock{createFn: func(ctx context.Context, contact *domain.Contact) error {
		return nil
	}, getByIDFn: nil, listFn: nil, updateFn: nil, deleteFn: nil})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = svc.Create(context.Background(), CreateCommand{Email: "", LegalName: "acme"})
	if !errors.Is(err, domain.ErrEmailRequired) {
		t.Fatalf("Create() error = %v, want ErrEmailRequired", err)
	}
}

// TestCreatePersistsContact verifies successful creation writes through repository.
func TestCreatePersistsContact(t *testing.T) {
	created := false
	svc, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error {
			created = true
			contact.ID = "c-1"
			return nil
		},
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) { return nil, nil },
		listFn:    func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn:  func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn:  func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, createErr := svc.Create(context.Background(), CreateCommand{Email: "john@example.com", LegalName: "Acme"})
	if createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}
	if !created {
		t.Fatalf("expected repository create call")
	}
	if result.ID != "c-1" {
		t.Fatalf("ID = %q, want %q", result.ID, "c-1")
	}
}

// TestGetValidatesID verifies id validation before repository access.
func TestGetValidatesID(t *testing.T) {
	svc, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		listFn:   func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, getErr := svc.Get(context.Background(), "  ")
	if !errors.Is(getErr, ErrInvalidID) {
		t.Fatalf("Get() error = %v, want ErrInvalidID", getErr)
	}
}

// TestListNormalizesPagination verifies list defaults and metadata.
func TestListNormalizesPagination(t *testing.T) {
	svc, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		listFn: func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) {
			if query.Page != 1 || query.Limit != 10 {
				t.Fatalf("normalized query = %+v", query)
			}
			return []domain.Contact{{ID: "a"}}, 1, nil
		},
		updateFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	page, listErr := svc.List(context.Background(), port.ListQuery{})
	if listErr != nil {
		t.Fatalf("List() error = %v", listErr)
	}
	if page.Total != 1 || page.TotalPages != 1 || len(page.Data) != 1 {
		t.Fatalf("unexpected list result: %+v", page)
	}
}

// TestUpdateAppliesPatch verifies partial updates and validation before persistence.
func TestUpdateAppliesPatch(t *testing.T) {
	record := &domain.Contact{ID: "c-1", Email: "a@example.com", FirstName: "John", LastName: "Doe"}
	updated := false

	svc, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return record, nil
		},
		listFn: func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) {
			return nil, 0, nil
		},
		updateFn: func(ctx context.Context, contact *domain.Contact) error {
			updated = true
			return nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	newEmail := "next@example.com"
	result, updateErr := svc.Update(context.Background(), "c-1", UpdateCommand{Email: &newEmail})
	if updateErr != nil {
		t.Fatalf("Update() error = %v", updateErr)
	}
	if !updated {
		t.Fatalf("expected repository update call")
	}
	if result.Email != "next@example.com" {
		t.Fatalf("Email = %q, want %q", result.Email, "next@example.com")
	}
}

// TestDeleteDelegates verifies delete operations delegate to repository.
func TestDeleteDelegates(t *testing.T) {
	deleted := false
	svc, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		listFn:   func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn: func(ctx context.Context, id string) error {
			deleted = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if deleteErr := svc.Delete(context.Background(), "c-1"); deleteErr != nil {
		t.Fatalf("Delete() error = %v", deleteErr)
	}
	if !deleted {
		t.Fatalf("expected repository delete call")
	}
}

// TestDeleteInvalidID verifies delete id validation.
func TestDeleteInvalidID(t *testing.T) {
	svc, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		listFn:   func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	deleteErr := svc.Delete(context.Background(), "")
	if !errors.Is(deleteErr, ErrInvalidID) {
		t.Fatalf("Delete() error = %v, want ErrInvalidID", deleteErr)
	}
}

// TestCreatePropagatesRepositoryError verifies wrapped repository errors on create.
func TestCreatePropagatesRepositoryError(t *testing.T) {
	svc, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error { return errors.New("db down") },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		listFn:   func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, createErr := svc.Create(context.Background(), CreateCommand{Email: "john@example.com", LegalName: "Acme"})
	if createErr == nil {
		t.Fatalf("expected create error")
	}
}

// TestGetPropagatesRepositoryError verifies wrapped repository errors on get.
func TestGetPropagatesRepositoryError(t *testing.T) {
	svc, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return nil, errors.New("db read failed")
		},
		listFn:   func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, getErr := svc.Get(context.Background(), "c-1")
	if getErr == nil {
		t.Fatalf("expected get error")
	}
}

// TestUpdateValidationAndErrors verifies update validation and repository error branches.
func TestUpdateValidationAndErrors(t *testing.T) {
	record := &domain.Contact{ID: "c-1", Email: "a@example.com", FirstName: "John", LastName: "Doe"}
	svc, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return record, nil
		},
		listFn: func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) {
			return nil, 0, nil
		},
		updateFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, updateErr := svc.Update(context.Background(), "", UpdateCommand{}); !errors.Is(updateErr, ErrInvalidID) {
		t.Fatalf("Update() error = %v, want ErrInvalidID", updateErr)
	}

	empty := ""
	if _, updateErr := svc.Update(context.Background(), "c-1", UpdateCommand{Email: &empty}); !errors.Is(updateErr, domain.ErrEmailRequired) {
		t.Fatalf("Update() error = %v, want ErrEmailRequired", updateErr)
	}

	svcWithGetError, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return nil, errors.New("missing")
		},
		listFn:   func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, updateErr := svcWithGetError.Update(context.Background(), "c-1", UpdateCommand{}); updateErr == nil {
		t.Fatalf("expected update load error")
	}
}

// TestDeletePropagatesRepositoryError verifies wrapped repository errors on delete.
func TestDeletePropagatesRepositoryError(t *testing.T) {
	svc, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		listFn:   func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn: func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return errors.New("delete failed") },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	deleteErr := svc.Delete(context.Background(), "c-1")
	if deleteErr == nil {
		t.Fatalf("expected delete error")
	}
}

// TestCreatePublishesIntegrationEvent verifies create publishes integration events.
func TestCreatePublishesIntegrationEvent(t *testing.T) {
	published := false
	svc, err := NewServiceWithPublisher(repositoryMock{
		createFn: func(ctx context.Context, contact *domain.Contact) error {
			contact.ID = "c-1"
			return nil
		},
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) { return nil, nil },
		listFn:    func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn:  func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn:  func(ctx context.Context, id string) error { return nil },
	}, integrationEventPublisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error {
		published = true
		if event.Topic != TopicContactCreated {
			t.Fatalf("Topic = %q, want %q", event.Topic, TopicContactCreated)
		}
		return nil
	}})
	if err != nil {
		t.Fatalf("NewServiceWithPublisher() error = %v", err)
	}

	_, createErr := svc.Create(context.Background(), CreateCommand{Email: "john@example.com", LegalName: "Acme"})
	if createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}
	if !published {
		t.Fatalf("expected integration event publish")
	}
}

// TestUpdatePublishesIntegrationEvent verifies update publishes integration events.
func TestUpdatePublishesIntegrationEvent(t *testing.T) {
	record := &domain.Contact{ID: "c-1", Email: "a@example.com", FirstName: "John", LastName: "Doe"}
	published := false
	svc, err := NewServiceWithPublisher(repositoryMock{
		createFn:  func(ctx context.Context, contact *domain.Contact) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*domain.Contact, error) { return record, nil },
		listFn:    func(ctx context.Context, query port.ListQuery) ([]domain.Contact, int64, error) { return nil, 0, nil },
		updateFn:  func(ctx context.Context, contact *domain.Contact) error { return nil },
		deleteFn:  func(ctx context.Context, id string) error { return nil },
	}, integrationEventPublisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error {
		published = true
		if event.Topic != TopicContactUpdated {
			t.Fatalf("Topic = %q, want %q", event.Topic, TopicContactUpdated)
		}
		return nil
	}})
	if err != nil {
		t.Fatalf("NewServiceWithPublisher() error = %v", err)
	}

	value := "next@example.com"
	_, updateErr := svc.Update(context.Background(), "c-1", UpdateCommand{Email: &value})
	if updateErr != nil {
		t.Fatalf("Update() error = %v", updateErr)
	}
	if !published {
		t.Fatalf("expected integration event publish")
	}
}
