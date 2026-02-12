# assets/application

Application services for asset upload, metadata lifecycle, and integration event emission.

## Key methods / endpoints / events
- Methods: `NewService(repository, storage, publishers...)`, `(*AssetService).Create`, `(*AssetService).Get`, `(*AssetService).List`, `(*AssetService).UpdateName`, `(*AssetService).Delete`, `(*AssetService).Exists`
- Endpoints: consumed by module HTTP adapter (`/assets`, `/assets/:id`).
- Events: `assets.v1.created`, `assets.v1.updated`, `assets.v1.deleted`.
