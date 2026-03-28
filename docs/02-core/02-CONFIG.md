# Config

The `config` package centralises all configuration loading for Mannaiah. It uses
[Viper](https://github.com/spf13/viper) under the hood and maps environment variables to typed Go
structs via `mapstructure` tags.

## How It Works

Each module declares its own config struct with `mapstructure` tags matching the environment variable
name and an optional `default` tag:

```go
type Config struct {
    Host string `mapstructure:"CORE_HOST" default:"0.0.0.0"`
    Port int    `mapstructure:"CORE_PORT" default:"8080"`
}
```

The loader reads `.env` files and live environment variables, fills all registered structs, and returns
a `ValidationError` listing any required fields (those without a `default`) that are absent.

## Key Types

| Type | Purpose |
|------|---------|
| `Provider` | Interface abstracting config loading |
| `Loader` | Concrete Viper-backed implementation |
| `ValidationError` | Aggregated list of missing required fields |
| `MissingFieldError` | Describes one missing field (name, struct path) |

## Usage

```go
err := config.Load(".env", logger,
    &coreConfig,
    &dbConfig,
    &redisConfig,
    // ...
)
```

All targets are filled atomically; a single missing required field fails the entire load so the
process exits at startup rather than failing at runtime.
