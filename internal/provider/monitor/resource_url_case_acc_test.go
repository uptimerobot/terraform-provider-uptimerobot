//go:build acceptance

package monitor_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func TestAccMonitorResource_URLCase(t *testing.T) {
	for _, createUppercase := range []bool{false, true} {
		label := "update host case"
		if createUppercase {
			label = "create uppercase"
		}
		t.Run(label, func(t *testing.T) {
			name := provideracctest.RandomName("acc-url-case")
			lowerURL := provideracctest.UniqueURL(name)
			upperURL := strings.Replace(lowerURL, "https://example.com", "HTTPS://ExAmPlE.com", 1)
			createURL := lowerURL
			if createUppercase {
				createURL = upperURL
			}
			var createdID string
			check := func(url string) resource.TestCheckFunc {
				return resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", url),
					func(state *terraform.State) error {
						id := state.RootModule().Resources["uptimerobot_monitor.test"].Primary.ID
						if createdID == "" {
							createdID = id
						}
						monitorID, err := strconv.ParseInt(id, 10, 64)
						if err != nil {
							return err
						}
						monitor, err := provideracctest.APIClient().GetMonitor(t.Context(), monitorID)
						if err != nil {
							return err
						}
						if monitor.URL != lowerURL {
							return fmt.Errorf("API URL = %q, want %q", monitor.URL, lowerURL)
						}
						return resource.TestCheckResourceAttr("uptimerobot_monitor.test", "id", createdID)(state)
					},
				)
			}
			lookupConfig := testAccMonitorResourceConfigWithURL(name+"-updated", upperURL) + fmt.Sprintf(`
data "uptimerobot_monitor" "by_url" {
  url = %q
}
data "uptimerobot_monitors" "by_url" {
  url = %q
}
`, upperURL, upperURL)
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { provideracctest.PreCheck(t) },
				ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
				CheckDestroy:             provideracctest.CheckMonitorDestroy,
				Steps: []resource.TestStep{
					{Config: testAccMonitorResourceConfigWithURL(name, createURL), Check: check(createURL)},
					{Config: testAccMonitorResourceConfigWithURL(name, upperURL), Check: check(upperURL)},
					{Config: testAccMonitorResourceConfigWithURL(name+"-updated", upperURL), Check: check(upperURL)},
					{Config: testAccMonitorResourceConfigWithURL(name+"-updated", upperURL), PlanOnly: true},
					{
						PreConfig: func() {
							waitForURLCaseMonitorInList(t, createdID, lowerURL)
						},
						Config: lookupConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							check(upperURL),
							resource.TestCheckResourceAttr("data.uptimerobot_monitor.by_url", "url", upperURL),
							resource.TestCheckResourceAttrPair("data.uptimerobot_monitor.by_url", "id", "uptimerobot_monitor.test", "id"),
							resource.TestCheckResourceAttr("data.uptimerobot_monitors.by_url", "ids.#", "1"),
							resource.TestCheckResourceAttrPair("data.uptimerobot_monitors.by_url", "ids.0", "uptimerobot_monitor.test", "id"),
						),
					},
					{Config: lookupConfig, PlanOnly: true},
				},
			})
		})
	}
}

// URL-only lookups read the unfiltered list, which has a separate API cache from
// name-filtered lists. Wait for that exact list to expose the created monitor.
func waitForURLCaseMonitorInList(t *testing.T, id, url string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	api := provideracctest.APIClient()
	for {
		monitors, err := api.GetMonitors(ctx)
		if err == nil {
			for _, monitor := range monitors {
				if strconv.FormatInt(monitor.ID, 10) == id && monitor.URL == url {
					return
				}
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("monitor %s with URL %q was not visible in the unfiltered list: %v (last API error: %v)", id, url, ctx.Err(), err)
		case <-time.After(5 * time.Second):
		}
	}
}
