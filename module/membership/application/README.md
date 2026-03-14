# Membership Application Package

Implements stamp, status-query, and migration use-cases.

## Key methods / endpoints / events
- Methods:
  - `NewService(...)`
  - `(*MembershipService).Stamp(...)`
  - `(*MembershipService).GetStatus(...)`
  - `(*MembershipService).ListStamps(...)`
  - `(*MembershipService).MigrateFromContactMetadata(...)`
- Endpoints: none.
- Events:
  - `membership.v1.changed`
