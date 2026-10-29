//go:build acceptance

package psp_test

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

func testAccPSPMonitorGroupsConfig(name, page string) string {
	settings := ""
	if page != "" {
		settings = "custom_settings = { page = { " + page + " } }"
	}
	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name = %q
  monitor_ids = []
  %s
}

data "uptimerobot_psp" "by_id" {
  id = uptimerobot_psp.test.id
}

data "uptimerobot_psp" "by_name" {
  name = uptimerobot_psp.test.name
  depends_on = [uptimerobot_psp.test]
}
`, name, settings)
}

func testAccCheckPSPMonitorGroups(want bool, theme string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["uptimerobot_psp.test"]
		if !ok {
			return fmt.Errorf("PSP missing from state")
		}
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		remote, err := provideracctest.APIClient().GetPSP(ctx, id)
		if err != nil {
			return err
		}
		if remote.CustomSettings == nil || remote.CustomSettings.Page == nil || remote.CustomSettings.Page.ShowMonitorsUnderGroups == nil {
			return fmt.Errorf("API omitted showMonitorsUnderGroups")
		}
		page := remote.CustomSettings.Page
		if *page.ShowMonitorsUnderGroups != want || (theme != "" && page.Theme != theme) {
			return fmt.Errorf("API grouping=%t theme=%q, want %t %q", *page.ShowMonitorsUnderGroups, page.Theme, want, theme)
		}
		return nil
	}
}

func TestAccPSPResource_ShowMonitorGroups(t *testing.T) {
	name := provideracctest.RandomName("acc-psp-groups")
	const attr = "custom_settings.page.show_monitor_groups"
	checks := func(configured bool, remote bool, theme string) resource.TestCheckFunc {
		stateCheck := resource.TestCheckNoResourceAttr("uptimerobot_psp.test", attr)
		if configured {
			stateCheck = resource.TestCheckResourceAttr("uptimerobot_psp.test", attr, strconv.FormatBool(remote))
		}
		return resource.ComposeAggregateTestCheckFunc(
			stateCheck,
			resource.TestCheckResourceAttr("data.uptimerobot_psp.by_id", attr, strconv.FormatBool(remote)),
			resource.TestCheckResourceAttr("data.uptimerobot_psp.by_name", attr, strconv.FormatBool(remote)),
			testAccCheckPSPMonitorGroups(remote, theme),
		)
	}
	importStep := func(want bool) resource.TestStep {
		return resource.TestStep{
			ResourceName: "uptimerobot_psp.test",
			ImportState:  true,
			ImportStateCheck: func(states []*terraform.InstanceState) error {
				if len(states) != 1 || states[0].Attributes[attr] != strconv.FormatBool(want) {
					return fmt.Errorf("import did not retain show_monitor_groups=%t", want)
				}
				return nil
			},
		}
	}
	enabled := testAccPSPMonitorGroupsConfig(name, `show_monitor_groups = true, theme = "dark"`)
	omitted := testAccPSPMonitorGroupsConfig(name, `theme = "light"`)
	var id int64
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckPSPDestroy,
		Steps: []resource.TestStep{
			{Config: testAccPSPMonitorGroupsConfig(name, `show_monitor_groups = false`), Check: checks(true, false, "")},
			importStep(false),
			{Config: enabled, Check: resource.ComposeAggregateTestCheckFunc(checks(true, true, "dark"), func(state *terraform.State) error {
				var err error
				id, err = strconv.ParseInt(state.RootModule().Resources["uptimerobot_psp.test"].Primary.ID, 10, 64)
				return err
			})},
			importStep(true),
			{Config: enabled, PlanOnly: true, ExpectNonEmptyPlan: false},
			{
				PreConfig: func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					value := false
					_, err := provideracctest.APIClient().UpdatePSP(ctx, id, &client.UpdatePSPRequest{CustomSettings: &client.CustomSettings{Page: &client.PageSettings{ShowMonitorsUnderGroups: &value}}})
					if err != nil {
						t.Fatal(err)
					}
				},
				Config: enabled, PlanOnly: true, ExpectNonEmptyPlan: true,
			},
			{Config: enabled, Check: checks(true, true, "dark")},
			{Config: omitted, Check: checks(false, true, "light")},
			{Config: omitted, PlanOnly: true, ExpectNonEmptyPlan: false},
			{Config: testAccPSPMonitorGroupsConfig(name, `show_monitor_groups = null, theme = "light"`), PlanOnly: true, ExpectNonEmptyPlan: false},
			{Config: testAccPSPMonitorGroupsConfig(name, `show_monitor_groups = false, theme = "light"`), Check: checks(true, false, "light")},
			{Config: testAccPSPMonitorGroupsConfig(name+"-unmanaged", ""), Check: checks(false, false, "light")},
			{Config: testAccPSPMonitorGroupsConfig(name+"-unmanaged", ""), PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}
