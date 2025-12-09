## Unreleased

### Fixed
- Validation gaps: type of `lastIncidentId`.

## 1.3.2 — 2025-12-09

### Added
- DNS configuration comparison and wait logic plus keyword/region checks to reduce drift on monitor updates.
- Maintenance window normalization and day sync to align plans with API behavior.
- PSP handling for API eventual consistency, including waits and monitor ID cleanup.

### Changed
- Increased monitor settle timeouts and added conditional settle logic during updates.
- Adjusted integration name uniqueness and region handling along with broader normalization updates.
- Documentation refreshed (contribution/issue-claiming guidance, monitor count note).

### Fixed
- Validation gaps: unique URL, GA code, `lastincidentid`, and inner config fields.
- PSP monitor ID state handling and computed schema usage; monitor ID eventual-consistency fixes.
- DNS configuration fixes and other minor cleanups.

### Tests/CI
- Hardened monitor acceptance tests (name normalization wrapper, config tweaks, flake mitigations) and validation test adjustments.
- Added gitignore entries for cache/temp Terraform files; toolchain now ignores `.terraformrc`.
- Dependency and CI bumps (`terraform-plugin-log`, `actions/checkout`).

## 1.3.1 — 2025-11-22

### Fixed
- Alert contacts usage on update.
- Grace period checks for non-heartbeat monitors.

### Tests/CI
- Added coverage for alert contacts with missing values.

## 1.3.0 — 2025-11-05

### Added
- Request debug output (enable with `UPTIMEROBOT_DEBUG=1`)
- User-Agent now includes Terraform/OpenTofu version
- DNS monitor configuration
- Client package for API interactions - error handling and retry logic
- Integrations implementation

### Changed
- Wait semantics on **delete** and **update** for PSP and monitors. Automation is now more predictable
- Recreate behavior when **monitor type** changes
- **BREAKING:** Alert contacts validation - `threshold` and `recurrence` are now **required** with no hidden defaults.
- Internal helpers and attribute handling refactor
- Refactored and split whole monitor resource

### Fixed
- Upgrader logic for `config` to avoid false diffs
- Client error handling paths
- Various minor inconsistencies and bugs

### Tests/CI
- Increased test coverage for monitor resource
- Improved test structure and execution

## 1.0.0
Initial release
