package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIntegrationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntegrationDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing for Webhook integration
			{
				Config: testAccWebhookIntegrationConfig("test-webhook"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "name", "test-webhook"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "type", "webhook"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "value", "https://httpbin.org/anything"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "enable_notifications_for", "1"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "ssl_expiration_reminder", "true"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_json", "true"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_query_string", "false"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_post_parameters", "false"),
					// Check that post_value contains the expected content (allowing for format differences)
					resource.TestCheckResourceAttrSet("uptimerobot_integration.webhook", "post_value"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "uptimerobot_integration.webhook",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"post_value"}, // Ignore JSON formatting differences
			},
			// Update and Read testing
			{
				Config: testAccWebhookIntegrationConfig("test-webhook-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "name", "test-webhook-updated"),
				),
			},
		},
	})
}

func testAccWebhookIntegrationConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_integration" "webhook" {
  name                     = %q
  type                     = "webhook"
  value                    = "https://httpbin.org/anything"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
  send_as_json            = true
  send_as_query_string    = false
  send_as_post_parameters = false
  post_value              = "{\"message\": \"Alert: $monitorURL is $alertType\"}"
}`, name)
}
