package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
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

// Provider-level tests.
func TestProvider(t *testing.T) {
	p := New("test")()
	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)
	if resp.TypeName != "uptimerobot" {
		t.Fatal("unexpected provider type name")
	}
}

func testAccPreCheck() {
	// Ensure required environment variables are set for real API testing
	if os.Getenv("UPTIMEROBOT_API_KEY") == "" {
		panic("UPTIMEROBOT_API_KEY must be set for acceptance tests")
	}
	// UPTIMEROBOT_ORGANIZATION_ID is optional and not required for testing
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

func testAccProviderConfig() string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    uptimerobot = {
      source = "hashicorp/uptimerobot"
    }
  }
}

provider "uptimerobot" {
  api_key = "%s"
}
`, os.Getenv("UPTIMEROBOT_API_KEY"))
}

// CheckDestroy functions for each resource type.
func testAccCheckMonitorDestroy(s *terraform.State) error {
	// Create a client to check if the monitor was properly deleted
	apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))

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
		_, err = apiClient.GetMonitor(id)
		if err == nil {
			return fmt.Errorf("Monitor %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckIntegrationDestroy(s *terraform.State) error {
	// Create a client to check if the integration was properly deleted
	apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))

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
		_, err = apiClient.GetIntegration(id)
		if err == nil {
			return fmt.Errorf("Integration %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckMaintenanceWindowDestroy(s *terraform.State) error {
	// Create a client to check if the maintenance window was properly deleted
	apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))

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
		_, err = apiClient.GetMaintenanceWindow(id)
		if err == nil {
			return fmt.Errorf("Maintenance window %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckPSPDestroy(s *terraform.State) error {
	// Create a client to check if the PSP was properly deleted
	apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))

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
		_, err = apiClient.GetPSP(id)
		if err == nil {
			return fmt.Errorf("PSP %s still exists", rs.Primary.ID)
		}
	}

	return nil
}
