# WooCommerce Contact Service Application Package

`application/contact/service` contains the contact sync use case orchestration for WooCommerce order ingestion.

## Responsibilities
- Validate integration availability before sync execution.
- Orchestrate paginated WooCommerce order retrieval.
- Map order billing fields into contact upsert commands.
- Perform concurrent contact upserts with run-wide email deduplication.
- Apply optional circuit-breaker fail-fast behavior for source and upsert dependencies.
- Emit integration lifecycle events through `application/contact/event`.

## Key Methods / Endpoints / Events
- Methods:
  - `service.NewService(cfg, source, target, publisher, logger, breakers...)`
  - `(*service.ContactSyncService).ValidateIntegration(ctx)`
  - `(*service.ContactSyncService).SyncContacts(ctx, trigger)`
- Endpoints: none in this package.
- Events: delegated to `application/contact/event`.
