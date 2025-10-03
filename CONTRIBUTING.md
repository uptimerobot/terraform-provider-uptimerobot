## Local dev build & Terraform/OpenTofu override

1. **Build the provider**
   ```bash
   make build
   # or: go build -o ./bin/terraform-provider-uptimerobot_v0.0.0-dev . 
   ```

2. **Create override config**
Create ```~/.terraformrc``` (UNIX/MacOS) or ```%APPDATA%\terraform.rc```(Windows).
You can also set TF_CLI_CONFIG_FILE to a custom path (works for Terraform and OpenTofu):
export TF_CLI_CONFIG_FILE="$PWD/.terraformrc"

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

## NOTES  
For any execution diagnostics add ```TF_LOG=TRACE``` before the command.
For example ```TF_LOG=TRACE terraform plan```

Alert contacts acceptance tests may be executed locally or via CI/CD by using
```UPTIMEROBOT_TEST_ALERT_CONTACT_ID``` env var. Refer to `internal/provider/monitor_resource_acc_test.go` for more info.
