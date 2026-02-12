# core/storage/s3

AWS SDK v2 S3 implementation for object storage.

## Key methods / endpoints / events
- Methods:
  - `s3.New(cfg, logger)`
  - `s3.Disabled(reason)`
  - `(*Client).Upload(ctx, request)`
  - `(*Client).Delete(ctx, key)`
  - `(*Client).Exists(ctx, key)`
  - `(*Client).AvailabilityError()`
- Endpoints: none.
- Events: logs include disabled-state and operation failures.
