package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Basic config with features + a few custom settings to cover both bool and string fields.
func testAccPSPResourceConfigWithFeatures(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name         = %q
  monitor_ids  = [12345, 67890]

  custom_settings = {
    page = {
      layout  = "logo_on_left"
      theme   = "dark"
      density = "compact"
    }
    colors = {
      main = "#112233"
      text = "#334455"
      link = "#556677"
    }
    features = {
      show_bars              = true
      show_monitor_url       = false
      enable_details_page    = true
      hide_paused_monitors   = true
      // leave the rest unset to ensure we don't send nulls
    }
  }
}
`, name)
}

// Updated config: flip a couple of feature flags and tweak a string field.
func testAccPSPResourceConfigWithFeaturesUpdated(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name         = %q
  monitor_ids  = [12345, 67890]

  custom_settings = {
    page = {
      layout  = "logo_on_left"
      theme   = "light"   // changed
      density = "compact"
    }
    colors = {
      main = "#113312ff"
      text = "#334455"
      link = "#778899"    // changed
    }
    features = {
      show_bars              = false    // flipped
      show_monitor_url       = true     // flipped
      enable_details_page    = true
      hide_paused_monitors   = true
    }
  }
}
`, name)
}

func TestAccPSPResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPSPDestroy,
		Steps: []resource.TestStep{
			// Create + Read
			{
				Config: testAccPSPResourceConfigWithFeatures("test-psp"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// top-level
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", "test-psp"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitor_ids.#", "2"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitor_ids.0", "12345"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitor_ids.1", "67890"),
					// nested: page
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.page.layout", "logo_on_left"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.page.theme", "dark"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.page.density", "compact"),
					// nested: colors
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.colors.main", "#112233"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.colors.text", "#334455"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.colors.link", "#556677"),
					// nested: features (booleans)
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.features.show_bars", "true"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.features.show_monitor_url", "false"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.features.enable_details_page", "true"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.features.hide_paused_monitors", "true"),
				),
			},
			// Update flags, strings
			{
				Config: testAccPSPResourceConfigWithFeaturesUpdated("test-psp-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", "test-psp-updated"),
					// updated page + colors
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.page.theme", "light"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.colors.link", "#778899"),
					// flipped features
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.features.show_bars", "false"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.features.show_monitor_url", "true"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.features.enable_details_page", "true"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.features.hide_paused_monitors", "true"),
				),
			},
			// Import testing
			{
				ResourceName:            "uptimerobot_psp.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"monitor_ids"},
			},
		},
	})
}
