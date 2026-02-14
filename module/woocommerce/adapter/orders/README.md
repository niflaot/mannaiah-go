# WooCommerce Orders Upserter Adapter Package

`adapter/orders` maps WooCommerce order-sync commands into the orders module application service while keeping WooCommerce-specific mapping outside the order domain.

## Responsibilities
- Ensure contacts exist before order upserts (via WooCommerce contacts upserter).
- Create orders on first sync by realm+identifier.
- Update status history for existing orders when status/comments change.
- Keep mapping logic normalized (status mapping, metadata normalization, shipping mapping).

## Key Methods / Endpoints / Events
- Methods:
  - `orders.NewUpserter(orderService, contactService)`
  - `(*orders.Upserter).UpsertByIdentifier(ctx, command)`
- Endpoints: none in this package.
- Events: none in this package.
