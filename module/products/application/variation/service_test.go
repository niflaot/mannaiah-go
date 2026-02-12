package variation

import (
	"context"
	errorspkg "errors"
	"testing"

	variationdomain "mannaiah/module/products/domain/variation"
	variationport "mannaiah/module/products/port/variation"
)

// repositoryMock defines persistence behavior for service tests.
type repositoryMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, variation *variationdomain.Variation) error
	// getFn defines get behavior.
	getFn func(ctx context.Context, id string) (*variationdomain.Variation, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context) ([]variationdomain.Variation, error)
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, variation *variationdomain.Variation) error
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, id string) error
}

// EnsureSchema ignores schema behavior for service tests.
func (m repositoryMock) EnsureSchema(ctx context.Context) error { return nil }

// Create executes configured create behavior.
func (m repositoryMock) Create(ctx context.Context, variation *variationdomain.Variation) error {
	return m.createFn(ctx, variation)
}

// GetByID executes configured get behavior.
func (m repositoryMock) GetByID(ctx context.Context, id string) (*variationdomain.Variation, error) {
	return m.getFn(ctx, id)
}

// List executes configured list behavior.
func (m repositoryMock) List(ctx context.Context) ([]variationdomain.Variation, error) {
	return m.listFn(ctx)
}

// Update executes configured update behavior.
func (m repositoryMock) Update(ctx context.Context, variation *variationdomain.Variation) error {
	return m.updateFn(ctx, variation)
}

// Delete executes configured delete behavior.
func (m repositoryMock) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

// TestNewService validates constructor behavior.
func TestNewService(t *testing.T) {
	if _, err := NewService(nil); !errorspkg.Is(err, ErrNilRepository) {
		t.Fatalf("NewService(nil) error = %v, want ErrNilRepository", err)
	}
}

// TestCreate verifies create behavior.
func TestCreate(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, variation *variationdomain.Variation) error {
			variation.ID = "v-1"
			return nil
		},
		getFn:    func(ctx context.Context, id string) (*variationdomain.Variation, error) { return nil, nil },
		listFn:   func(ctx context.Context) ([]variationdomain.Variation, error) { return nil, nil },
		updateFn: func(ctx context.Context, variation *variationdomain.Variation) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	entity, createErr := service.Create(context.Background(), CreateCommand{Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"})
	if createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}
	if entity.ID != "v-1" {
		t.Fatalf("entity.ID = %q, want %q", entity.ID, "v-1")
	}
}

// TestGetListDelete verifies get/list/delete behaviors.
func TestGetListDelete(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, variation *variationdomain.Variation) error { return nil },
		getFn: func(ctx context.Context, id string) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{ID: id, Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"}, nil
		},
		listFn: func(ctx context.Context) ([]variationdomain.Variation, error) {
			return []variationdomain.Variation{{ID: "v-1", Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"}}, nil
		},
		updateFn: func(ctx context.Context, variation *variationdomain.Variation) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, getErr := service.Get(context.Background(), ""); !errorspkg.Is(getErr, ErrInvalidID) {
		t.Fatalf("Get(empty) error = %v, want ErrInvalidID", getErr)
	}
	if _, getErr := service.Get(context.Background(), "v-1"); getErr != nil {
		t.Fatalf("Get() error = %v", getErr)
	}
	if values, listErr := service.List(context.Background()); listErr != nil || len(values) != 1 {
		t.Fatalf("List() error = %v len = %d", listErr, len(values))
	}
	if deleteErr := service.Delete(context.Background(), ""); !errorspkg.Is(deleteErr, ErrInvalidID) {
		t.Fatalf("Delete(empty) error = %v, want ErrInvalidID", deleteErr)
	}
}

// TestUpdateIgnoresDefinition verifies immutable definition behavior.
func TestUpdateIgnoresDefinition(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, variation *variationdomain.Variation) error { return nil },
		getFn: func(ctx context.Context, id string) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{ID: id, Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"}, nil
		},
		listFn: func(ctx context.Context) ([]variationdomain.Variation, error) { return nil, nil },
		updateFn: func(ctx context.Context, variation *variationdomain.Variation) error {
			if variation.Definition != variationdomain.DefinitionColor {
				return errorspkg.New("definition mutated")
			}
			return nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	name := "Dark Red"
	value := "#8B0000"
	definition := variationdomain.DefinitionSize
	entity, updateErr := service.Update(context.Background(), "v-1", UpdateCommand{Name: &name, Value: &value, Definition: &definition})
	if updateErr != nil {
		t.Fatalf("Update() error = %v", updateErr)
	}
	if entity.Definition != variationdomain.DefinitionColor {
		t.Fatalf("entity.Definition = %q, want %q", entity.Definition, variationdomain.DefinitionColor)
	}
}

// TestErrorWrapping verifies wrapped repository errors.
func TestErrorWrapping(t *testing.T) {
	repositoryErr := errorspkg.New("repository failed")
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, variation *variationdomain.Variation) error { return repositoryErr },
		getFn:    func(ctx context.Context, id string) (*variationdomain.Variation, error) { return nil, repositoryErr },
		listFn:   func(ctx context.Context) ([]variationdomain.Variation, error) { return nil, repositoryErr },
		updateFn: func(ctx context.Context, variation *variationdomain.Variation) error { return repositoryErr },
		deleteFn: func(ctx context.Context, id string) error { return repositoryErr },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"}); createErr == nil {
		t.Fatalf("expected create wrapped error")
	}
	if _, getErr := service.Get(context.Background(), "id"); getErr == nil {
		t.Fatalf("expected get wrapped error")
	}
	if _, listErr := service.List(context.Background()); listErr == nil {
		t.Fatalf("expected list wrapped error")
	}
	name := "Name"
	if _, updateErr := service.Update(context.Background(), "id", UpdateCommand{Name: &name}); updateErr == nil {
		t.Fatalf("expected update wrapped error")
	}
	if deleteErr := service.Delete(context.Background(), "id"); deleteErr == nil {
		t.Fatalf("expected delete wrapped error")
	}
}

// TestValidationErrors verifies validation propagation behavior.
func TestValidationErrors(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, variation *variationdomain.Variation) error { return nil },
		getFn: func(ctx context.Context, id string) (*variationdomain.Variation, error) {
			return &variationdomain.Variation{ID: "v-1", Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"}, nil
		},
		listFn:   func(ctx context.Context) ([]variationdomain.Variation, error) { return nil, nil },
		updateFn: func(ctx context.Context, variation *variationdomain.Variation) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{}); !errorspkg.Is(createErr, variationdomain.ErrNameRequired) {
		t.Fatalf("Create() error = %v, want variationdomain.ErrNameRequired", createErr)
	}
	value := ""
	if _, updateErr := service.Update(context.Background(), "v-1", UpdateCommand{Value: &value}); !errorspkg.Is(updateErr, variationdomain.ErrValueRequired) {
		t.Fatalf("Update() error = %v, want variationdomain.ErrValueRequired", updateErr)
	}
}

// TestRepositoryErrorUnwrap verifies wrapped errors include repository sentinel values.
func TestRepositoryErrorUnwrap(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, variation *variationdomain.Variation) error {
			return variationport.ErrNotFound
		},
		getFn: func(ctx context.Context, id string) (*variationdomain.Variation, error) {
			return nil, variationport.ErrNotFound
		},
		listFn: func(ctx context.Context) ([]variationdomain.Variation, error) { return nil, nil },
		updateFn: func(ctx context.Context, variation *variationdomain.Variation) error {
			return variationport.ErrNotFound
		},
		deleteFn: func(ctx context.Context, id string) error { return variationport.ErrNotFound },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{Name: "Red", Definition: variationdomain.DefinitionColor, Value: "#FF0000"}); !errorspkg.Is(createErr, variationport.ErrNotFound) {
		t.Fatalf("Create() error = %v, want variationport.ErrNotFound", createErr)
	}
	if _, getErr := service.Get(context.Background(), "id"); !errorspkg.Is(getErr, variationport.ErrNotFound) {
		t.Fatalf("Get() error = %v, want variationport.ErrNotFound", getErr)
	}
	if deleteErr := service.Delete(context.Background(), "id"); !errorspkg.Is(deleteErr, variationport.ErrNotFound) {
		t.Fatalf("Delete() error = %v, want variationport.ErrNotFound", deleteErr)
	}
}
