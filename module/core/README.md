# Core Module

`module/core` provides shared runtime capabilities for all future modules.

## Packages
- `config`: Viper-based startup configuration loading and validation from `.env` and environment variables.
- `logger`: Zap logger construction and resolution helpers.

## Goals
- Deterministic startup configuration rules.
- Strict required-field validation with startup error logs.
- Reusable logging primitives with configurable format and level.

## Key Methods / Endpoints / Events
- Methods:
  - `config.NewLoader(envFile, startupLogger)`
  - `config.Load(envFile, startupLogger, targets...)`
  - `(*config.Loader).Load(targets...)`
  - `logger.New(settings)`
  - `logger.NewWithWriters(settings, output, errorOutput)`
  - `logger.Resolve(provided, settings)`
- Endpoints: none in this module.
- Events: startup validation errors are emitted through Zap logger records.
