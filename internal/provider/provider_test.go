package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Provider-level tests.
func TestProvider(t *testing.T) {
	p := New("test")()
	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)
	if resp.TypeName != "uptimerobot" {
		t.Fatal("unexpected provider type name")
	}
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
