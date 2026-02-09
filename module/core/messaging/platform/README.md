# Messaging Platform Package

`platform` provides messaging runtime configuration and infrastructure-level error markers.

## Features
- Configuration model for retry/backoff, DLQ suffix, and in-memory buffer size.
- Non-retriable error marker utilities for handler retry classification.

## Usage Rules
- Use `platform.Config` as the transport-neutral messaging runtime config.
- Mark business-validation failures with `platform.NonRetriable(err)` in handlers to bypass retries.

## Key Methods / Endpoints / Events
- Methods:
  - `platform.Config.Normalized()`
  - `platform.NonRetriable(err)`
  - `platform.IsNonRetriable(err)`
- Endpoints: none in this package.
- Events: defines runtime retry classification behavior only.
