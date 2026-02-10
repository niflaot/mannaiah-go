# Contacts Store Adapter Package

`adapter/store` provides GORM persistence for contacts.

## Key Methods / Endpoints / Events
- Methods:
  - `store.NewRepository(db)`
  - `(*store.Repository).EnsureSchema(ctx)`
  - `(*store.Repository).Create(ctx, contact)`
  - `(*store.Repository).GetByID(ctx, id)`
  - `(*store.Repository).List(ctx, query)`
  - `(*store.Repository).Update(ctx, contact)`
  - `(*store.Repository).Delete(ctx, id)`
- Endpoints: none in this package.
- Events: none directly emitted; persistence changes are consumed by application-level event emission.
