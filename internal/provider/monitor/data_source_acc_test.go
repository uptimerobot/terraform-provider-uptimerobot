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

func testAccMonitorDataSourceResourceConfig(name, url, tag string, customFields map[string]string) string {
	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name          = %q
  url           = %q
  type          = "HTTP"
  interval      = 300
  timeout       = 30
  tags          = [%q]
  custom_fields = %s
}
`, name, url, tag, provideracctest.HCLStringMap(customFields))
}

func testAccMonitorDataSourceConfig(name, url, tag string, customFields map[string]string) string {
	return testAccMonitorDataSourceResourceConfig(name, url, tag, customFields) + fmt.Sprintf(`
data "uptimerobot_monitor" "by_id" {
  id = uptimerobot_monitor.test.id

  depends_on = [uptimerobot_monitor.test]
}

data "uptimerobot_monitor" "by_name" {
  name = uptimerobot_monitor.test.name

  depends_on = [uptimerobot_monitor.test]
}

data "uptimerobot_monitor" "by_filters" {
  name          = uptimerobot_monitor.test.name
  url           = uptimerobot_monitor.test.url
  tags          = [%q]
  group_id      = 0
  custom_fields = %s

  depends_on = [uptimerobot_monitor.test]
}

data "uptimerobot_monitors" "by_filters" {
  name          = uptimerobot_monitor.test.name
  url           = uptimerobot_monitor.test.url
  tags          = [%q]
  group_id      = 0
  custom_fields = %s

  depends_on = [uptimerobot_monitor.test]
}
`, tag, provideracctest.HCLStringMap(customFields), tag, provideracctest.HCLStringMap(customFields))
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
	url := provideracctest.UniqueURL(name)
	tag := "tf-acc-monitor-ds"
	customFields := map[string]string{
		"environment": "acceptance",
		"testcase":    tag,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorDataSourceResourceConfig(name, url, tag, customFields),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					testAccCheckMonitorVisibleInList(name),
				),
			},
			{
				Config: testAccMonitorDataSourceConfig(name, url, tag, customFields),
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
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_monitor.by_filters",
						"id",
						"uptimerobot_monitor.test",
						"id",
					),
					resource.TestCheckResourceAttr("data.uptimerobot_monitor.by_filters", "url", url),
					resource.TestCheckResourceAttr("data.uptimerobot_monitor.by_filters", "custom_fields.environment", "acceptance"),
					resource.TestCheckResourceAttr("data.uptimerobot_monitor.by_filters", "custom_fields.testcase", tag),
					resource.TestCheckResourceAttr("data.uptimerobot_monitors.by_filters", "ids.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_monitors.by_filters",
						"ids.0",
						"uptimerobot_monitor.test",
						"id",
					),
					resource.TestCheckResourceAttr("data.uptimerobot_monitors.by_filters", "monitors.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_monitors.by_filters",
						"monitors.0.id",
						"uptimerobot_monitor.test",
						"id",
					),
					resource.TestCheckResourceAttr("data.uptimerobot_monitors.by_filters", "monitors.0.name", name),
					resource.TestCheckResourceAttr("data.uptimerobot_monitors.by_filters", "monitors.0.url", url),
					resource.TestCheckResourceAttr("data.uptimerobot_monitors.by_filters", "monitors.0.tags.#", "1"),
					resource.TestCheckTypeSetElemAttr("data.uptimerobot_monitors.by_filters", "monitors.0.tags.*", tag),
					resource.TestCheckResourceAttr("data.uptimerobot_monitors.by_filters", "monitors.0.custom_fields.environment", "acceptance"),
					resource.TestCheckResourceAttr("data.uptimerobot_monitors.by_filters", "monitors.0.custom_fields.testcase", tag),
				),
			},
		},
	})
}
