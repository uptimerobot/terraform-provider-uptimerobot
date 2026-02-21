# API/UDP Follow-Up

## Scope
- Add full `type = "API"` monitor support in provider schema/transform/validation.
- Add full `type = "UDP"` monitor support in provider schema/transform/validation.

## Implementation Tasks
- Add `config.api_assertions` mapping end-to-end.
- Add `config.udp` mapping end-to-end.
- Add compare/read logic for API/UDP config fields to avoid perpetual diffs.
- Add wait/read normalization if API responses are eventually consistent.

## Tests
- Unit tests for transform, compare, and validation behavior.
- Acceptance tests for create/update/import flows for API and UDP monitors.
