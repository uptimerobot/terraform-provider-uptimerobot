package alertcontact

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestAlertContactResource_Metadata(t *testing.T) {
	t.Parallel()

	r := NewResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "uptimerobot"}, resp)

	if resp.TypeName != "uptimerobot_alert_contact" {
		t.Fatalf("unexpected type name %q", resp.TypeName)
	}
}

func TestAlertContactResource_Schema(t *testing.T) {
	t.Parallel()

	r := NewResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	for _, name := range []string{
		"id",
		"name",
		"type",
		"value",
		"notification_events",
		"ssl_expiration_reminder",
		"is_active",
		"status",
		"mobile_provider_id",
		"org_alert_contact_id",
		"one_signal_subscription_id",
		"one_signal_user_id",
		"device_fingerprint",
		"push_token",
		"android_push_up_channel",
		"android_push_down_channel",
	} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Fatalf("expected schema attribute %q", name)
		}
	}
}

func TestBuildCreateAlertContactRequestEmail(t *testing.T) {
	t.Parallel()

	req := buildCreateAlertContactRequest(alertContactResourceModel{
		Name:               types.StringValue("Work email"),
		Type:               types.StringValue("email"),
		Value:              types.StringValue("user@example.com"),
		NotificationEvents: types.StringValue("down"),
	})

	if req.Type != "Email" {
		t.Fatalf("unexpected type %q", req.Type)
	}
	if req.FriendlyName != "Work email" || req.Value != "user@example.com" {
		t.Fatalf("unexpected email request: %#v", req)
	}
	if req.EnableNotificationsFor != "Down" {
		t.Fatalf("unexpected notification events %q", req.EnableNotificationsFor)
	}
	if req.Platform != "" || req.OneSignalSubscriptionID != "" {
		t.Fatalf("email request should not include mobile fields: %#v", req)
	}
}

func TestBuildCreateAlertContactRequestMobile(t *testing.T) {
	t.Parallel()

	req := buildCreateAlertContactRequest(alertContactResourceModel{
		Name:                    types.StringValue("Pixel"),
		Type:                    types.StringValue("mobile_app"),
		NotificationEvents:      types.StringValue("up_and_down"),
		OneSignalSubscriptionID: types.StringValue("sub-1"),
		OneSignalUserID:         types.StringValue("user-1"),
		DeviceFingerprint:       types.StringValue("fingerprint-1"),
		PushToken:               types.StringValue("push-token"),
		AndroidPushUpChannel:    types.StringValue("up"),
		AndroidPushDownChannel:  types.StringValue("down"),
	})

	if req.Type != "MobileApp" || req.Platform != "android" {
		t.Fatalf("unexpected mobile type/platform: %#v", req)
	}
	if req.DeviceName != "Pixel" || req.OneSignalSubscriptionID != "sub-1" || req.OneSignalUserID != "user-1" {
		t.Fatalf("unexpected mobile request: %#v", req)
	}
	if req.Config == nil || req.Config.AndroidPushUpChannel != "up" || req.Config.AndroidPushDownChannel != "down" {
		t.Fatalf("unexpected mobile config: %#v", req.Config)
	}
}

func TestBuildCreateAlertContactRequestMobileOld(t *testing.T) {
	t.Parallel()

	req := buildCreateAlertContactRequest(alertContactResourceModel{
		Name:                    types.StringValue("iPhone"),
		Type:                    types.StringValue("mobile_app_old"),
		NotificationEvents:      types.StringValue("down"),
		OneSignalSubscriptionID: types.StringValue("sub-ios"),
		OneSignalUserID:         types.StringValue("user-ios"),
		DeviceFingerprint:       types.StringValue("fingerprint-ios"),
		PushToken:               types.StringValue("push-ios"),
	})

	if req.Type != "MobileAppOld" || req.Platform != "ios" {
		t.Fatalf("unexpected iOS type/platform: %#v", req)
	}
	if req.DeviceName != "iPhone" || req.OneSignalSubscriptionID != "sub-ios" || req.OneSignalUserID != "user-ios" {
		t.Fatalf("unexpected iOS request: %#v", req)
	}
	if req.DeviceFingerprint != "fingerprint-ios" || req.PushToken != "push-ios" {
		t.Fatalf("unexpected iOS identity fields: %#v", req)
	}
	if req.Config != nil {
		t.Fatalf("expected no Android config for iOS contact, got %#v", req.Config)
	}
}

func TestBuildUpdateAlertContactRequestIsActive(t *testing.T) {
	t.Parallel()

	base := alertContactResourceModel{
		Name:                  types.StringValue("Work email"),
		NotificationEvents:    types.StringValue("down"),
		SSLExpirationReminder: types.BoolValue(true),
	}

	inactivePlan := base
	inactiveConfig := base
	inactivePlan.IsActive = types.BoolValue(false)
	inactiveConfig.IsActive = types.BoolValue(false)
	inactiveReq := buildUpdateAlertContactRequest(inactivePlan, inactiveConfig)
	if inactiveReq.IsActive == nil || *inactiveReq.IsActive {
		t.Fatalf("expected inactive update request, got %#v", inactiveReq.IsActive)
	}

	activePlan := base
	activeConfig := base
	activePlan.IsActive = types.BoolValue(true)
	activeConfig.IsActive = types.BoolValue(true)
	activeReq := buildUpdateAlertContactRequest(activePlan, activeConfig)
	if activeReq.IsActive == nil || !*activeReq.IsActive {
		t.Fatalf("expected active update request, got %#v", activeReq.IsActive)
	}

	for _, isActive := range []types.Bool{types.BoolNull(), types.BoolUnknown()} {
		plan := base
		config := base
		plan.IsActive = types.BoolValue(true)
		config.IsActive = isActive
		req := buildUpdateAlertContactRequest(plan, config)
		if req.IsActive != nil {
			t.Fatalf("expected omitted active state for %#v, got %#v", isActive, req.IsActive)
		}
	}
}

func TestValidateAlertContactResourceCreate(t *testing.T) {
	t.Parallel()

	var emailDiags diag.Diagnostics
	validateAlertContactResourceCreate(alertContactResourceModel{
		Type: types.StringValue("email"),
	}, &emailDiags)
	if !emailDiags.HasError() || !strings.Contains(emailDiags[0].Summary(), "Missing email address") {
		t.Fatalf("expected missing email diagnostic, got %#v", emailDiags)
	}

	var mobileDiags diag.Diagnostics
	validateAlertContactResourceCreate(alertContactResourceModel{
		Type: types.StringValue("mobile_app_old"),
	}, &mobileDiags)
	if !mobileDiags.HasError() {
		t.Fatal("expected mobile diagnostics")
	}
	if got := mobileDiags.ErrorsCount(); got != 3 {
		t.Fatalf("expected 3 mobile diagnostics, got %d: %#v", got, mobileDiags)
	}

	var okDiags diag.Diagnostics
	validateAlertContactResourceCreate(alertContactResourceModel{
		Name:                    types.StringValue("iPhone"),
		Type:                    types.StringValue("mobile_app_old"),
		OneSignalSubscriptionID: types.StringValue("sub-1"),
		OneSignalUserID:         types.StringValue("user-1"),
		DeviceFingerprint:       types.StringValue("fingerprint-1"),
	}, &okDiags)
	if okDiags.HasError() {
		t.Fatalf("unexpected diagnostics: %#v", okDiags)
	}
}

func TestAlertContactResourceStateMobilePreservesHiddenIdentity(t *testing.T) {
	t.Parallel()

	prev := alertContactResourceModel{
		OneSignalUserID:   types.StringValue("user-1"),
		DeviceFingerprint: types.StringValue("fingerprint-1"),
		PushToken:         types.StringValue("push-token"),
	}
	state := alertContactResourceState(client.UserAlertContact{
		ID:                     101,
		Name:                   "Pixel",
		Type:                   "MobileApp",
		CustomValue:            "sub-1",
		EnableNotificationsFor: "Down",
		SSLExpirationReminder:  true,
		Status:                 "Active",
		Config: &client.AlertContactConfig{
			AndroidPushUpChannel:   "up",
			AndroidPushDownChannel: "down",
		},
	}, prev)

	if state.Type.ValueString() != "mobile_app" {
		t.Fatalf("unexpected type %q", state.Type.ValueString())
	}
	if !state.Value.IsNull() {
		t.Fatalf("expected mobile value to stay null, got %#v", state.Value)
	}
	if state.OneSignalSubscriptionID.ValueString() != "sub-1" {
		t.Fatalf("unexpected subscription id %q", state.OneSignalSubscriptionID.ValueString())
	}
	if state.OneSignalUserID.ValueString() != "user-1" || state.DeviceFingerprint.ValueString() != "fingerprint-1" {
		t.Fatalf("hidden identity was not preserved: %#v", state)
	}
	if state.PushToken.ValueString() != "push-token" {
		t.Fatalf("push token was not preserved: %#v", state.PushToken)
	}
	if state.AndroidPushDownChannel.ValueString() != "down" {
		t.Fatalf("unexpected down channel %q", state.AndroidPushDownChannel.ValueString())
	}
	if !state.IsActive.ValueBool() {
		t.Fatalf("expected active contact state, got %#v", state.IsActive)
	}
}

func TestAlertContactIsActiveState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   string
		prev     types.Bool
		want     bool
		wantNull bool
	}{
		{name: "active", status: "Active", want: true},
		{name: "paused", status: "Paused", want: false},
		{name: "not activated", status: "NotActivated", want: false},
		{name: "to migrate", status: "ToMigrate", want: false},
		{name: "unknown preserves previous", status: "Unexpected", prev: types.BoolValue(true), want: true},
		{name: "unknown without previous", status: "Unexpected", wantNull: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := alertContactIsActiveState(tt.status, tt.prev)
			if tt.wantNull {
				if !got.IsNull() {
					t.Fatalf("expected null active state, got %#v", got)
				}
				return
			}
			if got.IsNull() || got.IsUnknown() || got.ValueBool() != tt.want {
				t.Fatalf("expected active state %v, got %#v", tt.want, got)
			}
		})
	}
}

func TestAlertContactNotificationEventsToAPI(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"up_and_down": "UpAndDown",
		"down":        "Down",
		"up":          "Up",
		"none":        "None",
		"":            "UpAndDown",
	}

	for in, want := range tests {
		if got := alertContactNotificationEventsToAPI(in); got != want {
			t.Fatalf("alertContactNotificationEventsToAPI(%q) = %q, want %q", in, got, want)
		}
	}
}
