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
`, name, settings)
}

func testAccPSPMonitorGroupsDataSourcesConfig(name, page string) string {
	return testAccPSPMonitorGroupsConfig(name, page) + `
data "uptimerobot_psp" "by_id" {
  id = uptimerobot_psp.test.id
}

data "uptimerobot_psp" "by_name" {
  name = uptimerobot_psp.test.name
  depends_on = [uptimerobot_psp.test]
}
`
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
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		return testAccWaitPSPMonitorGroups(ctx, id, want, theme)
	}
}

// The public API keeps process-local PSP responses for 60 seconds. A successful
// write and several matching reads can still be followed by an older cached
// response from another API process. Let that cache window expire before testing
// refresh/import or reading data sources, while still checking the actual value.
func testAccWaitPSPMonitorGroups(ctx context.Context, id int64, want bool, theme string) error {
	apiClient := provideracctest.APIClient()
	visibleAfter := time.Now().Add(65 * time.Second)
	var last string
	consecutiveMatches := 0
	for {
		remote, err := apiClient.GetPSP(ctx, id)
		if err != nil {
			return err
		}
		if remote.CustomSettings == nil || remote.CustomSettings.Page == nil || remote.CustomSettings.Page.ShowMonitorsUnderGroups == nil {
			last = "API omitted showMonitorsUnderGroups"
			consecutiveMatches = 0
		} else {
			page := remote.CustomSettings.Page
			last = fmt.Sprintf("API grouping=%t theme=%q, want %t %q", *page.ShowMonitorsUnderGroups, page.Theme, want, theme)
			if *page.ShowMonitorsUnderGroups == want && (theme == "" || page.Theme == theme) {
				consecutiveMatches++
				if consecutiveMatches >= 3 && !time.Now().Before(visibleAfter) {
					return nil
				}
			} else {
				consecutiveMatches = 0
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("PSP settings did not become visible: %s: %w", last, ctx.Err())
		case <-time.After(5 * time.Second):
		}
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
					ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
					defer cancel()
					value := false
					_, err := provideracctest.APIClient().UpdatePSP(ctx, id, &client.UpdatePSPRequest{CustomSettings: &client.CustomSettings{Page: &client.PageSettings{ShowMonitorsUnderGroups: &value}}})
					if err != nil {
						t.Fatal(err)
					}
					if err := testAccWaitPSPMonitorGroups(ctx, id, false, "dark"); err != nil {
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

func TestAccPSPDataSource_ShowMonitorGroups(t *testing.T) {
	name := provideracctest.RandomName("acc-psp-group-lookups")
	var steps []resource.TestStep
	for _, value := range []bool{false, true} {
		page := "show_monitor_groups = " + strconv.FormatBool(value)
		steps = append(steps,
			resource.TestStep{
				Config: testAccPSPMonitorGroupsConfig(name, page),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.page.show_monitor_groups", strconv.FormatBool(value)),
					testAccCheckPSPMonitorGroups(value, ""),
				),
			},
			resource.TestStep{
				Config: testAccPSPMonitorGroupsDataSourcesConfig(name, page),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.uptimerobot_psp.by_id", "custom_settings.page.show_monitor_groups", strconv.FormatBool(value)),
					resource.TestCheckResourceAttr("data.uptimerobot_psp.by_name", "custom_settings.page.show_monitor_groups", strconv.FormatBool(value)),
					resource.TestCheckResourceAttrPair("data.uptimerobot_psp.by_id", "id", "uptimerobot_psp.test", "id"),
					resource.TestCheckResourceAttrPair("data.uptimerobot_psp.by_name", "id", "uptimerobot_psp.test", "id"),
				),
			},
		)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckPSPDestroy,
		Steps:                    steps,
	})
}
