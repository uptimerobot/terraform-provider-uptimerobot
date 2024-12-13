package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccMonitorResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    friendly_name = %[1]q
    url          = "https://example.com"
    type         = 1
    interval     = 300
}
`, name)
}

func TestAccMonitorResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMonitorResourceConfig("test-monitor"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "friendly_name", "test-monitor"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "1"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", "https://example.com"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "interval", "300"),
				),
			},
			// Update testing
			{
				Config: testAccMonitorResourceConfig("test-monitor-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "friendly_name", "test-monitor-updated"),
				),
			},
			// Import testing
			{
				ResourceName:      "uptimerobot_monitor.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
