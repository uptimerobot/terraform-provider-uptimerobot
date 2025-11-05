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
   make build
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
First add Uptimerobot API key to env vars:
`export UPTIMEROBOT_API_KEY="ur_XXXXXXXXXXXXXXXXXXXX"`

#### optional for alert-contact tests:
`export UPTIMEROBOT_TEST_ALERT_CONTACT_ID="1234567"`

### All acceptance tests (acc tests)
Use `make testacc` to run whole test suit

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

Before openning PR make sure that tests are passing locally.

## Notes:
- Some acceptance tests can be unstable due to network or API. Re-running is sometimes necessary.
- Never commit real API keys or logs containing secrets. The provider redacts known secrets in debug output.
