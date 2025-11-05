package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

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
				MarkdownDescription: "API key for authentication. Can also be set via the `UPTIMEROBOT_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Optional API endpoint URL. If not specified, the default endpoint will be used. Can also be set via the `UPTIMEROBOT_API_URL` environment variable.",
				Optional:            true,
			},
		},
	}
}

func (p *UptimeRobotProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	apiKey := os.Getenv("UPTIMEROBOT_API_KEY")
	apiURL := os.Getenv("UPTIMEROBOT_API_URL")

	var config UptimeRobotProviderModel

	// Only try to read config if Terraform actually provided something
	if !req.Config.Raw.IsNull() {
		resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Config values take precedence over env vars
	if !config.APIKey.IsNull() && !config.APIKey.IsUnknown() {
		apiKey = config.APIKey.ValueString()
	}
	if !config.APIURL.IsNull() && !config.APIURL.IsUnknown() {
		apiURL = config.APIURL.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key Configuration",
			"While configuring the provider, the API key was not found in the configuration or the UPTIMEROBOT_API_KEY environment variable.",
		)
		return
	}

	ua := fmt.Sprintf("terraform-provider-uptimerobot/%s Terraform/%s",
		p.version, strings.TrimSpace(req.TerraformVersion))

	client := client.NewClient(apiKey)
	client.SetUserAgent(ua)
	client.AddHeader("X-Terraform-Provider", "uptimerobot/"+p.version)

	// Override the default endpoint if specified in config or environment
	if apiURL == "" {
		apiURL = "https://api.uptimerobot.com/v3"
	}
	client.SetBaseURL(apiURL)

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
