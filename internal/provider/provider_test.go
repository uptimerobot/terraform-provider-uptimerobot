package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

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
	if os.Getenv("UPTIMEROBOT_API_KEY") == "" {
		panic("UPTIMEROBOT_API_KEY must be set for acceptance tests")
	}
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestProviderConfigure_EnvironmentVariables(t *testing.T) {
	const testAPIKey = "test-api-key-from-env"
	const testAPIURL = "http://test-api-url.com"

	t.Setenv("UPTIMEROBOT_API_KEY", testAPIKey)
	t.Setenv("UPTIMEROBOT_API_URL", testAPIURL)

	p := New("test")()
	req := provider.ConfigureRequest{}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Provider.Configure() failed with diagnostics: %v", resp.Diagnostics)
	}

	clientData := resp.ResourceData
	if clientData == nil {
		t.Fatal("ResourceData is nil, provider client was not configured")
	}

	apiClient, ok := clientData.(*client.Client)
	if !ok {
		t.Fatalf("Failed to type assert ResourceData to *client.Client, got %T", clientData)
	}

	if gotAPIKey := apiClient.ApiKey(); gotAPIKey != testAPIKey {
		t.Errorf("Expected API key to be %s, but got %s", testAPIKey, gotAPIKey)
	}

	if gotAPIURL := apiClient.BaseURL(); gotAPIURL != testAPIURL {
		t.Errorf("Expected API URL to be %s, but got %s", testAPIURL, gotAPIURL)
	}

	t.Setenv("UPTIMEROBOT_API_URL", "")
	p.Configure(context.Background(), req, resp)

	// Re-fetch the newly configured client instance.
	clientData = resp.ResourceData
	apiClient, ok = clientData.(*client.Client)
	if !ok {
		t.Fatalf("Failed to type assert ResourceData to *client.Client, got %T", clientData)
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("Provider.Configure() after clearing env failed with diagnostics: %v", resp.Diagnostics)
	}
	if gotAPIURL := apiClient.BaseURL(); gotAPIURL != "https://api.uptimerobot.com/v3" {
		t.Errorf("Expected API URL to be the default, but got %s", gotAPIURL)
	}
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
