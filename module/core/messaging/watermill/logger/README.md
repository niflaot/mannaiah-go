# Watermill Logger Package

`watermill/logger` provides Zap-to-Watermill logging adaptation.

## Responsibilities
- Adapt `*zap.Logger` into `watermill.LoggerAdapter`.
- Normalize log field values into stable serializable forms.
- Provide nil-safe logger fallback behavior.

## Key Methods / Endpoints / Events
- Methods:
  - `logger.NewZapAdapter(providedLogger)`
- Endpoints: none in this package.
- Events: none emitted directly by this package.
