package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMaintenanceWindowResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMaintenanceWindowResourceConfig("test-mw"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "name", "test-mw"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "type", "weekly"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "duration", "60"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "repeat", "weekly"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "week_day", "0"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "description", "Test maintenance window"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "tags.0", "test"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "tags.1", "maintenance"),
				),
			},
			// Update testing
			{
				Config: testAccMaintenanceWindowResourceConfigUpdated("test-mw-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "name", "test-mw-updated"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "duration", "120"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "description", "Updated test maintenance window"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "tags.0", "test-updated"),
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

func testAccMaintenanceWindowResourceConfig(name string) string {
	// Using Unix timestamp for 2024-12-11 11:38:23 +0100 as base
	startTime := "1702290000"

	return fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  friendly_name = "Test Monitor"
  type         = "http"
  url          = "https://example.com"
  interval     = 300
}

resource "uptimerobot_maintenance_window" "test" {
  name        = %[1]q
  type        = "weekly"
  start_time  = %[2]s
  duration    = 60
  monitors    = [uptimerobot_monitor.test.id]
  
  repeat      = "weekly"
  week_day    = 0
  
  description = "Test maintenance window"
  tags        = ["test", "maintenance"]
}
`, name, startTime)
}

func testAccMaintenanceWindowResourceConfigUpdated(name string) string {
	// Using Unix timestamp for 2024-12-11 11:38:23 +0100 as base
	startTime := "1702290000"

	return fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  friendly_name = "Test Monitor"
  type         = "http"
  url          = "https://example.com"
  interval     = 300
}

resource "uptimerobot_maintenance_window" "test" {
  name        = %[1]q
  type        = "weekly"
  start_time  = %[2]s
  duration    = 120
  monitors    = [uptimerobot_monitor.test.id]
  
  repeat      = "weekly"
  week_day    = 0
  
  description = "Updated test maintenance window"
  tags        = ["test-updated"]
}
`, name, startTime)
}
