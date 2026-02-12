# assets/adapter/store

GORM-backed persistence adapter for asset metadata.

## Key methods / endpoints / events
- Methods: `NewRepository(db)`, `(*Repository).EnsureSchema`, `(*Repository).Create`, `(*Repository).GetByID`, `(*Repository).List`, `(*Repository).UpdateName`, `(*Repository).SoftDelete`
- Endpoints: supports `/assets` query/CRUD handlers.
- Events: none directly (events are emitted by application service).
