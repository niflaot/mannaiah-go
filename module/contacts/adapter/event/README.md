# Contacts Event Adapter Package

`adapter/event` maps contact integration events to core messaging bus envelopes.

## Key Methods / Endpoints / Events
- Methods:
  - `event.NewPublisher(publisher)`
  - `(*event.Publisher).Publish(ctx, event)`
- Endpoints: none in this package.
- Events:
  - publishes `contacts.v1.created`
  - publishes `contacts.v1.updated`
