// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &UptimeRobotProvider{}
var _ provider.ProviderWithFunctions = &UptimeRobotProvider{}

// UptimeRobotProvider defines the provider implementation.
type UptimeRobotProvider struct {
	version string
}

// UptimeRobotProviderModel describes the provider data model.
type UptimeRobotProviderModel struct {
	APIKey types.String `tfsdk:"api_key"`
	APIURL types.String `tfsdk:"api_url"`
}

func (p *UptimeRobotProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "uptimerobot"
	resp.Version = p.version
}

func (p *UptimeRobotProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for authentication.",
				Required:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Optional API endpoint URL. If not specified, the default endpoint will be used.",
				Optional:            true,
			},
		},
	}
}

func (p *UptimeRobotProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config UptimeRobotProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read from environment variables if not set in configuration
	apiKey := config.APIKey.ValueString()
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key Configuration",
			"While configuring the provider, the API key was not found in the configuration. Please ensure the api_key argument is set in the provider configuration.",
		)

	}

	// Create a new client using the configuration
	client := client.NewClient(apiKey)

	// Override the default endpoint if specified in config or environment
	apiURL := config.APIURL.ValueString()
	if apiURL == "" {
		apiURL = "https://api.uptimerobot.com/v3"
	}

	client.SetBaseURL(apiURL)

	// Make the client available during DataSource and Resource Configure methods
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *UptimeRobotProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMonitorResource,
		NewPSPResource,
		NewMaintenanceWindowResource,
		NewIntegrationResource,
	}
}

func (p *UptimeRobotProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *UptimeRobotProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &UptimeRobotProvider{
			version: version,
		}
	}
}
