# Watermill Publisher Package

`watermill/publisher` adapts Watermill publisher implementations to `bus.Publisher`.

## Responsibilities
- Validate outbound integration event envelope basics.
- Map `bus.Message` into Watermill messages.
- Propagate/generate correlation metadata and ensure `event_id` metadata.

## Key Methods / Endpoints / Events
- Methods:
  - `publisher.NewAdapter(publisher)`
  - `(*publisher.Adapter).Publish(ctx, msg)`
- Endpoints: none in this package.
- Events: publishes integration event messages to configured topics.
