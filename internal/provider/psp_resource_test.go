package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccPSPResourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name = %q
  monitor_ids   = [12345, 67890]
}
`, name)
}

func TestAccPSPResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPSPDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPSPResourceConfig("test-psp"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", "test-psp"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitor_ids.#", "2"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitor_ids.0", "12345"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitor_ids.1", "67890"),
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
