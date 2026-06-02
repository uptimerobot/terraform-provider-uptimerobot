//go:build acceptance

package maintenancewindow_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func testAccMaintenanceWindowDataSourceResourceConfig(name string) string {
	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "test" {
  name              = %q
  interval          = "weekly"
  time              = "02:00:00"
  duration          = 60
  days              = [2, 4]
  auto_add_monitors = false
}
`, name)
}

func testAccMaintenanceWindowDataSourceConfig(name string) string {
	return testAccMaintenanceWindowDataSourceResourceConfig(name) + `
data "uptimerobot_maintenance_window" "by_id" {
  id = uptimerobot_maintenance_window.test.id

  depends_on = [uptimerobot_maintenance_window.test]
}

data "uptimerobot_maintenance_window" "by_name" {
  name = uptimerobot_maintenance_window.test.name

  depends_on = [uptimerobot_maintenance_window.test]
}
`
}

func testAccCheckMaintenanceWindowVisibleInList(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		apiClient := provideracctest.APIClient()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var lastListErr error
		for {
			maintenanceWindows, err := apiClient.ListAllMaintenanceWindows(ctx)
			if err != nil {
				lastListErr = err
			} else {
				for _, maintenanceWindow := range maintenanceWindows {
					if maintenanceWindow.Name == name {
						return nil
					}
				}
			}

			select {
			case <-ctx.Done():
				if lastListErr != nil {
					return fmt.Errorf("maintenance window %q was not visible in list endpoint before ctx.Done; last apiClient.ListAllMaintenanceWindows error: %v: %w", name, lastListErr, ctx.Err())
				}
				return fmt.Errorf("maintenance window %q was not visible in list endpoint before timeout: %w", name, ctx.Err())
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func TestAccMaintenanceWindowDataSource(t *testing.T) {
	name := provideracctest.RandomName("tf-acc-mw-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckMaintenanceWindowDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMaintenanceWindowDataSourceResourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "name", name),
					testAccCheckMaintenanceWindowVisibleInList(name),
				),
			},
			{
				Config: testAccMaintenanceWindowDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_maintenance_window.by_id",
						"id",
						"uptimerobot_maintenance_window.test",
						"id",
					),
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_maintenance_window.by_name",
						"id",
						"uptimerobot_maintenance_window.test",
						"id",
					),
					resource.TestCheckResourceAttr("data.uptimerobot_maintenance_window.by_name", "name", name),
					resource.TestCheckResourceAttr("data.uptimerobot_maintenance_window.by_name", "interval", "weekly"),
					resource.TestCheckResourceAttr("data.uptimerobot_maintenance_window.by_name", "time", "02:00:00"),
					resource.TestCheckResourceAttr("data.uptimerobot_maintenance_window.by_name", "duration", "60"),
					resource.TestCheckResourceAttr("data.uptimerobot_maintenance_window.by_name", "auto_add_monitors", "false"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_maintenance_window.by_name", "status"),
				),
			},
		},
	})
}
