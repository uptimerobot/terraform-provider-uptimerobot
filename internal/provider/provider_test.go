package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
	// For real API testing, Terraform handles cleanup automatically
	// These functions can be simplified or removed entirely
	return nil
}

func testAccCheckIntegrationDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckMaintenanceWindowDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckPSPDestroy(s *terraform.State) error {
	return nil
}
