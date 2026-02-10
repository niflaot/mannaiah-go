# Contacts Domain Package

`domain` defines contact entities and domain invariants.

## Key Methods / Endpoints / Events
- Methods:
  - `(domain.Contact).Validate()`
  - `domain.NewContactCreatedEvent(contact)`
  - `domain.NewContactUpdatedEvent(contact)`
- Endpoints: none in this package.
- Events:
  - `contacts.contact.created`
  - `contacts.contact.updated`
