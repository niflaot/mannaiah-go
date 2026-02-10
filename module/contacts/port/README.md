# Contacts Port Package

`port` defines repository contracts used by application use cases.

## Key Methods / Endpoints / Events
- Methods:
  - `port.Repository.Create(ctx, contact)`
  - `port.Repository.GetByID(ctx, id)`
  - `port.Repository.List(ctx, query)`
  - `port.Repository.Update(ctx, contact)`
  - `port.Repository.Delete(ctx, id)`
  - `port.IntegrationEventPublisher.Publish(ctx, event)`
- Endpoints: none in this package.
- Events: defines technology-neutral integration event contracts used by application and adapters.
