# Orders Store Adapter Package

`module/orders/adapter/store` persists normalized orders with relational child tables.

## Key Methods / Endpoints / Events
- Methods:
  - `store.NewRepository(db)`
  - `(*store.Repository).EnsureSchema(ctx)`
  - `(*store.Repository).Create(ctx, order)`
  - `(*store.Repository).Update(ctx, order)`
  - `(*store.Repository).GetByID(ctx, id)`
  - `(*store.Repository).List(ctx, query)`
  - `(*store.Repository).AppendStatus(ctx, id, entry)`
- Endpoints: none in this package.
- Events: none in this package.
