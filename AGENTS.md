# Repository Engineering Rules

## Project Structure
- Root module (`go.mod`) is a container module only and must not contain executable entrypoints.
- All submodules must live under `module/` (for example, `module/core`, `module/<name>`).

## Testing
- All production code must be unit tested to the maximum practical level.
- Follow TDD by writing or updating tests before implementation changes whenever possible.

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
