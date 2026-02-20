# Falabella Client Adapter Package

Outbound Falabella API adapter backed by direct HTTP calls and Seller Center signing profiles.

## Key methods / endpoints / events
- Methods: `falabella.NewClient`, `(*falabella.Client).Validate`, `(*falabella.Client).GetBrands`, `(*falabella.Client).SyncProduct`, `(*falabella.Client).SyncProductImages`, `(*falabella.Client).GetFeedStatus`
- Endpoints: none
- Events: none

## Startup Validation
- `Validate` performs a config-only check and does not issue outbound API calls. Configuration (URL, UserID, APIKey) is validated at construction time by `NewClient`.

## Debug Logging
- `falabella request attempt` debug entries include:
- `request_query_params`: decoded signed query parameters actually sent (`Action`, `Format` when present, `Timestamp`, `UserID`, `Version`, `Signature`).
- `request_body_params`: parsed outbound body parameters for XML/JSON requests.
- `Description` fields are intentionally excluded from `request_body_params` to keep logs focused and reduce noisy payload text.

## Signing Strategy
- Timestamp serialization is fixed to `YYYY-MM-DDTHH:MM:SS+0000` (example: `2026-02-18T04:47:31+0000`).
- HMAC calculation uses dynamic fallback profiles:
- `rfc3986` canonical input (URL-encoded values), lower/upper hex signature.
- `raw` canonical input (unencoded values), lower/upper hex signature.
- Outbound query strings remain RFC3986-encoded and append `Signature` as the final parameter.
