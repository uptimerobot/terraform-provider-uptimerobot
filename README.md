# Terraform Provider for UptimeRobot

This Terraform provider allows you to manage your UptimeRobot resources programmatically.

## Maintainer Notice

This is the official UptimeRobot Terraform provider.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/install) or [OpenTofu](https://opentofu.org/docs/intro/install/)
- An [UptimeRobot](https://uptimerobot.com) account.
- An UptimeRobot API Key. You can generate your Main API Key from your UptimeRobot dashboard under "My Settings" -> "API Settings" -> "Main API Key".

## Installation

To use this provider, add the following to your Terraform configuration, then run `terraform init`.

```hcl
terraform {
  required_providers {
    uptimerobot = {
      source  = "uptimerobot/uptimerobot"
      version = "~> 1.0.0"
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

Here's a basic example of how to create an UptimeRobot monitor:

```hcl
resource "uptimerobot_monitor" "mysite_monitor" {
  name = "My Production Website"
  url           = "https://myproductionsite.com"
  type          = "http"
  interval      = 300 # Interval in seconds (e.g., 300 for 5 minutes)
  # ... other monitor-specific arguments
}
```

## Resource Reference

Detailed documentation for the resources supported by this provider can be found in the `docs/resources/` directory or by clicking the links below:

- `uptimerobot_monitor`: (Link to `docs/resources/monitor.md` or similar)
- (Add links for other resources as they are created)

## Data Source Reference

Detailed documentation for the data sources supported by this provider can be found in the `docs/data-sources/` directory or by clicking the links below:

- `uptimerobot_account`: (Link to `docs/data-sources/account.md` or similar)
- (Add links for other data sources as they are created)

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](https://golang.org/doc/install) installed on your machine (version 1.24 or higher is recommended).

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

If you have `tfplugindocs` installed, you can generate or update documentation by running:

```shell
go generate
```

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

Contributions are welcome! Please open an issue or submit a pull request if you have improvements or bug fixes.

## License

This provider is distributed under the Mozilla Public License Version 2.0. See the `LICENSE` file for more information.
