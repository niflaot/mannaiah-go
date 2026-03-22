# Email SES Adapter

AWS SES adapter implementing outbound provider and SNS signature-verification ports.

## Key methods / endpoints / events
- Methods:
  - `NewProvider(ctx, cfg)`
  - `(*Provider).Send(...)`
  - `NewSNSMessageVerifier(cfg)`
- Endpoints: none.
- Events: none.
