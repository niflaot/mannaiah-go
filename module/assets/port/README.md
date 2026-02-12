# assets/port

Port contracts for asset metadata persistence, binary storage, and integration events.

## Key methods / endpoints / events
- Methods:
  - `Repository.EnsureSchema`, `Repository.Create`, `Repository.GetByID`, `Repository.List`, `Repository.UpdateName`, `Repository.SoftDelete`
  - `Storage.Upload`, `Storage.Delete`, `Storage.Exists`, `Storage.AvailabilityError`
  - `IntegrationEventPublisher.Publish`
- Endpoints: used by `/assets` endpoints.
- Events: `assets.v1.created`, `assets.v1.updated`, `assets.v1.deleted`.
