# Storage

The `storage` package provides file object storage backed by an S3-compatible API.

## Store Interface

| Method | Description |
|--------|-------------|
| `Upload(ctx, key, reader, size)` | Upload an object |
| `Download(ctx, key)` | Download an object as a stream |
| `Delete(ctx, key)` | Delete an object by key |
| `Exists(ctx, key)` | Check whether an object exists |
| `AvailabilityError()` | Returns a non-nil error if the backend is unavailable |

## Disabled Stub

When `STORAGE_ENABLED=false`, the runtime wires in a `Disabled` stub that satisfies the interface
and returns `ErrUnavailable` for every operation. Modules that require storage should call
`AvailabilityError()` before attempting operations and return a controlled `503` response when the
result is non-nil.

## Circuit Breaker

All S3 operations are wrapped with a configurable circuit breaker. When the backend is unhealthy
the circuit opens and operations return `ErrUnavailable` immediately without waiting for the full
HTTP timeout.

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `STORAGE_ENABLED` | | `true` | Enable object storage |
| `STORAGE_ENDPOINT` | | _(empty)_ | S3-compatible endpoint URL |
| `STORAGE_REGION` | | `us-east-1` | Bucket region |
| `STORAGE_BUCKET_NAME` | | _(empty)_ | Target bucket |
| `STORAGE_ACCESS_KEY` | ✓ | — | Access key ID |
| `STORAGE_SECRET_KEY` | ✓ | — | Secret access key |
| `STORAGE_FORCE_PATH_STYLE` | | `false` | Use path-style addressing (required for MinIO) |
| `STORAGE_UPLOAD_TIMEOUT_MS` | | `30000` | Upload operation timeout |
| `STORAGE_DOWNLOAD_TIMEOUT_MS` | | `30000` | Download operation timeout |
| `STORAGE_CIRCUIT_BREAKER_ENABLED` | | `true` | Enable circuit breaker |
| `STORAGE_CIRCUIT_BREAKER_MAX_REQUESTS` | | `1` | Half-open trial requests |
| `STORAGE_CIRCUIT_BREAKER_INTERVAL_MS` | | `60000` | Closed-state counter reset window |
| `STORAGE_CIRCUIT_BREAKER_TIMEOUT_MS` | | `30000` | Open-state cooldown window |
| `STORAGE_CIRCUIT_BREAKER_FAILURE_THRESHOLD` | | `5` | Failures before breaker opens |
