# Core City Code Resolver

Shared Colombian municipality code resolver used by integrations that exchange human-readable city names with external platforms while Mannaiah stores city codes.

## Key Methods

- `Resolve` maps a city name to a Colombian municipality code and passes through already-resolved numeric codes.
- `Name` maps a city code back to a human-readable city name.
- `IsNumericCode` reports whether a value is already a positive numeric municipality code.
