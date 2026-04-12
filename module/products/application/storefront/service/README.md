# products/application/storefront/service

Builds and caches the storefront navigation tree used by external storefront consumers.

## Key methods / endpoints / events
- `(*Service).Get(ctx)`
- `(*Service).Regenerate(ctx)`
- `(*Service).TriggerRefresh(ctx)`
- Consumed by `GET /storefront/navigation`
- No integration events are emitted yet.
