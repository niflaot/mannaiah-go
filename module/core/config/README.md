# Config Package

`config` loads configuration structs from `.env` first and then overlays environment variables.

## Features
- Uses `viper` with `.env` parsing and automatic environment variable overrides.
- Applies `default` tag values for optional fields.
- Treats fields without `default` as required startup configuration.
- Validates and logs all missing required values through a provided Zap logger.
- Supports loading and validating multiple module configuration structs in one startup call.
- Exposes provider-agnostic configuration loading via `config.Provider`.

## Usage Rules
- Provide pointers to structs when calling loader methods.
- Depend on `config.Provider` in higher-level services when decoupling from implementation details.
- Use `mapstructure` tags for stable configuration keys.
- Add `default` tags for optional fields.
- Omit `default` tags when a field must be present at startup.

## Key Methods / Endpoints / Events
- Methods:
  - `config.NewLoader(envFile, startupLogger)`
  - `config.Load(envFile, startupLogger, targets...)`
  - `config.LoadWith(provider, targets...)`
  - `(*config.Loader).Load(targets...)`
- Endpoints: none in this package.
- Events: missing required configuration values are emitted as Zap error logs during startup loading.
