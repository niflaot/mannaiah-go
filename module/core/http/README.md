# HTTP Package

`http` provides a Fiber-based HTTP server wrapper with config-driven address resolution and Zap request logging.

## Features
- Fiber server initialization with HTTP config defaults.
- Host resolution from core config when available (`CORE_HOST` is authoritative for configuration-driven startup).
- Port resolution from core config when available (`CORE_PORT` is authoritative for configuration-driven startup).
- Zap logger integration through `zapfiber` middleware.
- Route registration and route-group mounting APIs for future modules.
- Abstract router/context interfaces (`http.Router`, `http.Context`) to decouple module route code from Fiber internals, including request header access via `http.Context.GetHeader`.

## Usage Rules
- Load `http.Config` using the shared configuration loader.
- Prefer `NewWithCore` when composing runtime startup with core config.
- Use `CORE_HOST` and `CORE_PORT` as the single host/port environment variables in startup flows.
- `http.Config.Host` and `http.Config.Port` are code-level only and are not loaded from environment variables.
- Prefer `RegisterRoutes` and `MountRoutes` for provider-agnostic route registration.
- Use `Register` and `Mount` only when direct Fiber APIs are required.

## Key Methods / Endpoints / Events
- Methods:
  - `http.New(cfg, providedLogger)`
  - `http.NewWithCore(cfg, coreCfg, providedLogger)`
  - `http.AddressFrom(cfg, coreCfg)`
  - `http.NewAppError(status, message, cause)`
  - `(*http.Server).RegisterRoutes(register)`
  - `(*http.Server).MountRoutes(prefix, register)`
  - `(*http.Server).Register(register)`
  - `(*http.Server).Mount(prefix, register)`
  - `(*http.Server).Start()`
  - `(*http.Server).StartWithListener(listener)`
  - `(*http.Server).Shutdown(ctx)`
- Endpoints: none in this package.
- Events:
  - every response includes `X-Ray-ID` tracing header
  - all handler errors are mapped to JSON payload format: `{"message":"...","error":"..."}`
  - HTTP request logs are emitted through zapfiber using the configured Zap logger
