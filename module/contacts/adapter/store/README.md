# Contacts Store Adapter Package

`adapter/store` provides GORM persistence for contacts.

Uniqueness constraints are enforced at the database layer for high-performance, race-safe writes:
- unique `email`
- unique normalized `(documentType, documentNumber)` when both values are present

Metadata is persisted in normalized relation storage:
- `contacts` root table
- `contact_metadata` key-value table (`contact_id`, `key`, `value`)

## Key Methods / Endpoints / Events
- Methods:
  - `store.NewRepository(db)`
  - `(*store.Repository).EnsureSchema(ctx)`
  - `(*store.Repository).Create(ctx, contact)`
  - `(*store.Repository).GetByID(ctx, id)`
  - `(*store.Repository).List(ctx, query)`
  - `(*store.Repository).Update(ctx, contact)`
  - `(*store.Repository).Delete(ctx, id)`
  - Benchmarks: `BenchmarkRepositoryCreate`, `BenchmarkRepositoryList`
- Endpoints: none in this package.
- Events: none directly emitted; persistence changes are consumed by application-level event emission.
