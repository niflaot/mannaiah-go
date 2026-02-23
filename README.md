# Mannaiah Go

[![Build Status](https://ci.momlesstomato.dev/api/badges/flockstore/mannaiah-go/status.svg)](https://ci.momlesstomato.dev/flockstore/mannaiah-go)
![Latest Version](https://img.shields.io/badge/latest-v1.1.0-0A66C2)

Mannaiah Go is a modular monolith built with Go, DDD, and hexagonal architecture. The repository is organized as a container workspace with independent modules under `module/`, composed by the `core` runtime.

## Architecture

- `module/core`: shared runtime foundation (config, HTTP server, logging, database, messaging, cron, swagger aggregation).
- `module/auth`: authentication and authorization integration.
- `module/contacts`: contact domain.
- `module/orders`: order domain.
- `module/products`: product domain.
- `module/assets`: asset/storage domain.
- `module/falabella`: Falabella integration module.
- `module/woocommerce`: WooCommerce integration module.
- `e2e/`: root end-to-end validation flows.

## Key Runtime Endpoints

- `GET /status`: health/status endpoint.
- `GET /openapi.json`: aggregated OpenAPI document from core + modules.
- `GET /docs`: API documentation UI.

## Local Development

### Prerequisites

- Go `1.25.5`
- Docker (optional, for containerized runs)

### Start Locally

```bash
cp .env.example .env
go run ./module/core/cmd/api
```

The API listens on `CORE_HOST:CORE_PORT` (`0.0.0.0:8080` by default).

## Testing

### Module Unit Tests

```bash
for module in module/core module/auth module/contacts module/orders module/assets module/products module/falabella module/woocommerce; do
  (cd "$module" && go test ./...)
done
```

### Root E2E Tests

```bash
go test ./e2e -v -count=1
```

### WooCommerce Benchmark

```bash
cd module/woocommerce
go test ./application/contact/service -run '^$' -bench BenchmarkProcessCommands -benchmem -benchtime=100x -count=1
```

## Docker

### Build

```bash
docker build -t mannaiah-go:local .
```

### Run

```bash
docker run --rm -p 8080:8080 --env-file .env mannaiah-go:local
```

## CI/CD

- CI/CD is managed by Drone via `.drone.yml`.
- Validation pipeline runs module tests, e2e tests, and WooCommerce benchmark checks.
- Docker images are published to Nexus registry:
  - Registry: `docker.momlesstomato.dev`
  - Repository: `fl-docker/mannaiah-go`
- Drone secrets required for publish:
  - `nexus_username`
  - `nexus_password`
