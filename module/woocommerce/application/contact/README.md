# WooCommerce Contact Application Namespace

`application/contact` is a namespace package used to group contact-related WooCommerce use cases.

## Responsibilities
- Provide a stable package boundary for contact application features.
- Delegate concrete use-case behavior to child packages.

## Child Packages
- `application/contact/service`: contact sync use case orchestration.
- `application/contact/event`: contact sync integration event contracts/builders.

## Key Methods / Endpoints / Events
- Methods: none on this namespace package.
- Endpoints: none in this package.
- Events: defined by child packages.
