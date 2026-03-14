# Sync Record Module

Centralized synchronization execution registry for all modules.

## Key methods / endpoints / events
- Methods:
  - `Module.Recorder()`
  - `Module.Service()`
- Endpoints:
  - `GET /syncrecord/runs`
  - `GET /syncrecord/stats`
- Events: none.

## Purpose
- Persist sync run envelopes (`running`, `completed`, `failed`).
- Persist normalized run errors in child rows.
- Provide query and operational stats endpoints.
- Support retention cleanup by cron.
