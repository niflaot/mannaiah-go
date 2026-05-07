# Exports Module

The exports module generates CSV reports from runtime data and stores report objects in the configured MinIO/S3 storage backend.

## Key methods / endpoints / events

- `POST /exports/contacts` generates a contact CSV export and stores it under `exports/contacts/`.
- `POST /exports/orders` generates an order CSV export and stores it under `exports/orders/`.
- `POST /export/orders` is a compatibility alias for the order export endpoint.
- `GET /exports/reports` lists generated export report registry entries.
- `GET /exports/search?type=contacts|orders` filters generated report history by report type.
- `GET /exports/reports/:id` returns one report registry entry.

Reports are stored with deterministic timestamp stamps and SHA-256 hashes in their storage key. The registry stores report type, storage key, hash, row count, byte size, and generation timestamps.
