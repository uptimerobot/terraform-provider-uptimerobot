# Maintenance Windows Follow-Up

## Scope
- Reduce intermittent refresh-plan diffs after maintenance window updates.

## Implementation Tasks
- Revisit read-after-update consistency handling for monitor associations.
- Confirm null/omitted behavior for optional fields (including `auto_add_monitors`).
- Ensure state masking aligns with API eventual-consistency behavior.

## Tests
- Add acceptance coverage for add/remove monitor IDs and null/set transitions.
- Add retry-aware checks for update-heavy steps if needed.
