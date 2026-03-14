# Sync Record Store Adapter

GORM repository for `sync_runs` and `sync_run_errors` tables.

## Key methods / endpoints / events
- Methods:
  - `NewRepository(db)`
  - `(*Repository).CreateRun(...)`
  - `(*Repository).CompleteRun(...)`
  - `(*Repository).AddRunErrors(...)`
  - `(*Repository).ListRuns(...)`
  - `(*Repository).CleanupBefore(...)`
- Endpoints: none.
- Events: none.
