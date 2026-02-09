# Messaging Bus Package

`bus` provides technology-neutral messaging ports and envelope types for module integration.

## Features
- Transport-agnostic message envelope (`bus.Message`).
- Publisher output port contract (`bus.Publisher`).
- Optional subscriber registration port contract (`bus.Registrar`).
- Standard metadata keys for correlation and schema governance.

## Usage Rules
- Application/domain layers should depend on `bus.Publisher` and optionally `bus.Registrar`.
- Do not import Watermill or transport-specific packages from outside infrastructure adapters.

## Key Methods / Endpoints / Events
- Methods:
  - `bus.Publisher.Publish(ctx, msg)`
  - `bus.Registrar.AddHandler(topic, handler)`
- Endpoints: none in this package.
- Events: defines integration event envelope and metadata contracts only.
