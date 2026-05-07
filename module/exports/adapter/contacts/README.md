# Exports Contacts Adapter

Adapts the contacts application service to the exports module contact source port.

## Key methods / endpoints / events
- `ListContacts` pages contacts into flattened CSV rows.
- Optional membership consent status values populate `membershipOptIn` and `membershipOptInAt`.
- Checker metadata values populate `privacyAccepted` and `privacyAcceptedAt`.
