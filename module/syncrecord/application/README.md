# Sync Record Application Package

Implements sync-run recording, querying, and cleanup use-cases.

## Key methods / endpoints / events
- Methods:
  - `NewService(...)`
  - `(*RecorderService).StartRun(...)`
  - `(*RecorderService).CompleteRun(...)`
  - `(*RecorderService).FailRun(...)`
  - `(*RecorderService).ListRuns(...)`
  - `(*RecorderService).CleanupExpired(...)`
- Endpoints: none.
- Events: none.
