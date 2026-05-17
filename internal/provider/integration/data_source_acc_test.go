//go:build acceptance

package integration_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func testAccIntegrationDataSourceConfig(name, value string) string {
	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_integration" "webhook" {
  name                     = %q
  type                     = "webhook"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = false

  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
  post_value               = jsonencode({ message = "data source test" })
}

data "uptimerobot_integration" "by_id" {
  id = uptimerobot_integration.webhook.id

  depends_on = [uptimerobot_integration.webhook]
}

data "uptimerobot_integration" "by_name_type" {
  name = uptimerobot_integration.webhook.name
  type = uptimerobot_integration.webhook.type

  depends_on = [uptimerobot_integration.webhook]
}
`, name, value)
}

func TestAccIntegrationDataSource(t *testing.T) {
	name := provideracctest.RandomName("tf-acc-integration-ds")
	value := fmt.Sprintf("https://httpbin.org/anything/%s", name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationDataSourceConfig(name, value),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_integration.by_id",
						"id",
						"uptimerobot_integration.webhook",
						"id",
					),
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_integration.by_name_type",
						"id",
						"uptimerobot_integration.webhook",
						"id",
					),
					resource.TestCheckResourceAttr("data.uptimerobot_integration.by_name_type", "type", "webhook"),
					resource.TestCheckResourceAttr("data.uptimerobot_integration.by_name_type", "enable_notifications_for", "1"),
					resource.TestCheckResourceAttr("data.uptimerobot_integration.by_name_type", "ssl_expiration_reminder", "false"),
				),
			},
		},
	})
}
