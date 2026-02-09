# HTTP Package

`http` provides a Fiber-based HTTP server wrapper with config-driven address resolution and Zap request logging.

## Features
- Fiber server initialization with HTTP config defaults.
- Host/port resolution from HTTP config or fallback from core config.
- Zap logger integration through `zapfiber` middleware.
- Route registration and route-group mounting APIs for future modules.
- Abstract router/context interfaces (`http.Router`, `http.Context`) to decouple module route code from Fiber internals.

## Usage Rules
- Load `http.Config` using the shared configuration loader.
- Prefer `NewWithCore` when host/port should fallback to core config.
- Prefer `RegisterRoutes` and `MountRoutes` for provider-agnostic route registration.
- Use `Register` and `Mount` only when direct Fiber APIs are required.

## Key Methods / Endpoints / Events
- Methods:
  - `http.New(cfg, providedLogger)`
  - `http.NewWithCore(cfg, coreCfg, providedLogger)`
  - `http.AddressFrom(cfg, coreCfg)`
  - `(*http.Server).RegisterRoutes(register)`
  - `(*http.Server).MountRoutes(prefix, register)`
  - `(*http.Server).Register(register)`
  - `(*http.Server).Mount(prefix, register)`
  - `(*http.Server).Start()`
  - `(*http.Server).StartWithListener(listener)`
  - `(*http.Server).Shutdown(ctx)`
- Endpoints: none in this package.
- Events: HTTP request logs are emitted through zapfiber using the configured Zap logger.
