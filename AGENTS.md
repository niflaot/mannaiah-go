# Repository Engineering Rules

## Project Structure
- Root module (`go.mod`) is a container module only and must not contain executable entrypoints.
- All submodules must live under `module/` (for example, `module/core`, `module/<name>`).

## Testing
- All production code must be unit tested to the maximum practical level.
- Follow TDD by writing or updating tests before implementation changes whenever possible.
- Every new function must be covered by end-to-end test flows whenever it participates in runtime module behavior.
- Add resilience tests for critical flows, including authentication-provider outages, dependency connection failures (database/cache/messaging), and expected error-mapping behavior.
- Add concurrency/race-condition tests on uniqueness/idempotency critical paths whenever applicable.
- Add performance tests (benchmarks/load-oriented tests) for hot paths whenever practical, and keep them modular and reproducible.
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

## Architecture
- `module/core` is the foundational core module (`core/` in domain terms).
- All architecture outside `core/` (that is, outside `module/core`) must follow:
  - DDD (Domain-Driven Design)
  - Hexagonal Architecture (ports and adapters)
  - TDD delivery workflow

## Code Quality
- Prefer asynchronous/concurrent Go capabilities where they provide clear value.
- Keep code modular, reusable, and self-explanatory.
- For third-party integrations, prefer maintained libraries/SDKs behind adapters instead of custom protocol clients unless there is a justified gap.
- For external or unstable dependencies (remote APIs, cache, database connections, auth/JWKS), use configurable circuit breakers where practical, and test open-state graceful degradation behavior.
- Reduce file complexity through composition and package splitting:
  - avoid concentrating multiple responsibilities in a single file
  - split long services/stores into focused collaborators (for example, constructor/config, operations, mapping/validation, resilience helpers)
  - keep production files and test files intentionally small and navigable; when a file grows too large (roughly 250-300+ lines), refactor into cohesive units instead of adding more branches
