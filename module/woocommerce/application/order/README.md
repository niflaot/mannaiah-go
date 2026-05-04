# WooCommerce Order Application Namespace

`application/order` is a namespace package used to group order-related WooCommerce use cases.

## Responsibilities
- Provide a stable package boundary for order application features.
- Delegate concrete use-case behavior to child packages.

## Child Packages
- `application/order/service`: order sync use case orchestration.
- `application/order/event`: order sync integration event contracts/builders.

## Key Methods / Endpoints / Events
- Methods: none on this namespace package.
- Endpoints: none in this package.
- Events: defined by child packages.
