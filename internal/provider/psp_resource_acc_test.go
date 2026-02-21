//go:build acceptance

package provider

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func writeAccPSPImageFile(t *testing.T, name string, fill color.RGBA) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name+".png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test image file: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, fill)
		}
	}
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		t.Fatalf("encode test image: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close test image file: %v", err)
	}

	return path
}

// Basic config with features + a few custom settings to cover both bool and string fields.
func testAccPSPResourceConfigWithFeatures(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
	resource "uptimerobot_psp" "test" {
	  name         = %q

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

func testAccPSPResourceConfigWithoutMonitors(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name = %q

  // no monitor_ids
  monitor_ids = []
  
  custom_settings = {
    page = { layout = "logo_on_left", theme = "dark", density = "compact" }
  }
}
`, name)
}

func testAccPSPResourceConfigWithMonitor(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "psp" {
  name     = %q
  type     = "HTTP"
  url      = "https://example.com/psp-%s"
  interval = 300
}

resource "uptimerobot_psp" "test" {
  name = %q

  monitor_ids = [uptimerobot_monitor.psp.id]
}
`, name, name, name)
}

func testAccPSPResourceConfigCustomSettingsOmitDefaults(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name       = %q
  monitor_ids = []

  custom_settings = {
    page = {
      density = "compact"
      # layout/theme intentionally omitted to check consistency logic
    }
    colors = {
      # main intentionally omitted
      text = "#F9FAFB"
      link = "#60A5FA"
    }
    features = {
      show_bars = true
      # show_outage_updates intentionally omitted
    }
  }
}
`, name)
}

func testAccPSPResourceConfigCustomSettingsEmpty(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name        = %q
  monitor_ids = []

  custom_settings = {}
}
`, name)
}

func testAccPSPResourceConfigOptionalsSet(name, logoPath, iconPath string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name = %q

  logo_file_path = %q
  icon_file_path = %q
  ga_code        = "G-ABCDE12349"

  # Sensitive and not returned by the API. Provider should still not error.
  password = "change-me"
}
`, name, logoPath, iconPath)
}

func testAccPSPResourceConfigOptionalsOmitted(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name = %q

  # ga_code/password intentionally omitted for checking stability
}
`, name)
}

func testAccPSPResourceConfigPinnedAnnouncementID(name string, pinnedID int64) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name = %q

  pinned_announcement_id = %d
}
`, name, pinnedID)
}

func testAccPSPResourceConfigPinnedAnnouncementIDOmitted(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name = %q
}
`, name)
}

func TestAccPSPResource(t *testing.T) {
	nameCreate := randomName("test-psp")
	nameUpdate := randomName("test-psp-updated")
	nameNoMonitors := randomName("test-psp-nomon")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPSPDestroy,
		Steps: []resource.TestStep{
			// Create + Read
			{
				Config: testAccPSPResourceConfigWithFeatures(nameCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					// top-level
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", nameCreate),
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
				Config: testAccPSPResourceConfigWithFeaturesUpdated(nameUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", nameUpdate),
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
			{
				Config: testAccPSPResourceConfigWithoutMonitors(nameNoMonitors),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", nameNoMonitors),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitor_ids.#", "0"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitors_count", "0"),
				),
			},
			// Import testing
			{
				ResourceName:            "uptimerobot_psp.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"monitor_ids", "name", "custom_settings"},
			},
		},
	})
}

func TestAccPSPResource_MonitorCountFollowsMonitorIDs(t *testing.T) {
	name := randomName("test-psp-monitors")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPSPDestroy,
		Steps: []resource.TestStep{
			// Step 1: PSP with one monitor
			{
				Config: testAccPSPResourceConfigWithMonitor(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// one monitor in the set
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitor_ids.#", "1"),
					// count should match
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitors_count", "1"),
				),
			},
			// Step 2: same PSP, no monitors
			{
				Config: testAccPSPResourceConfigWithoutMonitors(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitor_ids.#", "0"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "monitors_count", "0"),
				),
			},
		},
	})
}

func TestAccPSPResource_CustomSettings_OmittedDefaultsNotPersisted(t *testing.T) {
	name := randomName("acc-psp-omit-defaults")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPSPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPSPResourceConfigCustomSettingsOmitDefaults(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", name),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.page.density", "compact"),
					resource.TestCheckNoResourceAttr("uptimerobot_psp.test", "custom_settings.page.layout"),
					resource.TestCheckNoResourceAttr("uptimerobot_psp.test", "custom_settings.page.theme"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.colors.text", "#F9FAFB"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.colors.link", "#60A5FA"),
					resource.TestCheckNoResourceAttr("uptimerobot_psp.test", "custom_settings.colors.main"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "custom_settings.features.show_bars", "true"),
					resource.TestCheckNoResourceAttr("uptimerobot_psp.test", "custom_settings.features.show_outage_updates"),
					resource.TestCheckNoResourceAttr("uptimerobot_psp.test", "custom_settings.font.family"),
				),
			},
		},
	})
}

func TestAccPSPResource_CustomSettings_EmptyObjectStable(t *testing.T) {
	name := randomName("acc-psp-empty-settings")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPSPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPSPResourceConfigCustomSettingsEmpty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", name),
					resource.TestCheckNoResourceAttr("uptimerobot_psp.test", "custom_settings.page.layout"),
					resource.TestCheckNoResourceAttr("uptimerobot_psp.test", "custom_settings.page.theme"),
					resource.TestCheckNoResourceAttr("uptimerobot_psp.test", "custom_settings.page.density"),
					resource.TestCheckNoResourceAttr("uptimerobot_psp.test", "custom_settings.colors.main"),
					resource.TestCheckNoResourceAttr("uptimerobot_psp.test", "custom_settings.features.show_bars"),
				),
			},
		},
	})
}

func TestAccPSPResource_OmitOptionalTopLevelFields_DoesNotError(t *testing.T) {
	name := randomName("acc-psp-opt")
	logoPath := writeAccPSPImageFile(t, "logo", color.RGBA{R: 20, G: 120, B: 220, A: 255})
	iconPath := writeAccPSPImageFile(t, "icon", color.RGBA{R: 220, G: 120, B: 20, A: 255})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPSPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPSPResourceConfigOptionalsSet(name, logoPath, iconPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", name),
					resource.TestCheckResourceAttrSet("uptimerobot_psp.test", "icon"),
					resource.TestCheckResourceAttrSet("uptimerobot_psp.test", "logo"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "icon_file_path", iconPath),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "logo_file_path", logoPath),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "ga_code", "G-ABCDE12349"),
				),
			},
			{
				Config: testAccPSPResourceConfigOptionalsOmitted(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", name),
					resource.TestCheckResourceAttrSet("uptimerobot_psp.test", "icon"),
					resource.TestCheckResourceAttrSet("uptimerobot_psp.test", "logo"),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "icon_file_path", iconPath),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "logo_file_path", logoPath),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "ga_code", "G-ABCDE12349"),
				),
			},
		},
	})
}

func TestAccPSPResource_PinnedAnnouncementID_OmitDoesNotClear(t *testing.T) {
	idStr, ok := testAccOptionalEnv("UPTIMEROBOT_TEST_PINNED_ANNOUNCEMENT_ID")
	if !ok {
		t.Skip("Set UPTIMEROBOT_TEST_PINNED_ANNOUNCEMENT_ID to run pinned_announcement_id acceptance")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		t.Fatalf("invalid UPTIMEROBOT_TEST_PINNED_ANNOUNCEMENT_ID %q: %v", idStr, err)
	}

	name := randomName("acc-psp-pinned")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPSPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPSPResourceConfigPinnedAnnouncementID(name, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", name),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "pinned_announcement_id", idStr),
				),
			},
			{
				Config: testAccPSPResourceConfigPinnedAnnouncementIDOmitted(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", name),
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "pinned_announcement_id", idStr),
				),
			},
		},
	})
}
