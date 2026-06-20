# Repository Engineering Rules

## Project Structure
- Root module (`go.mod`) is a container module only and must not contain executable entrypoints.
- All submodules must live under `module/` (for example, `module/core`, `module/<name>`).

## Tooling
- Use `ripgrep` (`rg`) for text and file searches whenever possible for faster performance

## CI/CD
- Drone CI is the only supported CI/CD orchestrator for this repository.
- CI/CD source of truth must be `.drone.yml`; do not add or maintain GitHub Actions workflows under `.github/`.
- Validation pipeline must run module tests, root e2e tests, and WooCommerce performance benchmarks.
- Container publish pipeline must push images to Nexus registry `docker.niflaot.dev` under repository `fl-docker/mannaiah-go`.
- Drone deployment credentials must be injected via secrets (`nexus_username`, `nexus_password`) and never hardcoded.
- Publish behavior must include:
  - `main` branch pushes tagged as `latest` plus commit SHA.
  - Git tag events tagged with the Git tag plus commit SHA.

## Testing
- All production code must be unit tested to the maximum practical level.
- Follow TDD by writing or updating tests before implementation changes whenever possible.
- Every new function must be covered by end-to-end test flows whenever it participates in runtime module behavior.
- When e2e or migration-oriented tests assert the latest migration version, do not leave hardcoded version numbers behind; update them to resolve the latest embedded migration version dynamically when newer migrations are present.
- Add resilience tests for critical flows, including authentication-provider outages, dependency connection failures (database/cache/messaging), and expected error-mapping behavior.
- Add concurrency/race-condition tests on uniqueness/idempotency critical paths whenever applicable.
- Add performance tests (benchmarks/load-oriented tests) for hot paths whenever practical, and keep them modular and reproducible.
- Critical workflows must include telemetry propagation tests (HTTP + messaging + dependency failure paths).
- External integration modules must include enabled/disabled path tests and outage tests (invalid credentials/host/timeouts) validating graceful behavior.
- Integration endpoints that are documented but unavailable due to invalid integration config must return controlled service-unavailable errors and must be covered by unit and e2e tests.

## Documentation
- Use Go doc style comments only on:
  - function signatures
  - private function signatures
  - unit test function signatures
  - interface declarations and interface methods
  - mocked interface method signatures used in unit tests (method signatures only, not properties)
  - struct declarations
  - struct fields/properties
- Do not add unnecessary inline comments inside function bodies.
- Every package must include a `README.md` describing context and usage within the module.
- Core modules must include a clear "Key methods / endpoints / events" section so users can quickly understand exposed behavior.
- Core must centralize Swagger/OpenAPI aggregation and exposure.
- Every endpoint must be documented in module-level OpenAPI specs.
- Modules must provide their own OpenAPI spec artifacts; core startup must merge all module specs into a single aggregated document.
- Composition roots must expose the aggregated spec endpoint (for example, `/openapi.json`).
- Shipping API contract updates must also update `module/shipping/README.md` endpoint matrix with request/response behavior notes for new routes.

## Architecture
- `module/core` is the foundational core module (`core/` in domain terms).
- All architecture outside `core/` (that is, outside `module/core`) must follow:
  - DDD (Domain-Driven Design)
  - Hexagonal Architecture (ports and adapters)
  - TDD delivery workflow
- Cross-module communication must use integration events through ports/adapters (orchestrated by core messaging) instead of direct module-to-module coupling.

## Code Quality
- Prefer asynchronous/concurrent Go capabilities where they provide clear value.
- Keep code modular, reusable, and self-explanatory.
- For third-party integrations, prefer maintained libraries/SDKs behind adapters instead of custom protocol clients unless there is a justified gap.
- For external or unstable dependencies (remote APIs, cache, database connections, auth/JWKS), use configurable circuit breakers where practical, and test open-state graceful degradation behavior.
- For SQL-backed persistence, prefer normalized relational schemas by default:
  - avoid storing queryable business structures as opaque JSON/text blobs when they can be modeled as relational child tables
  - enforce integrity with explicit keys/indexes/uniqueness constraints where applicable
  - when denormalization is intentionally chosen for performance, document the rationale and add consistency safeguards/tests
- For history/event-driven aggregates, apply this rule in all modules:
  - Source-of-truth: child history/event tables are authoritative (not root-table snapshot columns).
  - Read path: resolve "current/latest" state by querying latest history/event row with deterministic ordering (recommended: `occurred_at DESC, id DESC`).
  - Write path: do not persist duplicated `current_*` / `latest_*` fields in transactional root tables when the same information exists in history/event rows.
  - Filter path: status/state filters must be computed from history/event tables (subquery/join), not from root-table snapshot fields.
  - Exception: if denormalized snapshots are needed for performance, place them in dedicated derived/materialized read models (outside transactional source tables), document rationale, and add consistency verification tests.
- Reduce file complexity through composition and package splitting:
  - avoid concentrating multiple responsibilities in a single file
  - split long services/stores into focused collaborators (for example, constructor/config, operations, mapping/validation, resilience helpers)
  - keep production files and test files intentionally small and navigable; when a file grows too large (roughly 250-300+ lines), refactor into cohesive units instead of adding more branches
- Enforce feature package role-splitting for application modules:
  - keep `application/<feature>` as a namespace package when multiple responsibilities exist
  - place use-case orchestration in `application/<feature>/service`
  - place integration event contracts/builders in `application/<feature>/event`
  - do not define integration event topics/payload builders inside service packages when an `event` package exists

## Observability & Telemetry (Mandatory)
- Every runtime service must expose Prometheus metrics at `/metrics` from the core composition root.
- Distributed tracing must use OpenTelemetry only; exporter configuration must be environment-driven.
- Trace context must propagate across HTTP and integration events (`traceparent` metadata) end-to-end.
- Telemetry must be fail-open: exporter/backends failing must not break service startup or request handling.
- Metrics/spans must use low-cardinality labels/attributes only; never include PII, IDs, query strings, raw payload fragments, or secrets.
- New adapters that perform outbound I/O must include tracing spans and dependency metrics.
- Every telemetry change must include unit tests, integration propagation tests, and at least one performance benchmark for hot paths.
- `/metrics` exposure must be documented with network/access restrictions.

## Database Migrations (Mandatory)
- All SQL schema changes must be delivered through versioned migration files; do not rely on ad-hoc runtime `AutoMigrate` in production startup paths.
- When schema is touched, include both directions:
  - `*.up.sql` for forward changes
  - `*.down.sql` for rollback changes
- Migration files must live in the core migration source-of-truth directory and use monotonic numeric version prefixes.
- Pull requests that modify schema-affecting code must also include corresponding migration files and tests covering startup/apply behavior where practical.
- Prefer backward-compatible rollout strategy:
  - additive changes first (new table/column/index),
  - application rollout,
  - cleanup/drop in later migration.
- Destructive schema changes (drop/rename/type narrowing) require explicit rollback strategy and data-safety notes in the PR.
- New indexes/constraints must be intentional and validated against query paths and uniqueness/idempotency requirements.
- If denormalization is introduced for performance, document rationale and consistency safeguards/tests in the same PR.
