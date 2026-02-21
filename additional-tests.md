# Additional Test Follow-Up

## Acceptance Stability
- Convert remaining fixed resource names to unique/random names.
- Avoid parallel execution for tests sharing one API account context.
- Keep repeat-run script reports for tracking flaky cases.

## Provider-Level
- Add explicit checks for read-only fields returned by API.
- Expand tests for partial update semantics and clear-on-empty behaviors.
