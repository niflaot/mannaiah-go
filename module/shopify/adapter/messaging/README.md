# Shopify Messaging Adapter

This package listens to mainstream contact and order integration events and dispatches outbound Shopify updates when linked aggregates exist.

## Key methods / endpoints / events
- `messaging.NewContactConsumer(handler, logger)`
- `(*ContactConsumer).Register(registrar)`
- `(*ContactConsumer).handleMessage(ctx, topic, message)`
- `messaging.NewOrderConsumer(handler, logger)`
- `(*OrderConsumer).Register(registrar)`
- `(*OrderConsumer).handleMessage(ctx, topic, message)`
