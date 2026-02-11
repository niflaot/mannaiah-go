# WooCommerce Contact Sync Application Package

`application/contact` contains contact-related WooCommerce use cases and transport-agnostic event mapping for WooCommerce order ingestion.

## Responsibilities
- Validate integration availability before sync execution.
- Orchestrate paginated WooCommerce order retrieval.
- Map order billing fields into contact upsert commands.
- Perform concurrent contact upserts with run-wide email deduplication.
- Emit integration lifecycle events (`started`, `completed`, `failed`).

## Key Methods / Endpoints / Events
- Methods:
  - `contact.NewService(cfg, source, target, publisher, logger)`
  - `(*contact.ContactSyncService).ValidateIntegration(ctx)`
  - `(*contact.ContactSyncService).SyncContacts(ctx, trigger)`
- Endpoints: none in this package.
- Events:
  - `woocommerce.v1.contacts.sync.started`
  - `woocommerce.v1.contacts.sync.completed`
  - `woocommerce.v1.contacts.sync.failed`
