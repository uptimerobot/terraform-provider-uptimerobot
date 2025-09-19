package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIntegrationResource(t *testing.T) {
	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	name1 := "tfacc-webhook-" + suffix
	name2 := "tfacc-webhook-upd-" + suffix
	value := fmt.Sprintf("https://httpbin.org/anything?tfacc=%s", suffix)

	cfgCreate := testAccWebhookIntegrationConfig(name1, value)
	cfgUpdate := testAccWebhookIntegrationConfig(name2, value)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: cfgCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "name", name1),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "type", "webhook"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "value", value),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "enable_notifications_for", "1"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "ssl_expiration_reminder", "true"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_json", "true"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_query_string", "false"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_post_parameters", "false"),
				),
			},
			{
				// update just the name to verify Update works
				Config: cfgUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "name", name2),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "value", value),
				),
			},
			{
				ResourceName:            "uptimerobot_integration.webhook",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"name"}, // API may returned same value as before update. It is asserted and being checked.
			},
		},
	})
}

func testAccWebhookIntegrationConfig(name, value string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_integration" "webhook" {
  name                     = %q
  type                     = "webhook"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = true

  // webhook send options
  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
  post_value               = "{\"message\": \"Alert: $monitorURL is $alertType\"}"
}
`, name, value)
}
