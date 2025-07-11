package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccMaintenanceWindowResourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "test" {
  name = %q
  interval = "weekly"
  time     = "01:00:00"
  duration = 60
}
`, name)
}

func TestAccMaintenanceWindowResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMaintenanceWindowDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMaintenanceWindowResourceConfig("test-maintenance-window"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "name", "test-maintenance-window"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "interval", "weekly"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "time", "01:00:00"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "duration", "60"),
				),
			},
			// Update testing
			{
				Config: testAccMaintenanceWindowResourceConfig("test-maintenance-window-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "name", "test-maintenance-window-updated"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "interval", "weekly"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "time", "01:00:00"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "duration", "60"),
				),
			},
			// Import testing
			{
				ResourceName:      "uptimerobot_maintenance_window.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
