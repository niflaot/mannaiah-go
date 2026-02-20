# Sync Status Service

Provides use-case orchestration for Falabella async feed status resolution.

## Key methods

- `RecordEntry` — persists a new sync status entry after product sync submission.
- `GetExecutionByID` — retrieves one parent sync execution by identifier.
- `GetByExecutionID` — retrieves child feed rows for one execution identifier.
- `GetByFeedID` — retrieves a sync status entry by Falabella feed identifier.
- `GetByProductID` — retrieves sync status entries by source product identifier.
- `ResolveFeedStatus` — queries Falabella feed status API and updates the entry resolution.
- `ResolvePendingFeeds` — batch-resolves all pending feed entries by querying the Falabella FeedStatus API (used by cron).
