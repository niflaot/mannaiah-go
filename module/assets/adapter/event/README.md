# assets/adapter/event

Messaging adapter for publishing asset integration events onto core messaging bus.

## Key methods / endpoints / events
- Methods: `NewPublisher(busPublisher)`, `(*Publisher).Publish`
- Endpoints: none.
- Events: `assets.v1.created`, `assets.v1.updated`, `assets.v1.deleted`.
