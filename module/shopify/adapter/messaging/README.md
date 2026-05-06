# Shopify Messaging Adapter

This package listens to mainstream order integration events and dispatches outbound Shopify status updates when linked Shopify orders exist.

## Key methods / endpoints / events
- `messaging.NewOrderConsumer(handler, logger)`
- `(*OrderConsumer).Register(registrar)`
- `(*OrderConsumer).handleMessage(ctx, topic, message)`
