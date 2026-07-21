# Terraform Provider for UptimeRobot

This Terraform provider allows you to manage your UptimeRobot resources programmatically.

## Maintainer Notice

This is the official UptimeRobot Terraform provider.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.5 or [OpenTofu](https://opentofu.org/docs/intro/install/) >= 1.7
- An [UptimeRobot](https://uptimerobot.com) account.
- An UptimeRobot API Key. You can generate your Main API Key from your UptimeRobot dashboard under "My Settings" -> "API Settings" -> "Main API Key".

## Installation

To use this provider, add the following to your Terraform configuration, then run `terraform init`.

```hcl
terraform {
  required_providers {
    uptimerobot = {
      source  = "uptimerobot/uptimerobot"
      version = "~> 1.9.1"
    }
  }
}

provider "uptimerobot" {
  # Configuration options
}
```

## Provider Configuration

The provider requires an API key to interact with the UptimeRobot API.

```hcl
provider "uptimerobot" {
  api_key = "YOUR_UPTIMEROBOT_API_KEY"
  # api_url = "https://api.uptimerobot.com/v3" # Optional: Default is UptimeRobot API v3 URL
}
```

### Argument Reference

- `api_key` (String, Required): Your UptimeRobot Main API Key.
- `api_url` (String, Optional): The base URL for the UptimeRobot API. Defaults to `https://api.uptimerobot.com/v3`. Useful if UptimeRobot offers different API endpoints or for testing purposes.

## Usage Examples

Here's an example of how to create an UptimeRobot monitor, a maintenance window, an integration, and a public status page:

```terraform
terraform {
  required_providers {
    uptimerobot = {
      source  = "uptimerobot/uptimerobot"
      version = "~> 1.9.1"
    }
  }
}

provider "uptimerobot" {
  api_key = "YOUR_UPTIMEROBOT_API_KEY"
}

resource "uptimerobot_monitor" "website" {
  name     = "My Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
}

resource "uptimerobot_maintenance_window" "weekly" {
  name     = "Weekly Maintenance"
  type     = "weekly"
  duration = 60
  interval = "weekly"
  time     = "23:00"
}

resource "uptimerobot_integration" "slack" {
  name  = "Team Slack"
  type  = "slack"
  value = "https://hooks.slack.com/services/XXXXX/YYYYY/ZZZZZ"
}

resource "uptimerobot_psp" "main_status" {
  name        = "Example.com Status"
  monitor_ids = [uptimerobot_monitor.website.id]
}
```

## Resource Reference

Detailed documentation for the resources supported by this provider can be found in the `docs/resources/` directory or by clicking the links below:

- [uptimerobot_monitor](docs/resources/monitor.md)
- [uptimerobot_maintenance_window](docs/resources/maintenance_window.md)
- [uptimerobot_integration](docs/resources/integration.md)
- [uptimerobot_psp](docs/resources/psp.md)
- [uptimerobot_psp_announcement](docs/resources/psp_announcement.md)

## Data Source Reference

Detailed documentation for the data sources supported by this provider can be found in the `docs/data-sources/` directory or by clicking the links below:

- [uptimerobot_alert_contact](docs/data-sources/alert_contact.md)
- [uptimerobot_alert_contacts](docs/data-sources/alert_contacts.md)
- [uptimerobot_all_alert_contacts](docs/data-sources/all_alert_contacts.md)
- [uptimerobot_current_user](docs/data-sources/current_user.md)
- [uptimerobot_ip_ranges](docs/data-sources/ip_ranges.md)
- [uptimerobot_integration](docs/data-sources/integration.md)
- [uptimerobot_maintenance_window](docs/data-sources/maintenance_window.md)
- [uptimerobot_monitor](docs/data-sources/monitor.md)
- [uptimerobot_monitor_group](docs/data-sources/monitor_group.md)
- [uptimerobot_psp](docs/data-sources/psp.md)
- [uptimerobot_psp_announcement](docs/data-sources/psp_announcement.md)
- [uptimerobot_tag](docs/data-sources/tag.md)
- [uptimerobot_tags](docs/data-sources/tags.md)

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](https://golang.org/doc/install) installed on your machine. Use the version declared in `go.mod` (currently 1.26.5), which is also what CI uses.

### Building The Provider

1.  Clone the repository.
2.  Enter the repository directory.
3.  Build the provider using the Go `install` command:

    ```shell
    go install
    ```
    This will build the provider and put the provider binary in the `$GOPATH/bin` directory (or `$GOBIN` if set).

### Local Development and Testing

To use your locally built provider for testing with a Terraform configuration, you can specify the development override in your Terraform CLI configuration file. See [Provider Development Overrides](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers).

Example `~/.terraformrc` or `terraform.rc`:

```hcl
provider_installation {
  dev_overrides {
    "uptimerobot/uptimerobot" = "/path/to/your/gopath/bin" # or wherever 'go install' places the binary
    # For example: "uptimerobot/uptimerobot" = "$HOME/go/bin"
  }
  # For TF 0.13+ installations, you can also use a direct path to the binary
  # fs_mirror {
  #   "https://registry.terraform.io/providers/uptimerobot/uptimerobot" = "/path/to/your/project/terraform-provider-uptimerobot"
  # }
}
```

### Generating Documentation

Generate or update registry documentation by running:

```shell
go generate ./...
```

This formats Terraform examples and regenerates the files in `docs/` from the provider schema and templates.

### Running Acceptance Tests

Acceptance tests create real resources against the UptimeRobot API and may incur costs or affect your UptimeRobot account.
Ensure you have the `UPTIMEROBOT_API_KEY` environment variable set before running tests.

To run the full suite of Acceptance tests:

```shell
make testacc
```

*Note: Acceptance tests create real resources*

### Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
To add a new dependency `github.com/author/dependency`:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on proposing changes, running tests, and opening pull requests.

## License

This provider is distributed under the Mozilla Public License Version 2.0. See the `LICENSE` file for more information.
