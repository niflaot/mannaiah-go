package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	corehttp "mannaiah/module/core/http"
	pageservice "mannaiah/module/storefront/application/page/service"
	renderableservice "mannaiah/module/storefront/application/renderable/service"
	"mannaiah/module/storefront/domain"
	"mannaiah/module/storefront/port"
)

// renderableServiceMock defines renderable handler dependencies for tests.
type renderableServiceMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, cmd renderableservice.CreateCommand) (*domain.Renderable, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context, query port.RenderableListQuery) ([]domain.Renderable, int64, error)
}

// Create persists a new draft renderable.
func (m renderableServiceMock) Create(ctx context.Context, cmd renderableservice.CreateCommand) (*domain.Renderable, error) {
	return m.createFn(ctx, cmd)
}

// GetByID loads one renderable by identifier.
func (m renderableServiceMock) GetByID(_ context.Context, _ string) (*domain.Renderable, error) {
	return nil, renderableservice.ErrRenderableNotFound
}

// Update applies draft changes.
func (m renderableServiceMock) Update(_ context.Context, _ renderableservice.UpdateCommand) (*domain.Renderable, error) {
	return nil, errors.New("not implemented")
}

// Delete removes one renderable.
func (m renderableServiceMock) Delete(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

// List returns paginated renderables.
func (m renderableServiceMock) List(ctx context.Context, query port.RenderableListQuery) ([]domain.Renderable, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, query)
	}
	return []domain.Renderable{}, 0, nil
}

// Publish creates a new published renderable snapshot.
func (m renderableServiceMock) Publish(_ context.Context, _ string) (*domain.RenderableVersion, error) {
	return nil, errors.New("not implemented")
}

// ListVersions returns paginated published versions.
func (m renderableServiceMock) ListVersions(_ context.Context, _ string, _ int, _ int) ([]domain.RenderableVersion, int64, error) {
	return nil, 0, errors.New("not implemented")
}

// GetVersionByID loads one published version.
func (m renderableServiceMock) GetVersionByID(_ context.Context, _ string, _ string) (*domain.RenderableVersion, error) {
	return nil, errors.New("not implemented")
}

// Rollback creates a fresh published snapshot from one historical version.
func (m renderableServiceMock) Rollback(_ context.Context, _ string, _ string) (*domain.RenderableVersion, error) {
	return nil, errors.New("not implemented")
}

// pageServiceMock defines static-page handler dependencies for tests.
type pageServiceMock struct{}

// Create persists a new static page.
func (pageServiceMock) Create(_ context.Context, _ pageservice.CreateCommand) (*domain.StaticPage, error) {
	return nil, errors.New("not implemented")
}

// GetByID loads one static page.
func (pageServiceMock) GetByID(_ context.Context, _ string) (*domain.StaticPage, error) {
	return nil, pageservice.ErrStaticPageNotFound
}

// Update applies page mutations.
func (pageServiceMock) Update(_ context.Context, _ pageservice.UpdateCommand) (*domain.StaticPage, error) {
	return nil, errors.New("not implemented")
}

// Delete removes one static page.
func (pageServiceMock) Delete(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

// List returns paginated static pages.
func (pageServiceMock) List(_ context.Context, _ port.StaticPageListQuery) ([]domain.StaticPage, int64, error) {
	return []domain.StaticPage{}, 0, nil
}

// authorizerMock defines auth behavior for storefront handler tests.
type authorizerMock struct {
	// requireFn defines authorization behavior.
	requireFn func(ctx context.Context, header string, requiredPermissions ...string) error
}

// Require authenticates and authorizes requests using required permissions.
func (m authorizerMock) Require(ctx context.Context, header string, requiredPermissions ...string) error {
	if m.requireFn != nil {
		return m.requireFn(ctx, header, requiredPermissions...)
	}
	return nil
}

// IsUnauthorized reports authentication errors.
func (m authorizerMock) IsUnauthorized(err error) bool {
	return errors.Is(err, errUnauthorized)
}

// IsForbidden reports authorization errors.
func (m authorizerMock) IsForbidden(err error) bool {
	return errors.Is(err, errForbidden)
}

var (
	// errUnauthorized defines authentication failures for tests.
	errUnauthorized = errors.New("unauthorized")
	// errForbidden defines authorization failures for tests.
	errForbidden = errors.New("forbidden")
)

// TestNewHandlerRejectsNilServices verifies constructor validation behavior.
func TestNewHandlerRejectsNilServices(t *testing.T) {
	if _, err := NewHandler(nil, pageServiceMock{}); !errors.Is(err, ErrNilRenderableService) {
		t.Fatalf("NewHandler(nil, pageServiceMock{}) error = %v, want %v", err, ErrNilRenderableService)
	}
	if _, err := NewHandler(renderableServiceMock{}, nil); !errors.Is(err, ErrNilPageService) {
		t.Fatalf("NewHandler(renderableServiceMock{}, nil) error = %v, want %v", err, ErrNilPageService)
	}
}

// TestRegisterRoutesRequiresPermission verifies storefront routes enforce storefront:manage.
func TestRegisterRoutesRequiresPermission(t *testing.T) {
	handler, err := NewHandler(renderableServiceMock{}, pageServiceMock{}, authorizerMock{
		requireFn: func(ctx context.Context, header string, requiredPermissions ...string) error {
			if len(requiredPermissions) != 1 || requiredPermissions[0] != "storefront:manage" {
				t.Fatalf("requiredPermissions = %#v, want storefront:manage", requiredPermissions)
			}
			return errForbidden
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := newHTTPServerForHandler(t, handler)
	request, _ := http.NewRequest(http.MethodGet, "/storefront/renderable", nil)
	response := runRequest(t, server, request)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusForbidden)
	}
}

// TestCreateRenderableSuccess verifies request decoding and create behavior.
func TestCreateRenderableSuccess(t *testing.T) {
	var received renderableservice.CreateCommand
	handler, err := NewHandler(renderableServiceMock{
		createFn: func(ctx context.Context, cmd renderableservice.CreateCommand) (*domain.Renderable, error) {
			received = cmd
			return &domain.Renderable{ID: "renderable-1", Kind: cmd.Kind, Metadata: cmd.Metadata, Content: cmd.Content, Draft: true}, nil
		},
	}, pageServiceMock{}, authorizerMock{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := newHTTPServerForHandler(t, handler)
	request, _ := http.NewRequest(http.MethodPost, "/storefront/renderable", strings.NewReader(`{"kind":" static_page ","metadata":{"title":"About"},"content":{"body":"hello"}}`))
	request.Header.Set("Content-Type", "application/json")
	response := runRequest(t, server, request)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	if received.Kind != "static_page" {
		t.Fatalf("received.Kind = %q, want %q", received.Kind, "static_page")
	}

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if payload["id"] != "renderable-1" {
		t.Fatalf("payload.id = %v, want %q", payload["id"], "renderable-1")
	}
	if payload["kind"] != "static_page" {
		t.Fatalf("payload.kind = %v, want %q", payload["kind"], "static_page")
	}
}

// newHTTPServerForHandler creates servers for handler tests.
func newHTTPServerForHandler(t *testing.T, handler *Handler) *corehttp.Server {
	t.Helper()

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8166}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	return server
}

// runRequest runs HTTP requests against test servers.
func runRequest(t *testing.T, server *corehttp.Server, request *http.Request) *http.Response {
	t.Helper()

	response, err := server.App().Test(request)
	if err != nil {
		t.Fatalf("App().Test() error = %v", err)
	}

	return response
}
