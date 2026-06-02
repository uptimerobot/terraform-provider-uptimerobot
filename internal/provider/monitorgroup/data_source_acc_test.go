//go:build acceptance

package monitorgroup_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func testAccMonitorGroupDataSourceResourceConfig(name string) string {
	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor_group" "test" {
  name = %q
}
`, name)
}

func testAccMonitorGroupDataSourceConfig(name string) string {
	return testAccMonitorGroupDataSourceResourceConfig(name) + `
data "uptimerobot_monitor_group" "by_id" {
  id = uptimerobot_monitor_group.test.id

  depends_on = [uptimerobot_monitor_group.test]
}

data "uptimerobot_monitor_group" "by_name" {
  name = uptimerobot_monitor_group.test.name

  depends_on = [uptimerobot_monitor_group.test]
}
`
}

func testAccCheckMonitorGroupVisibleInList(resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %q not found in state", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource %q has empty ID in state", resourceName)
		}

		apiClient := provideracctest.APIClient()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var lastListErr error
		for {
			groups, err := apiClient.ListAllMonitorGroups(ctx)
			if err != nil {
				lastListErr = err
			} else {
				for _, group := range groups {
					if fmt.Sprintf("%d", group.ID) == rs.Primary.ID {
						return nil
					}
				}
			}

			select {
			case <-ctx.Done():
				if lastListErr != nil {
					return fmt.Errorf("monitor group ID %s was not visible in list endpoint before ctx.Done; last apiClient.ListAllMonitorGroups error: %v: %w", rs.Primary.ID, lastListErr, ctx.Err())
				}
				return fmt.Errorf("monitor group ID %s was not visible in list endpoint before timeout: %w", rs.Primary.ID, ctx.Err())
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func TestAccMonitorGroupDataSource(t *testing.T) {
	name := provideracctest.RandomName("tf-acc-monitor-group-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckMonitorGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorGroupDataSourceResourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor_group.test", "name", name),
					testAccCheckMonitorGroupVisibleInList("uptimerobot_monitor_group.test"),
				),
			},
			{
				Config: testAccMonitorGroupDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_monitor_group.by_id",
						"id",
						"uptimerobot_monitor_group.test",
						"id",
					),
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_monitor_group.by_name",
						"id",
						"uptimerobot_monitor_group.test",
						"id",
					),
					resource.TestCheckResourceAttr("data.uptimerobot_monitor_group.by_name", "name", name),
					resource.TestCheckResourceAttrSet("data.uptimerobot_monitor_group.by_name", "created_at"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_monitor_group.by_name", "updated_at"),
				),
			},
		},
	})
}
