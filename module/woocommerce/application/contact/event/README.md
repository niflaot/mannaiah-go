# WooCommerce Contact Event Application Package

`application/contact/event` defines technology-neutral integration event contracts and builders for WooCommerce contact sync lifecycle notifications.

## Responsibilities
- Define contact sync integration event topics.
- Define event payload contracts for sync lifecycle notifications.
- Build event envelopes for `started`, `completed`, and `failed` states.
- Provide no-op publisher fallback resolution.

## Key Methods / Endpoints / Events
- Methods:
  - `event.ResolvePublisher(publisher)`
  - `event.NewSyncStartedEvent(trigger)`
  - `event.NewSyncCompletedEvent(summary)`
  - `event.NewSyncFailedEvent(summary, syncErr)`
- Endpoints: none in this package.
- Events:
  - `woocommerce.v1.contacts.sync.started`
  - `woocommerce.v1.contacts.sync.completed`
  - `woocommerce.v1.contacts.sync.failed`
