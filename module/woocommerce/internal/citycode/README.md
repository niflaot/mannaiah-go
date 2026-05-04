# citycode

`internal/citycode` resolves WooCommerce billing city name strings to Colombian municipality numeric codes.

## Behaviour

- City data is embedded at compile time from `cities.json` (1119 Colombian municipalities).
- `Resolve(name string) string` normalises the input (lowercase + accent stripping) and performs an O(1) map lookup.
- When no exact match is found, a Levenshtein similarity fallback scans all keys and accepts the best candidate above the 80 % similarity threshold.
- Returns `"-1"` when no match meets the threshold.
- `IsNumericCode(value string) bool` reports whether a stored city code is already resolved (numeric), used to guard update paths from overwriting valid codes.

## Performance

- Init: one-time JSON unmarshal of 1119 entries at package init.
- Hot path (exact match): single map lookup + normalization — O(1).
- Fuzzy path (rare): linear scan over 1119 keys with Levenshtein distance — negligible at this cardinality.
