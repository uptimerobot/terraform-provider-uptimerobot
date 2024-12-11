package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccPSPResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name     = "test-monitor"
    url      = "https://example.com"
    type     = "http"
    interval = 300
}

resource "uptimerobot_psp" "test" {
    name     = %[1]q
    type     = "public"
    sort     = "name-asc"
    monitors = [uptimerobot_monitor.test.id]
}
`, name)
}

func TestAccPSPResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPSPResourceConfig("test-psp"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", "test-psp"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "type", "public"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "sort", "name-asc"),
				),
			},
			// Update testing
			{
				Config: testAccPSPResourceConfig("test-psp-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", "test-psp-updated"),
				),
			},
			// Import testing
			{
				ResourceName:      "uptimerobot_psp.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
