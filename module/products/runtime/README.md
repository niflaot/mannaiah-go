# products/runtime

Module composition root for product wiring and OpenAPI artifact exposure.

## Key methods / endpoints / events
- Methods: `runtime.New(db, assetLookup)`, `runtime.NewWithConfig(db, assetLookup, cfg, cacheStore, logger)`, `(*runtime.Module).RegisterRoutes(router)`, `(*runtime.Module).Load(loader)`, `(*runtime.Module).ConfigureScheduler(scheduler)`, `(*runtime.Module).Start(ctx)`, `(*runtime.Module).Stop(ctx)`, `runtime.OpenAPISpec()`
- Endpoints: `/products`, `/products/:id`, `/variations`, `/variations/:id`, `/storefront/navigation`
- Events: no integration events are emitted yet.
