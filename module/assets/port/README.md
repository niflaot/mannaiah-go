# assets/port

Port contracts for asset metadata persistence, nested-folder hierarchy operations, binary storage, and integration events.

## Key methods / endpoints / events
- Methods:
  - `Repository.EnsureSchema`, `Repository.Create`, `Repository.GetByID`, `Repository.List`, `Repository.Update`, `Repository.UpdateBinary`, `Repository.ListByTagNames`, `Repository.SoftDelete`
  - `Repository.CreateFolder`, `Repository.GetFolderByID`, `Repository.ListFolders`, `Repository.ListAllFolders`, `Repository.UpdateFolder`, `Repository.SoftDeleteFolder`, `Repository.ExistsFolder`
  - `Storage.Upload`, `Storage.Download`, `Storage.Delete`, `Storage.Exists`, `Storage.AvailabilityError`
  - `IntegrationEventPublisher.Publish`
- Endpoints: used by `/assets*`, `/assets/folders*`, and `/assets/folders/tree` endpoints.
- Events: `assets.v1.created`, `assets.v1.updated`, `assets.v1.deleted`, `asset_folders.v1.created`, `asset_folders.v1.updated`, `asset_folders.v1.deleted`.
