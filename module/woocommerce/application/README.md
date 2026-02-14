# WooCommerce Application Namespace

`application` organizes WooCommerce use cases by feature-focused subpackages to keep growth manageable as integrations expand.

## Responsibilities
- Provide a stable namespace for feature use cases.
- Keep each use case isolated in a dedicated child package.

## Feature Packages
- `application/contact`: contact feature namespace package.
- `application/contact/service`: contact-related WooCommerce sync use case orchestration.
- `application/contact/event`: contact-related WooCommerce integration event contracts/builders.
- `application/order`: order feature namespace package.
- `application/order/service`: order-related WooCommerce sync use case orchestration.
- `application/order/event`: order-related WooCommerce integration event contracts/builders.

## Key Methods / Endpoints / Events
- Methods:
  - none on this namespace package.
- Endpoints: none in this package.
- Events: defined by child feature packages.
