# Integrations Follow-Up

## Scope
- Improve acceptance stability for integration update/read scenarios.

## Implementation Tasks
- Normalize region/location fields exactly as API returns.
- Re-check bool field consistency (`auto_resolve` and similar fields).
- Add targeted wait/read consistency after update-heavy integration scenarios.

## Tests
- Add/adjust acceptance tests with unique names and update transitions.
- Keep drift checks deterministic under eventual consistency.
