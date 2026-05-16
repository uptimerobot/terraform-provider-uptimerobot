//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccIntegrationDataSourceConfig(name, value string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
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
	name := randomName("tf-acc-integration-ds")
	value := fmt.Sprintf("https://httpbin.org/anything/%s", name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntegrationDestroy,
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
