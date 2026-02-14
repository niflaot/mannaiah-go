# Contacts Application Package

`application` defines contact use cases and command/query orchestration.

## Key Methods / Endpoints / Events
- Methods:
  - `application.NewService(repository)`
  - `application.NewServiceWithPublisher(repository, publisher)`
  - `(*application.ContactService).Create(ctx, command)`
  - `(*application.ContactService).Get(ctx, id)`
  - `(*application.ContactService).List(ctx, query)`
  - `(*application.ContactService).Update(ctx, id, command)`
  - `(*application.ContactService).Delete(ctx, id)`
- Contact payload capability:
  - supports `metadata` (`map[string]string`) in create/update flows.
- Endpoints: none in this package.
- Events:
  - emits integration event `contacts.v1.created` on successful create
  - emits integration event `contacts.v1.updated` on successful update
