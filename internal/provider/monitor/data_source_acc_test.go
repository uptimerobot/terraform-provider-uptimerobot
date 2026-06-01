//go:build acceptance

package monitor_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func testAccMonitorDataSourceResourceConfig(name, tag string) string {
	url := provideracctest.UniqueURL(name)
	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = %q
  type     = "HTTP"
  interval = 300
  timeout  = 30
  tags     = [%q]
}
`, name, url, tag)
}

func testAccMonitorDataSourceConfig(name, tag string) string {
	return testAccMonitorDataSourceResourceConfig(name, tag) + `
data "uptimerobot_monitor" "by_id" {
  id = uptimerobot_monitor.test.id

  depends_on = [uptimerobot_monitor.test]
}

data "uptimerobot_monitor" "by_name" {
  name = uptimerobot_monitor.test.name

  depends_on = [uptimerobot_monitor.test]
}
`
}

func testAccCheckMonitorVisibleInList(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		apiClient := provideracctest.APIClient()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		var lastGetMonitorsErr error
		for {
			monitors, err := apiClient.GetMonitorsByName(ctx, name)
			if err != nil {
				lastGetMonitorsErr = err
			} else {
				for _, monitor := range monitors {
					if monitor.Name == name {
						return nil
					}
				}
			}

			select {
			case <-ctx.Done():
				if lastGetMonitorsErr != nil {
					return fmt.Errorf("monitor %q was not visible in list endpoint before ctx.Done; last apiClient.GetMonitorsByName error: %v: %w", name, lastGetMonitorsErr, ctx.Err())
				}
				return fmt.Errorf("monitor %q was not visible in list endpoint before timeout: %w", name, ctx.Err())
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func TestAccMonitorDataSource(t *testing.T) {
	name := provideracctest.RandomName("tf-acc-monitor-ds")
	tag := "tf-acc-monitor-ds"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorDataSourceResourceConfig(name, tag),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					testAccCheckMonitorVisibleInList(name),
				),
			},
			{
				Config: testAccMonitorDataSourceConfig(name, tag),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_monitor.by_id",
						"id",
						"uptimerobot_monitor.test",
						"id",
					),
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_monitor.by_name",
						"id",
						"uptimerobot_monitor.test",
						"id",
					),
					resource.TestCheckResourceAttr("data.uptimerobot_monitor.by_name", "name", name),
					resource.TestCheckResourceAttr("data.uptimerobot_monitor.by_name", "type", "HTTP"),
					resource.TestCheckResourceAttr("data.uptimerobot_monitor.by_name", "tags.#", "1"),
					resource.TestCheckTypeSetElemAttr("data.uptimerobot_monitor.by_name", "tags.*", tag),
					resource.TestCheckResourceAttrSet("data.uptimerobot_monitor.by_name", "status"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_monitor.by_name", "group_id"),
				),
			},
		},
	})
}
