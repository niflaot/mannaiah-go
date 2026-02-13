# assets/adapter/event

Messaging adapter for publishing asset integration events onto core messaging bus.

## Key methods / endpoints / events
- Methods: `NewPublisher(busPublisher)`, `(*Publisher).Publish`
- Endpoints: none.
- Events: `assets.v1.created`, `assets.v1.updated`, `assets.v1.deleted`, `asset_folders.v1.created`, `asset_folders.v1.updated`, `asset_folders.v1.deleted`.
