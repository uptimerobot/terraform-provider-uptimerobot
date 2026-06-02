//go:build acceptance

package psp_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func testAccPSPDataSourceResourceConfig(name string) string {
	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name        = %q
  monitor_ids = []

  custom_settings = {
    page = {
      layout  = "logo_on_left"
      theme   = "dark"
      density = "compact"
    }
    features = {
      show_bars = true
    }
  }
}
`, name)
}

func testAccPSPDataSourceConfig(name string) string {
	return testAccPSPDataSourceResourceConfig(name) + `
data "uptimerobot_psp" "by_id" {
  id = uptimerobot_psp.test.id

  depends_on = [uptimerobot_psp.test]
}

data "uptimerobot_psp" "by_name" {
  name = uptimerobot_psp.test.name

  depends_on = [uptimerobot_psp.test]
}
`
}

func testAccCheckPSPVisibleInList(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		apiClient := provideracctest.APIClient()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var lastListErr error
		for {
			statusPages, err := apiClient.ListAllPSPs(ctx)
			if err != nil {
				lastListErr = err
			} else {
				for _, statusPage := range statusPages {
					if statusPage.Name == name {
						return nil
					}
				}
			}

			select {
			case <-ctx.Done():
				if lastListErr != nil {
					return fmt.Errorf("PSP %q was not visible in list endpoint before ctx.Done; last apiClient.ListAllPSPs error: %v: %w", name, lastListErr, ctx.Err())
				}
				return fmt.Errorf("PSP %q was not visible in list endpoint before timeout: %w", name, ctx.Err())
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func TestAccPSPDataSource(t *testing.T) {
	name := provideracctest.RandomName("tf-acc-psp-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckPSPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPSPDataSourceResourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp.test", "name", name),
					testAccCheckPSPVisibleInList(name),
				),
			},
			{
				Config: testAccPSPDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_psp.by_id",
						"id",
						"uptimerobot_psp.test",
						"id",
					),
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_psp.by_name",
						"id",
						"uptimerobot_psp.test",
						"id",
					),
					resource.TestCheckResourceAttr("data.uptimerobot_psp.by_name", "name", name),
					resource.TestCheckResourceAttr("data.uptimerobot_psp.by_name", "status", "ENABLED"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_psp.by_name", "url_key"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp.by_name", "monitors_count", "0"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp.by_name", "monitor_ids.#", "0"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp.by_name", "custom_settings.page.layout", "logo_on_left"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp.by_name", "custom_settings.page.theme", "dark"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp.by_name", "custom_settings.page.density", "compact"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp.by_name", "custom_settings.features.show_bars", "true"),
				),
			},
		},
	})
}
