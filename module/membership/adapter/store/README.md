# Membership Store Adapter

GORM repository for membership stamps and stamp-latest status resolution.

## Key methods / endpoints / events
- Methods:
  - `NewRepository(db)`
  - `(*Repository).SaveStamp(...)`
  - `(*Repository).GetStatus(...)`
  - `(*Repository).GetStatuses(...)`
  - `(*Repository).ListStamps(...)`
- Endpoints: none.
- Events: none.
