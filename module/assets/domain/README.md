# assets/domain

Domain entities and invariants for assets, nested folders, tags, and metadata validation.

## Key methods / endpoints / events
- Methods: `(*Asset).Normalize()`, `(Asset).ValidateCreate()`, `ValidateID(id)`, `(*Folder).Normalize()`, `(Folder).ValidateCreate()`, `ValidateFolderID(id)`, `BuildFolderSlug(name)`
- Endpoints: consumed by `/assets*` and `/assets/folders*` handlers.
- Events: mapped by application layer to integration events.
