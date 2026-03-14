# Membership Application Package

Implements stamp and status-query use-cases.

## Key methods / endpoints / events
- Methods:
  - `NewService(...)`
  - `(*MembershipService).Stamp(...)`
  - `(*MembershipService).GetStatus(...)`
  - `(*MembershipService).GetStatuses(...)`
  - `(*MembershipService).ListStamps(...)`
- Endpoints: none.
- Events:
  - `membership.v1.changed`
