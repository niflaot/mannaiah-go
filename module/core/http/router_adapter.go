package http

import "github.com/gofiber/fiber/v2"

// fiberRouterAdapter adapts Fiber router registration to abstract router contracts.
type fiberRouterAdapter struct {
	// router is the wrapped Fiber router.
	router fiber.Router
}

// newFiberRouterAdapter creates a router adapter over a Fiber router.
func newFiberRouterAdapter(router fiber.Router) Router {
	return &fiberRouterAdapter{router: router}
}

// Get registers a GET route handler.
func (a *fiberRouterAdapter) Get(path string, handler Handler) {
	a.router.Get(path, adaptHandler(handler))
}

// Post registers a POST route handler.
func (a *fiberRouterAdapter) Post(path string, handler Handler) {
	a.router.Post(path, adaptHandler(handler))
}

// Put registers a PUT route handler.
func (a *fiberRouterAdapter) Put(path string, handler Handler) {
	a.router.Put(path, adaptHandler(handler))
}

// Patch registers a PATCH route handler.
func (a *fiberRouterAdapter) Patch(path string, handler Handler) {
	a.router.Patch(path, adaptHandler(handler))
}

// Delete registers a DELETE route handler.
func (a *fiberRouterAdapter) Delete(path string, handler Handler) {
	a.router.Delete(path, adaptHandler(handler))
}

// adaptHandler wraps abstract handlers into Fiber handlers.
func adaptHandler(handler Handler) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if handler == nil {
			return nil
		}

		return handler(&fiberContextAdapter{ctx: ctx})
	}
}
