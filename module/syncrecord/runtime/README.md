# Sync Record Runtime Package

Composition root for sync record wiring, HTTP loading, and retention cleanup lifecycle.

## Key methods / endpoints / events
- Methods:
  - `New(cfg, db, scheduler...)`
  - `(*Module).Load(loader)`
  - `(*Module).Start(ctx)`
  - `(*Module).Stop(ctx)`
  - `(*Module).Recorder()`
- Endpoints:
  - `GET /syncrecord/runs`
  - `GET /syncrecord/runs/:id`
  - `GET /syncrecord/stats`
- Events: none.
