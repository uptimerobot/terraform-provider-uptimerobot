// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing.
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

func testAccPreCheck(t *testing.T) {
	// Verify required environment variables are set
	if os.Getenv("UPTIMEROBOT_API_KEY") == "" {
		t.Fatal("UPTIMEROBOT_API_KEY must be set for acceptance tests")
	}
	if os.Getenv("UPTIMEROBOT_ORGANIZATION_ID") == "" {
		t.Fatal("UPTIMEROBOT_ORGANIZATION_ID must be set for acceptance tests")
	}
}
