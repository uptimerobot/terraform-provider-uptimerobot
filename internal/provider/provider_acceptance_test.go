//go:build acceptance

package provider

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"uptimerobot": providerserver.NewProtocol6WithError(New("test")()),
}

func TestMain(m *testing.M) {
	// TestMain runs before default flag parsing,
	// due to this - parse once so -test.timeout is available.
	if !flag.Parsed() {
		flag.Parse()
	}

	softTimeout, enabled, err := acceptanceSoftTimeout()
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid TF_ACC_SOFT_TIMEOUT: %v\n", err)
		os.Exit(2)
	}

	done := make(chan struct{})
	if enabled {
		go func() {
			timer := time.NewTimer(softTimeout)
			defer timer.Stop()
			select {
			case <-timer.C:
				fmt.Fprintf(
					os.Stderr,
					"ERROR: acceptance test timeout reached after %s\n",
					softTimeout,
				)
				fmt.Fprintln(
					os.Stderr,
					"Run stopped early to avoid a panic. For full goroutine traceback, rerun with TF_ACC_SOFT_TIMEOUT=0.",
				)
				fmt.Fprintln(os.Stderr, "Tip: rerun with TF_LOG=DEBUG and a narrower -run filter to isolate the hanging step.")
				os.Exit(1)
			case <-done:
			}
		}()
	}

	code := m.Run()
	close(done)
	os.Exit(code)
}

func acceptanceSoftTimeout() (time.Duration, bool, error) {
	// TF_ACC_SOFT_TIMEOUT=0 for disabling
	// TF_ACC_SOFT_TIMEOUT=30m for setting exact duration
	// default behavior - it will use setted value in test.timeout
	if raw := os.Getenv("TF_ACC_SOFT_TIMEOUT"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return 0, false, err
		}
		if d <= 0 {
			return 0, false, nil
		}
		return d, true, nil
	}

	tf := flag.Lookup("test.timeout")
	if tf == nil {
		return 0, false, nil
	}

	d, err := time.ParseDuration(tf.Value.String())
	if err != nil || d <= 0 {
		return 0, false, nil
	}

	soft := d - time.Minute
	if soft < time.Minute {
		soft = d / 2
	}
	if soft <= 0 {
		return 0, false, nil
	}

	return soft, true, nil
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests are skipped unless TF_ACC=1")
	}

	if os.Getenv("UPTIMEROBOT_API_KEY") == "" {
		t.Fatal("UPTIMEROBOT_API_KEY must be set for acceptance tests when TF_ACC=1")
	}
}

func testAccProviderConfig() string {
	if !testAccNeedsExplicitProviderSource() {
		return fmt.Sprintf(`
provider "uptimerobot" {
  api_key = "%s"
}
`, os.Getenv("UPTIMEROBOT_API_KEY"))
	}

	source := testAccProviderSource()
	version, hasVersion := testAccProviderVersion()

	requiredProvider := fmt.Sprintf(`
    uptimerobot = {
      source = "%s"
    }`, source)
	if hasVersion {
		requiredProvider = fmt.Sprintf(`
    uptimerobot = {
      source  = "%s"
      version = "%s"
    }`, source, version)
	}

	return fmt.Sprintf(`
terraform {
  required_providers {
%s
  }
}

provider "uptimerobot" {
  api_key = "%s"
}
`, requiredProvider, os.Getenv("UPTIMEROBOT_API_KEY"))
}

func testAccNeedsExplicitProviderSource() bool {
	return os.Getenv("TF_ACC_PROVIDER_HOST") != "" ||
		os.Getenv("TF_ACC_PROVIDER_NAMESPACE") != "" ||
		os.Getenv("TF_ACC_PROVIDER_VERSION") != ""
}

func testAccProviderSource() string {
	namespace := os.Getenv("TF_ACC_PROVIDER_NAMESPACE")
	if namespace == "" {
		namespace = "uptimerobot"
	}

	host := os.Getenv("TF_ACC_PROVIDER_HOST")
	if host == "" {
		return fmt.Sprintf("%s/uptimerobot", namespace)
	}

	return fmt.Sprintf("%s/%s/uptimerobot", host, namespace)
}

func testAccProviderVersion() (string, bool) {
	version := os.Getenv("TF_ACC_PROVIDER_VERSION")
	if version == "" {
		return "", false
	}

	return version, true
}

// CheckDestroy functions for each resource type.
func testAccCheckMonitorDestroy(s *terraform.State) error {
	// Create a client to check if the monitor was properly deleted
	apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_monitor" {
			continue
		}

		// Convert ID from string to int64
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("Error converting monitor ID to int64: %v", err)
		}

		// Try to get the monitor - it should not exist
		err = apiClient.WaitMonitorDeleted(ctx, id, 90*time.Second)
		if err != nil {
			return fmt.Errorf("Monitor %s still exists: %w", rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckIntegrationDestroy(s *terraform.State) error {
	// Create a client to check if the integration was properly deleted
	apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_integration" {
			continue
		}

		// Convert ID from string to int64
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("Error converting integration ID to int64: %v", err)
		}

		// Try to get the integration - it should not exist
		err = apiClient.WaitIntegrationDeleted(ctx, id, 90*time.Second)
		if err != nil {
			return fmt.Errorf("Integration %s still exists: %w", rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckMaintenanceWindowDestroy(s *terraform.State) error {
	// Create a client to check if the maintenance window was properly deleted
	apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_maintenance_window" {
			continue
		}

		// Convert ID from string to int64
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("Error converting maintenance window ID to int64: %v", err)
		}

		// Try to get the maintenance window - it should not exist
		err = apiClient.WaitMaintenanceWindowDeleted(ctx, id, 90*time.Second)
		if err != nil {
			return fmt.Errorf("Maintenance window %s still exists: %w", rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckPSPDestroy(s *terraform.State) error {
	// Create a client to check if the PSP was properly deleted
	apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_psp" {
			continue
		}

		// Convert ID from string to int64
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("Error converting PSP ID to int64: %v", err)
		}

		// Try to get the PSP - it should not exist
		err = apiClient.WaitPSPDeleted(ctx, id, 90*time.Second)
		if err != nil {
			return fmt.Errorf("PSP %s still exists: %w", rs.Primary.ID, err)
		}
	}

	return nil
}
