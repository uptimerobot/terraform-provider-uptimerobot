//go:build acceptance

package monitor_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func TestAccMonitorResource_HTTPCredentials(t *testing.T) {
	for _, monitorType := range []string{"HTTP", "KEYWORD", "API"} {
		t.Run(monitorType, func(t *testing.T) {
			name := provideracctest.RandomName("acc-http-credentials")
			var id string
			config := func(suffix, credentials string) string {
				return httpCredentialsConfig(name, suffix, monitorType, credentials)
			}
			check := func(username, password, auth string) resource.TestCheckFunc {
				return checkHTTPCredentials(t, &id, username, password, auth)
			}
			credentials := `http_username = "user"
http_password = "password"`
			empty := `auth_type = "NONE"
http_username = ""
http_password = ""`
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { provideracctest.PreCheck(t) },
				ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
				CheckDestroy:             provideracctest.CheckMonitorDestroy,
				Steps: []resource.TestStep{
					{Config: config("", credentials), Check: check("user", "password", "HTTP_BASIC")},
					{Config: config("", ""), PlanOnly: true},
					{Config: config("-omitted", ""), Check: check("user", "password", "HTTP_BASIC")},
					{Config: config("-omitted", "http_username = null\nhttp_password = null"), PlanOnly: true},
					{Config: config("-omitted", `http_password = ""`), Check: check("user", "", "HTTP_BASIC")},
					{Config: config("-omitted", `http_username = ""`), Check: check("", "", "HTTP_BASIC")},
					{Config: config("-omitted", credentials), Check: check("user", "password", "HTTP_BASIC")},
					{Config: config("-omitted", `auth_type = "NONE"`), Check: check("user", "password", "NONE")},
					{Config: config("-omitted", empty), Check: check("", "", "NONE")},
					{Config: config("-omitted", empty), PlanOnly: true},
					{Config: config("-omitted", `auth_type = "NONE"`), PlanOnly: true},
					{Config: config("-unmanaged", `auth_type = "NONE"`), Check: check("", "", "NONE")},
					{Config: config("-unmanaged", credentials), Check: check("user", "password", "HTTP_BASIC")},
					{ResourceName: "uptimerobot_monitor.test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"http_password", "status"},
						// Runtime health status may change between reads and is not configuration.
						// Import has no prior state to distinguish a stale API response.
						// A matching read from one process does not expire another process's
						// 60-second response cache. Keep the strict import checks after expiry.
						PreConfig: func() {
							select {
							case <-t.Context().Done():
								t.Fatal(t.Context().Err())
							case <-time.After(65 * time.Second):
							}
						},
						ImportStateCheck: func(states []*terraform.InstanceState) error {
							for _, state := range states {
								if state.Attributes["http_password"] != "" {
									return fmt.Errorf("import must not copy the remote password into state")
								}
							}
							return nil
						}},
					{Config: config("-unmanaged", credentials), PlanOnly: true},
				},
			})
		})
	}
}

func TestAccMonitorResource_HTTPCredentialsDrift(t *testing.T) {
	name := provideracctest.RandomName("acc-credential-drift")
	var id string
	config := func(suffix, credentials string) string {
		return httpCredentialsConfig(name, suffix, "HTTP", credentials)
	}
	check := func(username, password string) resource.TestCheckFunc {
		return checkHTTPCredentials(t, &id, username, password, "HTTP_BASIC")
	}
	mutate := func(username, password string) func() {
		return func() {
			monitorID, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			_, err = provideracctest.APIClient().UpdateMonitor(t.Context(), monitorID, &client.UpdateMonitorRequest{
				Name: name, Type: client.MonitorTypeHTTP, Interval: 300, HTTPUsername: &username, HTTPPassword: &password,
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := waitHTTPCredentials(t.Context(), monitorID, username, password, "HTTP_BASIC"); err != nil {
				t.Fatal(err)
			}
		}
	}
	credentials := `http_username = "managed"
http_password = "managed"`
	ignored := `http_username = "ignored"
http_password = "ignored"
lifecycle { ignore_changes = [http_username, http_password] }`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckMonitorDestroy,
		Steps: []resource.TestStep{
			{Config: config("", ""), Check: check("", "")},
			{Config: config("", ""), PlanOnly: true},
			{Config: config("-changed", ""), PreConfig: mutate("external", "external-password"), Check: check("external", "external-password")},
			{Config: config("-changed", ""), PlanOnly: true},
			{Config: config("", credentials), Check: check("managed", "managed")},
			{Config: config("-ignored", ignored), PreConfig: mutate("external", "external-password"), Check: check("external", "external-password")},
			{Config: config("-ignored", ignored), PlanOnly: true},
			{Config: config("-ignored", `http_username = ""`), Check: check("", "external-password")},
			{Config: config("-ignored", `http_username = ""`), PreConfig: mutate("drifted", "external-password"), Check: check("", "external-password")},
			{Config: config("-ignored", `http_username = ""`), PlanOnly: true},
		},
	})
}

func TestAccMonitorResource_HTTPCredentialsDeferredAndEmpty(t *testing.T) {
	name := provideracctest.RandomName("acc-credential-deferred")
	var id string
	deferred := func(value string) string {
		return fmt.Sprintf("resource \"terraform_data\" \"credentials\" { input = %q }\n", value) + httpCredentialsConfig(name, "", "HTTP", `
http_username = terraform_data.credentials.output
http_password = terraform_data.credentials.output`)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckMonitorDestroy,
		Steps: []resource.TestStep{
			{Config: deferred(""), Check: checkHTTPCredentials(t, &id, "", "", "HTTP_BASIC")},
			{Config: deferred(""), PlanOnly: true},
			{Config: deferred(" user:päss "), Check: checkHTTPCredentials(t, &id, " user:päss ", " user:päss ", "HTTP_BASIC")},
			{Config: deferred(""), Check: checkHTTPCredentials(t, &id, "", "", "HTTP_BASIC")},
			{Config: deferred(""), PlanOnly: true},
		},
	})
}

func TestAccMonitorResource_HTTPCredentialsUpgrade(t *testing.T) {
	name := provideracctest.RandomName("acc-credential-upgrade")
	var id string
	configured := httpCredentialsConfig(name, "", "HTTP", `http_username = "user"
http_password = "password"`)
	omitted := httpCredentialsConfig(name, "", "HTTP", "")
	cleared := httpCredentialsConfig(name, "", "HTTP", `http_username = ""
http_password = ""`)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { provideracctest.PreCheck(t) },
		CheckDestroy: provideracctest.CheckMonitorDestroy,
		Steps: []resource.TestStep{
			{Config: configured, ExternalProviders: map[string]resource.ExternalProvider{"uptimerobot": {Source: "uptimerobot/uptimerobot", VersionConstraint: "1.13.2"}}, Check: checkHTTPCredentials(t, &id, "user", "password", "HTTP_BASIC")},
			{Config: omitted, ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories, PlanOnly: true},
			{Config: cleared, ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories, Check: checkHTTPCredentials(t, &id, "", "", "HTTP_BASIC")},
			{Config: cleared, ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories, PlanOnly: true},
			{Config: omitted, ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories, PlanOnly: true},
		},
	})
}

func httpCredentialsConfig(name, suffix, monitorType, credentials string) string {
	extra := ""
	if monitorType == "KEYWORD" {
		extra = `keyword_type = "ALERT_NOT_EXISTS"
keyword_value = "Example Domain"
keyword_case_type = "CaseInsensitive"`
	}
	if monitorType == "API" {
		extra = `config = {
  api_assertions = {
    logic = "AND"
    checks = [{ property = "status_code", comparison = "equals", target = jsonencode(200) }]
  }
}`
	}
	return fmt.Sprintf(`
provider "uptimerobot" {}
resource "uptimerobot_monitor" "test" {
  name = %q
  url = %q
  type = %q
  interval = 300
  timeout = 30
  group_id = 0
  assigned_alert_contacts = []
  %s
  %s
}
`, name+suffix, provideracctest.UniqueURL(name), monitorType, extra, credentials)
}

func checkHTTPCredentials(t *testing.T, id *string, username, password, auth string) resource.TestCheckFunc {
	t.Helper()
	return func(state *terraform.State) error {
		monitor, ok := state.RootModule().Resources["uptimerobot_monitor.test"]
		if !ok {
			return fmt.Errorf("monitor missing from state")
		}
		if *id == "" {
			*id = monitor.Primary.ID
		}
		if monitor.Primary.ID != *id {
			return fmt.Errorf("credential update replaced the monitor")
		}
		if monitor.Primary.Attributes["http_username"] != username {
			return fmt.Errorf("username in state does not match")
		}
		monitorID, err := strconv.ParseInt(*id, 10, 64)
		if err != nil {
			return err
		}
		return waitHTTPCredentials(t.Context(), monitorID, username, password, auth)
	}
}

func waitHTTPCredentials(ctx context.Context, id int64, username, password, auth string) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	for {
		monitor, err := provideracctest.APIClient().GetMonitor(ctx, id)
		if err == nil && monitor.HTTPUsername == username && monitor.HTTPPassword == password && monitor.AuthType == auth {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("API credentials did not match: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}
}
