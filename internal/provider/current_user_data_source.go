package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ datasource.DataSource              = &currentUserDataSource{}
	_ datasource.DataSourceWithConfigure = &currentUserDataSource{}
)

// NewCurrentUserDataSource returns the current user data source.
func NewCurrentUserDataSource() datasource.DataSource {
	return &currentUserDataSource{}
}

type currentUserDataSource struct {
	client *client.Client
}

type currentUserDataSourceModel struct {
	ID                         types.String `tfsdk:"id"`
	Email                      types.String `tfsdk:"email"`
	FullName                   types.String `tfsdk:"full_name"`
	MonitorsCount              types.Int64  `tfsdk:"monitors_count"`
	MonitorLimit               types.Int64  `tfsdk:"monitor_limit"`
	SMSCredits                 types.Int64  `tfsdk:"sms_credits"`
	Plan                       types.String `tfsdk:"plan"`
	SubscriptionMonitorLimit   types.Int64  `tfsdk:"subscription_monitor_limit"`
	SubscriptionExpirationDate types.String `tfsdk:"subscription_expiration_date"`
	SubscriptionStatus         types.String `tfsdk:"subscription_status"`
}

func (d *currentUserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *currentUserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_current_user"
}

func (d *currentUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads account metadata for the UptimeRobot user authenticated by the configured API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Static data source ID, always `current`.",
			},
			"email": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Email address of the authenticated user. This is marked sensitive to avoid displaying it in Terraform output, but Terraform state still stores the value.",
			},
			"full_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Full name of the authenticated user, if configured.",
			},
			"monitors_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Current number of monitors in the account.",
			},
			"monitor_limit": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Effective monitor limit for the account.",
			},
			"sms_credits": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Available SMS credits for the account.",
			},
			"plan": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current subscription plan name returned by the API.",
			},
			"subscription_monitor_limit": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Monitor limit reported on the active subscription.",
			},
			"subscription_expiration_date": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Active subscription expiration timestamp, if returned by the API.",
			},
			"subscription_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Active subscription status, if returned by the API.",
			},
		},
	}
}

func (d *currentUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data currentUserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	user, err := d.client.GetCurrentUser(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read current user", err.Error())
		return
	}

	state := currentUserDataSourceState(user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func currentUserDataSourceState(user *client.CurrentUser) currentUserDataSourceModel {
	return currentUserDataSourceModel{
		ID:                         types.StringValue("current"),
		Email:                      types.StringValue(user.Email),
		FullName:                   stringState(user.FullName),
		MonitorsCount:              types.Int64Value(user.MonitorsCount),
		MonitorLimit:               types.Int64Value(user.MonitorLimit),
		SMSCredits:                 types.Int64Value(user.SMSCredits),
		Plan:                       stringState(user.ActiveSubscription.Plan),
		SubscriptionMonitorLimit:   types.Int64Value(user.ActiveSubscription.MonitorLimit),
		SubscriptionExpirationDate: stringPtrState(user.ActiveSubscription.ExpirationDate),
		SubscriptionStatus:         stringPtrState(user.ActiveSubscription.Status),
	}
}

func stringPtrState(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return stringState(*value)
}
