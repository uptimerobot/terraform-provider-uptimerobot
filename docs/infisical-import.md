# Temporary GitHub → Infisical copy

This branch copies terraform-provider-uptimerobot's existing GitHub Actions secrets into UptimeRobot's
**Applications** project (`1811949a-baed-427c-9eb1-9525f6b43964`) on
`https://app.infisical.com`. Keep the PR as a draft and close it without merging
after the migration. Tracked in [UPT-4543](https://linear.app/uptimerobot/issue/UPT-4543/add-all-secrets-to-the-infisical).

| GitHub source | Infisical environment | Folder | Entries |
| --- | --- | --- | ---: |
| Repository Actions secrets | `repo` | `/terraform-provider-uptimerobot` | 3 |

The explicit inventory was checked on 2026-10-16. Organization-only
secrets, ordinary GitHub variables, and `INFISICAL_MIGRATION_TOKEN` are excluded.
Application and deployment workflows are unchanged. Environment import jobs use
`deployment: false`; repository import jobs do not select a GitHub environment.

## Prerequisites

Keep the corresponding Infisical GitHub syncs disabled throughout migration and
verification. An enabled sync could propagate imported values back to GitHub.
`SYNCS_CONFIRMED_DISABLED` records the operator's check; it does not inspect or
change sync settings.

Set `INFISICAL_MIGRATION_TOKEN` as a repository Actions secret. Prefer a short-lived
machine identity token limited to reading values and creating secrets in the
listed locations. An explicitly authorized CLI user-session token also works,
retains that user's permissions, and attributes operations to that user. Tokens
are not renewed by the importer. Never put credentials in source, command
arguments, workflow configuration, or logs.

Refresh the inventory before each run using these metadata endpoints:

```sh
gh api repos/uptimerobot/terraform-provider-uptimerobot/actions/secrets --paginate --jq '.secrets[].name'
```

Exclude the temporary migration token. Keep `scripts/infisical-secret-inventory.json`
and the explicit `SOURCE_*` workflow bindings aligned. GitHub resolves environment
→ repository → organization precedence; a removed source secret could otherwise
expose a fallback value.

## Run and verify

1. Push `chore/bac-370-infisical-import` with `MIGRATION_MODE: dry-run` and
   `SYNCS_CONFIRMED_DISABLED: 'false'`. Tests run first, then the listed scopes check
   source availability and existing destination values without writing.
2. After all dry-run jobs pass and syncs are confirmed disabled, change the
   top-level settings to `MIGRATION_MODE: apply` and
   `SYNCS_CONFIRMED_DISABLED: 'true'`. Commit and push. No merge into
   `main` is needed. The importer creates missing entries only,
   reads each back, then compares the full scope with GitHub.
3. Require all scope jobs to pass and compare destination names with GitHub.
   For another read-only check, set `MIGRATION_MODE: verify`, commit, and push.
4. Once all repositories using the migration credential are finished, revoke the
   session/token, remove its temporary GitHub copies, and close draft PRs without
   merging. Keep the original source secrets. Sync activation and rotation are
   separate tasks.

Every push while `apply` is selected can attempt a copy. Existing matching values
are preserved; conflicts stop that scope before writes. Failure can leave a
partial import, and reruns create only the missing entries. Finish or cancel
pending runs before switching modes; concurrency does not guarantee ordering.
Run repositories in small batches to reduce shared Infisical rate limits.

Explicit HTTP 429 responses get at most three retries. Numeric `Retry-After`
delays up to 120 seconds are honored; missing headers use 60, 120, and 120 seconds.
Longer or unrecognized delays stop for a later manual retry. Ambiguous network
failures are not automatically retried, including after creates.

Values travel through step-scoped environment variables and HTTPS JSON bodies,
never command arguments, files, or artifacts. Logs contain names and status only.
Read-back disables reference expansion and imported secrets; literal `${...}`
values stop for manual review. Newlines and special characters are preserved,
including multiline signing keys. Verification proves equality with GitHub's
resolved values at execution time, not service health or running-server settings.

## Local checks

```sh
python3 -B -m unittest discover -s scripts -p test_import_infisical_secrets.py
```

Tests use dummy values and mocked requests without accessing live services.
