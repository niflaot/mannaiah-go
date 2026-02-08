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
  - interface declarations and interface methods
  - struct declarations
  - struct fields/properties
- Do not add unnecessary inline comments inside function bodies.

## Architecture
- `module/core` is the foundational core module (`core/` in domain terms).
- All architecture outside `core/` (that is, outside `module/core`) must follow:
  - DDD (Domain-Driven Design)
  - Hexagonal Architecture (ports and adapters)
  - TDD delivery workflow

## Code Quality
- Prefer asynchronous/concurrent Go capabilities where they provide clear value.
- Keep code modular, reusable, and self-explanatory.
