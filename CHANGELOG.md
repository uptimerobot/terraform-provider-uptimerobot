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

