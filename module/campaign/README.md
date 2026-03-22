# Campaign Module

Campaign planning and asynchronous audience send orchestration.

## Key methods / endpoints / events
- Methods:
  - `Module.Service()`
- Endpoints:
  - `POST /campaigns`
  - `GET /campaigns`
  - `GET /campaigns/:id`
  - `PATCH /campaigns/:id`
  - `DELETE /campaigns/:id`
  - `POST /campaigns/:id/send`
  - `POST /campaigns/:id/test`
- Events:
  - publishes `campaign.v1.delivery` for per-recipient send outcomes.
