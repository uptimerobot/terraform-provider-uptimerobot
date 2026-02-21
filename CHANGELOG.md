## Unreleased

### Added
- Monitor `group_id` support (`groupId` in API) for create, update, read/import, and state comparison/wait logic.
- Added `type = "API"` monitor support in `uptimerobot_monitor`.
- Added `config.api_assertions` support with create/update/read transforms and API payload mapping to `config.apiAssertions`.

### Changed
- Monitor schema constraints were aligned to API v3 for currently supported monitor types:
  - `interval` minimum is now `30`.
  - `name` max length is now `250`.
  - `http_username` and `http_password` max length is now `255`.
  - `response_time_threshold` range is now `0..60000`.
  - `port` range now allows `0..65535` (type-specific validation still applies).
  - `auth_type` validation now allows `NONE`, `HTTP_BASIC`, `DIGEST`, `BEARER`.
- Monitor docs were updated accordingly, including `group_id` and config-branch notes for API v3.
- Extended monitor URL validation and HTTP-like method handling to include API monitors.
- Updated monitor docs/examples to cover API monitor assertions configuration.
- Acceptance test execution strategy was adjusted:
  - Pull Requests now run a reduced acceptance matrix (`terraform latest`, `opentofu latest`) for faster feedback.
  - `main`/manual runs execute an extended acceptance matrix (4 lanes) for broader compatibility coverage.

### Fixed
- DNS config validation warning now correctly treats both `config.dns_records` and `config.ssl_expiration_period_days` as managed DNS config fields.
- PSP create/update no longer fails with API 400 when `icon`/`logo` are configured as URL strings: provider now rejects non-empty `icon`/`logo` config values with a clear validation message, matching API v3 multipart-only behavior for these fields.
- Stabilized monitor/maintenance-window/PSP eventual-consistency waits by requiring consecutive matching reads before treating updates as settled.
- Fixed monitor alert-contact settle comparison for explicit clears (`assigned_alert_contacts = []`).
- Fixed monitor refresh drift after maintenance-window updates by stabilizing read snapshots against managed `maintenance_window_ids`.
- Fixed PagerDuty integration refresh drift for `location` and `auto_resolve` by making read mapping resilient to stale replica responses.

### Tests/CI
- Acceptance tests now use unique randomized names in key monitor/PSP/maintenance-window cases to reduce cross-test collisions.
- Removed acceptance `t.Parallel()` usage for account-shared monitor config scenarios to improve deterministic results.
- Added unit tests for PagerDuty integration read parsing/sticky behavior (`location`/`auto_resolve`).
- Added a dedicated CI `unit` job that runs only non-acceptance tests; acceptance jobs now run only `TestAcc*`.
- Consolidated monitor unit tests into thematic files with explicit section separators to reduce test-file fragmentation.
- Added acceptance coverage for API monitor assertions round-trip and API-specific validation cases.
- Added unit coverage for API assertions config transform/compare/marshal behavior.

## 1.3.9 — 2025-12-24

### Fixed
- Prevented `config.dns_records` from showing as `(known after apply)` when `http_method_type` is GET/HEAD on HTTP/KEYWORD monitors.

## 1.3.8 — 2025-12-24

### Fixed
- Surface integration creation conflicts (already exists) as diagnostics with API message details instead of raw 409 errors.
- Surface PSP creation access-denied errors (403) with API message details instead of raw HTTP errors.

## 1.3.7 — 2025-12-24

### Fixed
- Normalized monitor `name` and `url` when the API returns HTML-escaped values, avoiding drift and import/update issues.
- Import now correctly handles percent-encoded monitor `name`/`url` values.

### Changed
- Documentation clarifies that `name` and `url` should be written as plain text (no HTML entities).

### Tests
- Added acceptance coverage for importing monitors with encoded `name`/`url`.
- Adjusted monitor acceptance expectations and added unit tests for monitor `name`/`url` normalization.

## 1.3.6 — 2025-12-20

### Fixed
- Stabilized `uptimerobot_psp` attribute handling (including `custom_settings`) to prevent inconsistent API responses causing perpetual diffs.
- Provider now validates `monitor_ids` consistency after apply to prevent drift.
- Various inconsistencies related to `uptimerobot_psp` resource.

### Tests
- Added acceptance coverage for PSP attribute consistency and `custom_settings` stability.

## 1.3.5 — 2025-12-17

### Changed
- **BREAKING:** `keyword_value`, `keyword_type`, and `keyword_case_type` are now only allowed for `type = KEYWORD` monitors. Non-KEYWORD monitors no longer send or retain these fields to prevent drift.
- **BREAKING:** `type = KEYWORD` monitors now require explicitly setting `keyword_case_type` (no implicit defaulting).

### Fixed
- Added `keyword_value` max length validation (≤ 500 characters) to match API constraints.
- Create/update plan validation now requires `keyword_type`, `keyword_case_type`, and `keyword_value` to be set and known for `type = KEYWORD`.
- Port monitor validation behavior was adjusted.
- Validate `url` format for `type = HTTP/KEYWORD` to require an `http://` or `https://` URL, avoiding API-side "Wrong URL or IP" errors for bare hosts.

### Tests
- Consolidated monitor port/validation tests into `monitor_validate_test.go` and removed the dedicated CRUD port-guard test file.
- Added coverage for PORT high-level plan validation and KEYWORD validation edge cases.

## 1.3.4 — 2025-12-16

### Fixed
- State upgrader for `config` block now includes `dns_records` attribute when upgrading from V3/V4 state, fixing "Value Conversion Error" when using `config` with only `ssl_expiration_period_days`
- Plan modifier normalizes partial `config` objects to include all expected attributes

### Tests
- Added unit tests for V3/V4 config state upgrades

## 1.3.3 — 2025-12-11

### Fixed
- Changed field types of `Incident.ID` and `Monitor.LastIncidentID` to string

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
