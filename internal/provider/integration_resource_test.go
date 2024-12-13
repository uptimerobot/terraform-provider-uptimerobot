package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIntegrationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing for Slack integration
			{
				Config: testAccSlackIntegrationConfig("test-slack"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_integration.slack", "friendly_name", "test-slack"),
					resource.TestCheckResourceAttr("uptimerobot_integration.slack", "type", "slack"),
					resource.TestCheckResourceAttr("uptimerobot_integration.slack", "value", "https://hooks.slack.com/services/XXXXX/YYYYY/ZZZZZ"),
					resource.TestCheckResourceAttr("uptimerobot_integration.slack", "custom_value", "#monitoring"),
					resource.TestCheckResourceAttr("uptimerobot_integration.slack", "enable_notifications_for", "1"),
					resource.TestCheckResourceAttr("uptimerobot_integration.slack", "ssl_expiration_reminder", "true"),
				),
			},
			// Create and Read testing for Webhook integration
			{
				Config: testAccWebhookIntegrationConfig("test-webhook"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "friendly_name", "test-webhook"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "type", "webhook"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "value", "https://example.com/webhook"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "custom_value", "POST"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "enable_notifications_for", "1"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "ssl_expiration_reminder", "true"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_json", "true"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_query_string", "false"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "post_value", "{\"message\": \"Alert: $monitorURL is $alertType\"}"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "uptimerobot_integration.webhook",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccWebhookIntegrationConfig("test-webhook-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "friendly_name", "test-webhook-updated"),
				),
			},
		},
	})
}

func testAccSlackIntegrationConfig(name string) string {
	return fmt.Sprintf(`
resource "uptimerobot_integration" "slack" {
  friendly_name             = %[1]q
  type                     = "slack"
  value                    = "https://hooks.slack.com/services/XXXXX/YYYYY/ZZZZZ"
  custom_value             = "#monitoring"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}`, name)
}

func testAccWebhookIntegrationConfig(name string) string {
	return fmt.Sprintf(`
resource "uptimerobot_integration" "webhook" {
  friendly_name             = %[1]q
  type                     = "webhook"
  value                    = "https://example.com/webhook"
  custom_value             = "POST"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
  send_as_json            = true
  send_as_query_string    = false
  post_value              = "{\"message\": \"Alert: $monitorURL is $alertType\"}"
}`, name)
}
