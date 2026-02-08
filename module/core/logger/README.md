# Logger Package

`logger` provides centralized Zap logger construction for core and future modules.

## Features
- Builds configurable loggers from `LOG_FORMAT` and `LOG_LEVEL`.
- Supports `pretty` (console) and `json` output formats.
- Supports logger injection via `Resolve` so callers can keep an existing logger.
- Exposes writer-based constructor for unit tests and custom sinks.

## Usage Rules
- Use `Settings` loaded from core configuration.
- Use `Resolve` when callers may pass an already initialized logger.

## Key Methods / Endpoints / Events
- Methods:
  - `logger.New(settings)`
  - `logger.NewWithWriters(settings, output, errorOutput)`
  - `logger.Resolve(provided, settings)`
- Endpoints: none in this package.
- Events: log records are emitted according to configured encoder format and level threshold.
