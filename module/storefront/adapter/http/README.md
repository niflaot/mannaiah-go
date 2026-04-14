# storefront/adapter/http

HTTP adapter for storefront renderable and static-page management endpoints.

## Endpoints

- `GET /storefront/renderable`
- `POST /storefront/renderable`
- `GET /storefront/renderable/:id`
- `PATCH /storefront/renderable/:id`
- `DELETE /storefront/renderable/:id`
- `POST /storefront/renderable/:id/publish`
- `GET /storefront/renderable/:id/versions`
- `GET /storefront/renderable/:id/versions/:versionId`
- `POST /storefront/renderable/:id/versions/:versionId/rollback`
- `GET /storefront/page`
- `POST /storefront/page`
- `GET /storefront/page/:id`
- `PATCH /storefront/page/:id`
- `POST /storefront/page/:id/archive`
- `DELETE /storefront/page/:id`

## Notes

- `GET /storefront/page` defaults to active pages and accepts `archived=true|false` for explicit filtering.
- `POST /storefront/page/:id/archive` preserves renderable/version history while removing the page from active storefront listings.