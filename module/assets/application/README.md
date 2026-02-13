# assets/application

Application services for asset upload, metadata lifecycle, and integration event emission.

## Key methods / endpoints / events
- Methods: `NewService(repository, storage, publishers...)`, `(*AssetService).Create`, `(*AssetService).Get`, `(*AssetService).List`, `(*AssetService).Update`, `(*AssetService).Delete`, `(*AssetService).Exists`, `(*AssetService).CreateFolder`, `(*AssetService).GetFolder`, `(*AssetService).ListFolders`, `(*AssetService).UpdateFolder`, `(*AssetService).DeleteFolder`
- Endpoints: consumed by module HTTP adapter (`/assets*`, `/assets/folders*`).
- Events: `assets.v1.created`, `assets.v1.updated`, `assets.v1.deleted`, `asset_folders.v1.created`, `asset_folders.v1.updated`, `asset_folders.v1.deleted`.
