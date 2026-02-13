# assets/adapter/store

GORM-backed persistence adapter for asset metadata.

## Key methods / endpoints / events
- Methods: `NewRepository(db)`, `(*Repository).EnsureSchema`, `(*Repository).Create`, `(*Repository).GetByID`, `(*Repository).List`, `(*Repository).Update`, `(*Repository).SoftDelete`, `(*Repository).CreateFolder`, `(*Repository).GetFolderByID`, `(*Repository).ListFolders`, `(*Repository).UpdateFolder`, `(*Repository).SoftDeleteFolder`, `(*Repository).ExistsFolder`
- Endpoints: supports `/assets*` and `/assets/folders*` handlers.
- Events: none directly (events are emitted by application service).
