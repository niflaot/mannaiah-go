# WooCommerce SDK Adapter Package

`adapter/woocommerce` integrates the maintained WooCommerce SDK (`github.com/jmolboy/woocommerce-go`) behind module ports.

## Responsibilities
- Validate WooCommerce connectivity and credentials.
- Retrieve paginated orders.
- Map SDK entities into port-level order contracts (status, billing/shipping snapshot, items with value, shipping charges, comments).
- Use tolerant raw-response fallback decoding when SDK strict decoding fails on non-scalar metadata.

## Key Methods / Endpoints / Events
- Methods:
  - `woocommerce.NewClient(cfg)`
  - `(*woocommerce.Client).Validate(ctx)`
  - `(*woocommerce.Client).ListOrders(ctx, page, pageSize)`
  - `(*woocommerce.Client).SearchOrders(ctx, search, page, pageSize)`
  - `(*woocommerce.Client).GetOrderByID(ctx, orderID)`
- Endpoints: none in this package.
- Events: none in this package.
