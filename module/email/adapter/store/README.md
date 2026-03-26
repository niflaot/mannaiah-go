# Email Store Adapter

GORM repository for email deliveries and status history.

## Key methods / endpoints / events
- Methods:
  - `NewRepository(db)`
  - `(*Repository).CreateDelivery(...)`
  - `(*Repository).UpdateDeliveryStatus(...)`
  - `(*Repository).AddStatusEntry(...)`
  - `(*Repository).ListByEmail(...)`
- Endpoints: none.
- Events: none.
