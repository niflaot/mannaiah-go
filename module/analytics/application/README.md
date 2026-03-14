# Analytics Application Package

Implements analytics status, seed, and resolver use-cases.

## Key methods / endpoints / events
- Methods:
  - `NewService(enabled, db, store)`
  - `(*AnalyticsService).Status(...)`
  - `(*AnalyticsService).Seed(...)`
  - `(*AnalyticsService).ResolveContacts(...)`
  - `(*AnalyticsService).CountContacts(...)`
  - `(*AnalyticsService).Ingest*...` (contacts/orders/membership/campaign)
- Endpoints: none.
- Events: none.
