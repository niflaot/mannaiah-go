# Email Application Package

Implements send, SNS webhook status update, and complaint/bounce handling use-cases.

## Key methods / endpoints / events
- Methods:
  - `NewService(repository, provider, membershipStamper...)`
  - `(*EmailService).Send(...)`
  - `(*EmailService).HandleWebhook(...)`
- Endpoints: none.
- Events: none.
