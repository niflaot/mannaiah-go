# core/storage

Provider-agnostic object storage interfaces and constructors.

## Key methods / endpoints / events
- Methods:
  - `storage.NewS3(cfg, logger)`
  - `storage.Disabled(reason)`
  - `(storage.Store).Upload(ctx, req)`
  - `(storage.Store).Delete(ctx, key)`
  - `(storage.Store).Exists(ctx, key)`
  - `(storage.Store).AvailabilityError()`
- Endpoints: none.
- Events: storage availability and failures are emitted through logger records.
