# WooCommerce Event Adapter Package

`adapter/event` publishes WooCommerce integration events through the core messaging bus.

## Responsibilities
- Serialize payloads.
- Map metadata to core bus metadata keys.
- Publish topic-based integration messages.

## Key Methods / Endpoints / Events
- Methods:
  - `event.NewPublisher(publisher)`
  - `(*event.Publisher).Publish(ctx, event)`
- Endpoints: none in this package.
- Events: publishes WooCommerce sync lifecycle integration events.
