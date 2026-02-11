# WooCommerce Contacts Adapter Package

`adapter/contacts` bridges WooCommerce sync commands into contacts application service operations.

## Responsibilities
- Resolve contacts by email.
- Create contacts when missing.
- Update contacts when present.
- Handle duplicate-create races through find-and-update fallback.

## Key Methods / Endpoints / Events
- Methods:
  - `contacts.NewUpserter(service)`
  - `(*contacts.Upserter).UpsertByEmail(ctx, command)`
- Endpoints: none in this package.
- Events: none in this package.
