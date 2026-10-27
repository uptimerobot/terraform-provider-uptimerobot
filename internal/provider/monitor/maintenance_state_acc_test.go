//go:build acceptance

package monitor_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func TestAccMonitorResource_MaintenanceWindowOwnership(t *testing.T) {
	name := provideracctest.RandomName("acc-mw-ownership")
	url := provideracctest.UniqueURL(name)
	config := func(suffix, monitorIDs, windowMonitors string) string {
		return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  type     = "HTTP"
  url      = %q
  interval = 300
  timeout  = 30
  %s
}

resource "uptimerobot_maintenance_window" "test" {
  name              = %q
  interval          = "weekly"
  time              = "09:00:00"
  days              = [2, 4]
  duration          = 120
  auto_add_monitors = false
  %s
}
`, name+suffix, url, monitorIDs, name, windowMonitors)
	}
	external := config("", "", "monitor_ids = [uptimerobot_monitor.test.id]")
	renamed := config("-renamed", "", "monitor_ids = [uptimerobot_monitor.test.id]")
	explicit := config("-renamed", "maintenance_window_ids = [uptimerobot_maintenance_window.test.id]", "")
	omitted := config("-unmanaged", "", "")
	cleared := config("-unmanaged", "maintenance_window_ids = []", "")
	var windowID string
	checkMembership := func(attached bool) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			monitor, ok := s.RootModule().Resources["uptimerobot_monitor.test"]
			if !ok || monitor.Primary == nil {
				return fmt.Errorf("monitor missing from state")
			}
			window, ok := s.RootModule().Resources["uptimerobot_maintenance_window.test"]
			if !ok || window.Primary == nil {
				return fmt.Errorf("maintenance window missing from state")
			}
			windowID = window.Primary.ID
			monitorID, err := strconv.ParseInt(monitor.Primary.ID, 10, 64)
			if err != nil {
				return err
			}
			apiClient := provideracctest.APIClient()
			return retry.RetryContext(context.Background(), 90*time.Second, func() *retry.RetryError {
				remote, err := apiClient.GetMonitor(context.Background(), monitorID)
				if err != nil {
					return retry.NonRetryableError(err)
				}
				found := false
				for _, mw := range remote.MaintenanceWindows {
					if strconv.FormatInt(mw.ID, 10) == windowID {
						found = true
					}
				}
				if found != attached {
					return retry.RetryableError(fmt.Errorf("maintenance assignment: got %t, want %t", found, attached))
				}
				return nil
			})
		}
	}
	unmanaged := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids.#"),
		checkMembership(true),
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			provideracctest.CheckMonitorDestroy,
			provideracctest.CheckMaintenanceWindowDestroy,
		),
		Steps: []resource.TestStep{
			{Config: external, Check: unmanaged},
			{Config: renamed, Check: unmanaged},
			{Config: renamed, PlanOnly: true, ExpectNonEmptyPlan: false},
			{
				Config: explicit,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair("uptimerobot_monitor.test", "maintenance_window_ids.*", "uptimerobot_maintenance_window.test", "id"),
					checkMembership(true),
				),
			},
			{
				ResourceName: "uptimerobot_monitor.test",
				ImportState:  true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 || states[0].Attributes["maintenance_window_ids.#"] != "1" {
						return fmt.Errorf("expected one imported maintenance-window assignment")
					}
					for key, value := range states[0].Attributes {
						if strings.HasPrefix(key, "maintenance_window_ids.") && key != "maintenance_window_ids.#" && value == windowID {
							return nil
						}
					}
					return fmt.Errorf("import did not preserve the assigned maintenance window")
				},
			},
			{Config: omitted, Check: unmanaged},
			{Config: omitted, PlanOnly: true, ExpectNonEmptyPlan: false},
			{
				Config: cleared,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids.#", "0"),
					checkMembership(false),
				),
			},
			{Config: cleared, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}
