# Contributing to terraform-provider-uptimerobot

Thanks for helping improve the provider! This doc shows how to build, test, and propose changes.

## Prerequisites
- Go **1.24.x**
- Terraform **≥ 1.5** or OpenTofu **≥ 1.7**
- UptimeRobot **v3** API key (for acceptance tests)

---

## Local dev build & Terraform/OpenTofu override

1. **Build the provider**
   ```bash
   make build-dev
   # or: go build -o ./bin/terraform-provider-uptimerobot_v0.0.0-dev .
   ```

2. **Create override config**
Create ```~/.terraformrc``` (UNIX/MacOS) or ```%APPDATA%\terraform.rc```(Windows).
You can also set `TF_CLI_CONFIG_FILE` to a custom path (works for Terraform and OpenTofu):
`export TF_CLI_CONFIG_FILE="$PWD/.terraformrc"`
Example:

    ```
    # ~/.terraformrc (example)
    provider_installation {
    dev_overrides {
        "uptimerobot/uptimerobot" = "/ABSOLUTE/PATH/TO/local/build/dir"
    }
    direct {}
    }
    ```

3. **Use dev version in your config**
    ```
    terraform {
      required_providers {
        uptimerobot = {
          source  = "uptimerobot/uptimerobot"
          # version intentionally omitted for local dev with dev_overrides
        }
      }
    }
    ```
    Then run
    ```bash
    terraform init -upgrade
    terraform apply
    ```

## Running tests

### Unit tests
`go test ./... -race`

### Acceptance tests with real API interaction
First add UptimeRobot API key to env vars:
`export UPTIMEROBOT_API_KEY="ur_XXXXXXXXXXXXXXXXXXXX"`

#### optional for alert-contact tests:
`export UPTIMEROBOT_TEST_ALERT_CONTACT_ID="1234567"`

### All acceptance tests (acc tests)
Use `make testacc` to run whole test suite

They can be slow and sometimes flaky due to network and API interactions and connections.
Or use
`go test ./internal/provider -run 'Acc' -v -timeout 45m -parallel=1`

### Single test execution
`go test ./internal/provider -run '^TestAcc_Monitor_Name_HTMLNormalization$' -v -timeout 45m -parallel=1`

## Debugging
- Terraform/OpenTofu logs:
`TF_LOG=TRACE terraform plan`
- Provider HTTP/debug:
`UPTIMEROBOT_DEBUG=1 terraform apply`
Secrets in request/response bodies are auto-redacted in provider logs.

## Linting
For linter checks use `make lint` it will run golangci-lint. It will also run on any PRs automatically in CI/CD.

## Commits:
Use Conventional Commits for changes: feat:, fix:, chore:, test:, docs:, etc.
Flag breaking changes with ! (e.g., feat!: ...) and call them out in the PR + changelog.

## Provider design guardrails (important)
- **Explicit > implicit**
Null or omitted - leave existing remote value as-is.
Empty ("", [], {}) - clear on server, if API supports clearing.
- **Predictable planning**
Lean into Terraform’s null vs empty semantics.
Don’t clear remote state just because an attribute is omitted.
- **Easy drift control**
If users changed something in the UI, avoid fighting it. Support lifecycle.ignore_changes patterns.
- **Normalize API**
Nil or "" to types.StringNull() if applicable to prevent big and unneeded diffs.

## Docs & examples
Update resource docs when adding or changing schema.
Keep examples runnable, prefer small configs that prove behavior.

## Opening a PR
Include in a PR a description of changes, scope, user impact, edge cases.
Along the changes add tests (unit and/or acceptance), and a CHANGELOG.md entry under Unreleased.

Note any BREAKING changes loudly and why they are needed.
Maintainers handle version increment and releases.

Before opening a PR, make sure tests pass locally.

## Notes:
- Some acceptance tests can be unstable due to network or API. Re-running is sometimes necessary.
- Never commit real API keys or logs containing secrets. The provider redacts known secrets in debug output.


# Working on issues: claiming & coordination

## How to claim

**1.** Pick an open issue.
**2.** Comment:
`Claiming this. Plan: … (1–3 bullets). Draft PR in ~48h.`
**3.** A maintainer will assign you and add status to an issue
If two people ask, maintainers choose based on the first clear plan or prior context.
**4.** Open a draft PR within agreed time (~48 hours) referencing the issue (`Resolves #123`). This “locks” the issue to you.
Additionally, other words(case insensitive) may be used for referencing issue depending on the change:
- `close`
- `closes`
- `closed`
- `fix`
- `fixes`
- `fixed`
- `resolve`
- `resolves`

## Inactivity / handoff
- No draft PR in 48h - the issue may be unassigned.
- No progress updates in 7 days - the issue may be unassigned for someone else to take.
- If you stop, please comment and unassign yourself.

## Large or breaking changes
For non-trivial schema/API behavior or breaking changes, open a short RFC issue first with problem, approach, and impact. Wait for maintainer acknowledgment and approval before coding.

## Etiquette
Keep discussion on the issue and not in the PR until implementation starts.
If you want to pair, say so in the thread.
Don’t force-push over shared branches.

## Some comments examples
Claim: “Claiming this issue. Plan: parse X, validate Y, tests Z. Draft PR in 48h.”
Blocker: “Blocked by API response. Need guidance on …”
Handoff: “Unassigning, can’t continue. Notes: …”


## Issues labels
### Statuses:
`status/triage` — inbox / needs review. New issue
`status/needs-info` — Waiting for reporter details (repro steps, versions, logs, minimal config)
`status/duplicate` - Duplicate of another issue
`status/invalid` - Not correct issue, for example not related to the provider
`status/assigned` — Issue is assigned to someone to implement
`status/in-progress` — Work is in progress
`blocked/external` — Blocked by external team/vendor
`blocked/internal` — Blocked by related work in org/repos
`blocked/upstream` — Blocked by external upstream (API/tooling)
`status/review` — Ready for code reviews
`status/wontfix` — Won't be implemented or fixed
`status/reopened` - Reopened with new info
`status/ready-to-merge` - Changes are ready to be merged


### Types:
`bug` — Bug in code, logic or structure
`enhancement` — New feature or request
`documentation` — Documentation updates
`refactor` — Activities related to refactoring of the current structure or logic
`test` — Additional tests, unit or acceptance
`breaking-change` — Breaking change in any aspect of logic, structure, etc

### When we may reject an issue

We close issues with a clear reason and label:

- **Duplicate** — Tracked elsewhere.
  Label: `status/duplicate`. We link the canonical / original issue and close.

- **Invalid / no repro** — Not a provider bug, misconfiguration, or we can’t reproduce.
  Label: `status/invalid`. If you later provide a minimal repro, we’ll reopen.

- **Wontfix (by design / out of scope)** — Conflicts with our design guardrails or provider scope.
  Label: `status/wontfix`.



### Design principles & common “won’t fix” cases

We follow these principles to keep plans predictable and safe. Requests that conflict with them are typically closed as `status/wontfix`:

- **Explicit > implicit**: no implicit destructive changes or silent clears.
- **Predictable planning**: respect Terraform’s `null` (leave as-is) vs empty (`[]`, `""`, `{}` - clear) semantics.
- **Easy drift control**: don’t fight UI changes. Support `lifecycle.ignore_changes`. Avoid hidden mutations.
- **No hidden defaults** that mask drift or mutate server state without an explicit config change.
- **Deterministic behavior**: avoid long-running/background jobs or “eventually” semantics that break plan/apply expectations.
- **Import & plan stability**: avoid changes that degrade importability or create persistent spurious diffs.


### Other common reasons for `status/wontfix`

- **Out of provider scope** — Better solved via modules, external tooling, or `lifecycle.ignore_changes`.
  _Example:_ templating/transforming payloads at apply time → use a module or `templatefile()`.

- **Breaks import/plan predictability** — Behaviors that make plans non-deterministic or degrade imports.
  _Example:_ background polling that mutates remote config “eventually,” causing plan flapping after apply.

- **High maintenance / low impact** — Disproportionate test/maintenance surface for marginal user benefit.
  _Example:_ niche API flag that varies per account, requires complex mocks, and has no clear demand.

- **Upstream limitations** — Needs an UptimeRobot API change.
  Use `blocked/upstream`. We’ll revisit if upstream moves.



## Reopening closed issues
  Maintainers and collaborators with triage or write access may reopen issues.

  Bring new information, like reproducible configuration, additional information, or when upstream change, or clarified scope and we’ll happily reopen or create a follow-up or new issue.

  Apply `status/reopened` when reopening, then it will be moved back to status/triage.

  If the scope is different (e.g., different resource/path), the versions are majorly different, or the symptoms/error have materially changed. Link the prior issue for context.

When to reopen vs. open new:
- **Reopen** if it’s the same problem and now includes a minimal repro, supported versions, or an upstream change that removes the original blocker.
- **New issue** if the scope is different (e.g., different resource/path), the versions are majorly different, or the symptoms/error have materially changed. Link the prior issue for context.


## Questions belong in Discussions and not in the Issues

“How do I configure X?”, “Why does my plan show a change?”, and general how-to/support questions go to **GitHub Discussions**.
Issues are for **bugs** with a reproducible configuration and **feature requests** with a clear problem statement and proposed UX.

If a question is opened as an issue, we’ll move it to Discussions or close with the `question` label and a link to Discussions.
