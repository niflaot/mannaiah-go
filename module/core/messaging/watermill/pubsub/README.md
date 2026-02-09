# Watermill PubSub Package

`watermill/pubsub` builds in-memory Watermill pubsub resources.

## Responsibilities
- Build GoChannel pubsub with normalized runtime settings.
- Configure ack behavior suitable for retry/DLQ flow.
- Use Zap-backed Watermill logging.

## Key Methods / Endpoints / Events
- Methods:
  - `pubsub.NewGoChannel(cfg, providedLogger)`
- Endpoints: none in this package.
- Events: none emitted directly by this package.
