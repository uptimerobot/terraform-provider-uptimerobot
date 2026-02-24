//go:build acceptance

package provider

import (
	"context"
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
