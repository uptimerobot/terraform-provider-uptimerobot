//go:build acceptance

package monitor_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func testAccMonitorWithThresholdMode(name, url, extra string) string {
	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  type     = "HTTP"
  url      = %q
  interval = 300
  timeout  = 30
%s
}
`, name, url, extra)
}

// A configuration that never mentions the mode must plan clean: the attribute is computed, so the mode the monitor
// already has is kept.
func TestAccMonitorResource_ThresholdModeOmittedPlansClean(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-threshold-mode-omitted")
	url := provideracctest.UniqueURL(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorWithThresholdMode(name, url, "  response_time_threshold = 1500"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "response_time_threshold", "1500"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "auto_thresholds.%", "0"),
				),
			},
			{
				Config:   testAccMonitorWithThresholdMode(name, url, "  response_time_threshold = 1500"),
				PlanOnly: true,
			},
		},
	})
}

func TestAccMonitorResource_ThresholdModeAutoRefusesValues(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-threshold-mode-conflict")
	url := provideracctest.UniqueURL(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorWithThresholdMode(name, url, "  response_time_threshold_mode = \"auto\"\n  response_time_threshold      = 1500"),
				ExpectError: regexp.MustCompile(
					`(?s)Response time threshold conflicts with.*response_time_threshold_mode`),
			},
		},
	})
}

// Needs an account where automatic thresholds are available, so it only runs when asked for.
func TestAccMonitorResource_ThresholdModeAuto(t *testing.T) {
	if os.Getenv("UPTIMEROBOT_TEST_AUTO_THRESHOLDS") == "" {
		t.Skip("set UPTIMEROBOT_TEST_AUTO_THRESHOLDS=1 on an account with automatic thresholds to run this test")
	}
	name := acctest.RandomWithPrefix("acc-threshold-mode-auto")
	url := provideracctest.UniqueURL(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorWithThresholdMode(name, url, "  response_time_threshold_mode = \"auto\""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "response_time_threshold_mode", "auto"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "auto_thresholds.%"),
				),
			},
			{
				// Thresholds the server derives must not show as drift.
				Config:   testAccMonitorWithThresholdMode(name, url, "  response_time_threshold_mode = \"auto\""),
				PlanOnly: true,
			},
			{
				Config: testAccMonitorWithThresholdMode(name, url, "  response_time_threshold_mode = \"manual\"\n  response_time_threshold      = 2000"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "response_time_threshold_mode", "manual"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "response_time_threshold", "2000"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "auto_thresholds.%", "0"),
				),
			},
		},
	})
}
