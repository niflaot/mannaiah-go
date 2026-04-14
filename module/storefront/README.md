# Storefront Module

Provides storefront content management APIs for reusable renderables and their first bound child resource, static pages.

## Key methods / endpoints / events

### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/storefront/renderable` | Create a renderable draft |
| `GET` | `/storefront/renderable` | List renderables |
| `GET` | `/storefront/renderable/:id` | Get a renderable |
| `PATCH` | `/storefront/renderable/:id` | Update a renderable draft |
| `DELETE` | `/storefront/renderable/:id` | Delete a renderable and bound static page/version history |
| `POST` | `/storefront/renderable/:id/publish` | Publish the current renderable snapshot |
| `GET` | `/storefront/renderable/:id/versions` | List published renderable versions |
| `GET` | `/storefront/renderable/:id/versions/:versionId` | Get one published renderable version |
| `POST` | `/storefront/renderable/:id/versions/:versionId/rollback` | Roll back to a published version by creating a new published snapshot |
| `POST` | `/storefront/page` | Create a static page bound to a renderable |
| `GET` | `/storefront/page` | List static pages, defaulting to active rows and supporting `archived` filtering |
| `GET` | `/storefront/page/:id` | Get a static page |
| `PATCH` | `/storefront/page/:id` | Update a static page binding or metadata |
| `POST` | `/storefront/page/:id/archive` | Archive a static page without deleting bound renderable history |
| `DELETE` | `/storefront/page/:id` | Delete a static page |

### Integration Events

- No integration events are emitted yet.

## Domain concepts

- **Renderable**: Reusable storefront content unit storing `kind`, metadata JSON, content JSON, and draft state.
- **Published version**: Timestamped immutable snapshot of a renderable captured only when publishing or rolling back.
- **Rollback**: Copies one existing published version into a fresh published version with a new timestamp.
- **Static page**: Child resource storing title, URL, SEO-tags JSON, optional archive timestamp, and a one-to-one renderable binding.
- **Archive**: Soft-removes a static page from active listings/navigation while preserving renderable/version history.

## Performance notes

- Draft edits are stored only on the renderable root row; version rows are created only for published snapshots and rollbacks.
- JSON payloads are compacted before persistence to reduce storage overhead and improve hash comparisons.
- Version listing uses indexed `(renderable_id, published_at)` ordering for efficient history retrieval.
- Static pages use indexed unique URL and renderable bindings for fast lookup and conflict detection.
- Static page archive filtering uses an indexed `archived_at` column to keep active and archived listings efficient.