package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Configs

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

func testAccIntegrationPreCheck(t *testing.T) {
	if v := os.Getenv("UPTIMEROBOT_API_KEY"); v == "" {
		t.Skip("UPTIMEROBOT_API_KEY must be set to run integration acceptance tests")
	}
}

func testAccIntegrationProviderConfig() string {
	return fmt.Sprintf(`
provider "uptimerobot" {
  api_key = "%s"
}
`, os.Getenv("UPTIMEROBOT_API_KEY"))
}

// Helpers

func randomName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()%1e9)
}

func testAccOptionalEnv(key string) (string, bool) {
	v := os.Getenv(key)
	return v, v != ""
}

// Acceptance tests

func TestAccIntegrationResource(t *testing.T) {
	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	name1 := "tfacc-webhook-" + suffix
	name2 := "tfacc-webhook-upd-" + suffix
	value := fmt.Sprintf("https://httpbin.org/anything?tfacc=%s", suffix)

	cfgCreate := testAccWebhookIntegrationConfig(name1, value)
	cfgUpdate := testAccWebhookIntegrationConfig(name2, value)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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

func TestAcc_Integration_Webhook_JSONPlanModifier_RoundTrip(t *testing.T) {
	name := randomName("acc-webhook-json")
	resourceName := "uptimerobot_integration.test"

	cfg1 := fmt.Sprintf(`
%s

resource "uptimerobot_integration" "test" {
  name                     = %q
  type                     = "webhook"
  value                    = "https://example.com/hook"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
  // canonical JSON (key order a,b)
  post_value               = jsonencode({ a = 1, b = "x" })
  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
}
`, testAccIntegrationProviderConfig(), name)

	// Same logical JSON but with different key order/formatting; plan should be empty
	cfg2 := fmt.Sprintf(`
%s

resource "uptimerobot_integration" "test" {
  name                     = %q
  type                     = "webhook"
  value                    = "https://example.com/hook"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
  // key order b,a -> should be treated equivalent
  post_value               = "{\"b\":\"x\",\"a\":1}"
  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
}
`, testAccIntegrationProviderConfig(), name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccIntegrationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: cfg1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "webhook"),
					resource.TestCheckResourceAttr(resourceName, "post_value", `{"a":1,"b":"x"}`),
				),
			},
			{
				Config: cfg2,
				// If the JSON plan modifier works, there should be no changes
				// (i.e., implicit "expect empty plan")
			},
		},
	})
}

func TestAcc_Integration_Mattermost_CustomValue_Clear(t *testing.T) {
	mwURL, ok := testAccOptionalEnv("UPTIMEROBOT_TEST_MATTERMOST_WEBHOOK_URL")
	if !ok {
		t.Skip("set UPTIMEROBOT_TEST_MATTERMOST_WEBHOOK_URL to run this test")
	}
	name := randomName("acc-mattermost")
	resourceName := "uptimerobot_integration.test"

	cfgCreate := fmt.Sprintf(`
%s
resource "uptimerobot_integration" "test" {
  name                     = %q
  type                     = "mattermost"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = false
  custom_value             = "initial-note"
}
`, testAccIntegrationProviderConfig(), name, mwURL)

	cfgClear := fmt.Sprintf(`
%s
resource "uptimerobot_integration" "test" {
  name                     = %q
  type                     = "mattermost"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = false
  custom_value             = ""
}
`, testAccIntegrationProviderConfig(), name, mwURL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccIntegrationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: cfgCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "mattermost"),
					resource.TestCheckResourceAttr(resourceName, "custom_value", "initial-note"),
				),
			},
			{
				Config: cfgClear,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "custom_value", ""),
				),
			},
		},
	})
}

func TestAcc_Integration_PagerDuty_RegionAndAutoResolve(t *testing.T) {
	name := randomName("acc-pagerduty")
	resourceName := "uptimerobot_integration.test"
	key1 := "01234567890123456789012345678901" // 32 chars
	key2 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	cfg1 := fmt.Sprintf(`
%s
resource "uptimerobot_integration" "test" {
  name                     = %q
  type                     = "pagerduty"
  value                    = %q
  enable_notifications_for = 1   # match API default to avoid drift
  ssl_expiration_reminder  = true
  location                 = "us"
  auto_resolve             = true
}
`, testAccIntegrationProviderConfig(), name, key1)

	cfg2 := fmt.Sprintf(`
%s
resource "uptimerobot_integration" "test" {
  name                     = %q
  type                     = "pagerduty"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = false
  location                 = "eu"
  auto_resolve             = false
}
`, testAccIntegrationProviderConfig(), name, key2)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccIntegrationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: cfg1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "pagerduty"),
					resource.TestCheckResourceAttr(resourceName, "location", "us"),
					resource.TestCheckResourceAttr(resourceName, "auto_resolve", "true"),
				),
			},
			{
				Config: cfg2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "location", "eu"),
					resource.TestCheckResourceAttr(resourceName, "auto_resolve", "false"),
				),
			},
		},
	})
}

func TestAcc_Integration_Pushover_Priority_RoundTrip(t *testing.T) {
	userKey, ok := testAccOptionalEnv("UPTIMEROBOT_TEST_PUSHOVER_USER_KEY")
	if !ok {
		t.Skip("set UPTIMEROBOT_TEST_PUSHOVER_USER_KEY to run this test")
	}
	name := randomName("acc-pushover")
	resourceName := "uptimerobot_integration.test"

	cfg1 := fmt.Sprintf(`
%s
resource "uptimerobot_integration" "test" {
  name                     = %q
  type                     = "pushover"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = false
  priority                 = "High"
}
`, testAccIntegrationProviderConfig(), name, userKey)

	cfg2 := fmt.Sprintf(`
%s
resource "uptimerobot_integration" "test" {
  name                     = %q
  type                     = "pushover"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = false
  priority                 = "Normal"
}
`, testAccIntegrationProviderConfig(), name, userKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccIntegrationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: cfg1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "pushover"),
					resource.TestCheckResourceAttr(resourceName, "priority", "High"),
				),
			},
			{
				Config: cfg2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "priority", "Normal"),
				),
			},
		},
	})
}
