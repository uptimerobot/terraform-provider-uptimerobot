//go:build acceptance

package integration_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/maputil"
)

// Configs

func testAccWebhookIntegrationConfig(name, value string, enableNotificationsFor int64, sslExpirationReminder bool) string {
	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_integration" "webhook" {
  name                     = %q
  type                     = "webhook"
  value                    = %q
  enable_notifications_for = %d
  ssl_expiration_reminder  = %t

  // webhook send options
  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
  post_value               = "{\"message\": \"Alert: $monitorURL is $alertType\"}"
}
`, name, value, enableNotificationsFor, sslExpirationReminder)
}

func testAccWebhookIntegrationConfigWithCustomHeaders(name, value string, headers *map[string]string) string {
	customHeaders := ""
	if headers != nil {
		customHeaders = "\n  custom_headers = " + provideracctest.HCLStringMap(*headers)
	}

	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_integration" "webhook" {
  name                     = %q
  type                     = "webhook"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = true

  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
  post_value               = jsonencode({ message = "Alert: $monitorURL is $alertType" })%s
}
`, name, value, customHeaders)
}

func testAccCheckWebhookCustomHeaders(resourceName string, expected map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse integration ID %q: %w", rs.Primary.ID, err)
		}

		apiClient := provideracctest.APIClient()

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		backoff := 500 * time.Millisecond
		var lastErr error
		for {
			integration, err := apiClient.GetIntegration(ctx, id)
			if err != nil {
				lastErr = err
			} else if maputil.EqualStringMap(integration.CustomHeaders, expected) {
				return nil
			} else {
				lastErr = fmt.Errorf("expected custom_headers %#v, got %#v", expected, integration.CustomHeaders)
			}

			select {
			case <-ctx.Done():
				if lastErr != nil {
					return lastErr
				}
				return ctx.Err()
			case <-time.After(backoff):
			}

			if backoff < 5*time.Second {
				backoff *= 2
				if backoff > 5*time.Second {
					backoff = 5 * time.Second
				}
			}
		}
	}
}

// Acceptance tests

func TestAccIntegrationResource(t *testing.T) {
	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	name1 := "tfacc-webhook-" + suffix
	name2 := "tfacc-webhook-upd-" + suffix
	value := fmt.Sprintf("https://httpbin.org/anything?tfacc=%s", suffix)

	cfgCreate := testAccWebhookIntegrationConfig(name1, value, 2, false)
	cfgUpdate := testAccWebhookIntegrationConfig(name2, value, 3, true)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: cfgCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "name", name1),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "type", "webhook"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "value", value),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "enable_notifications_for", "2"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "ssl_expiration_reminder", "false"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_json", "true"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_query_string", "false"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "send_as_post_parameters", "false"),
				),
			},
			{
				// Update the name and integration settings to verify both paths settle.
				Config: cfgUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "name", name2),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "value", value),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "enable_notifications_for", "3"),
					resource.TestCheckResourceAttr("uptimerobot_integration.webhook", "ssl_expiration_reminder", "true"),
				),
			},
			{
				Config:             cfgUpdate,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
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
	name := provideracctest.RandomName("acc-webhook-json")
	resourceName := "uptimerobot_integration.test"
	value := fmt.Sprintf("https://example.com/hook?tfacc=%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	cfg1 := fmt.Sprintf(`
%s

resource "uptimerobot_integration" "test" {
  name                     = %q
  type                     = "webhook"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
  // canonical JSON (key order a,b)
  post_value               = jsonencode({ a = 1, b = "x" })
  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
}
`, provideracctest.ProviderConfig(), name, value)

	// Same logical JSON but with different key order/formatting; plan should be empty
	cfg2 := fmt.Sprintf(`
%s

resource "uptimerobot_integration" "test" {
  name                     = %q
  type                     = "webhook"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
  // key order b,a -> should be treated equivalent
  post_value               = "{\"b\":\"x\",\"a\":1}"
  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
}
`, provideracctest.ProviderConfig(), name, value)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { provideracctest.PreCheck(t) },
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
			{
				Config:             cfg2,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Integration_Webhook_CustomHeaders(t *testing.T) {
	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := "tfacc-webhook-headers-" + suffix
	value := fmt.Sprintf("https://httpbin.org/anything?tfacc_headers=%s", suffix)
	resourceName := "uptimerobot_integration.webhook"

	initialHeaders := map[string]string{
		"Authorization": "Bearer initial",
		"X-Source":      "terraform",
	}
	updatedHeaders := map[string]string{
		"Authorization": "Bearer updated",
		"X-Trace-ID":    suffix,
	}
	emptyHeaders := map[string]string{}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookIntegrationConfigWithCustomHeaders(name, value, &initialHeaders),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "custom_headers.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_headers.Authorization", "Bearer initial"),
					resource.TestCheckResourceAttr(resourceName, "custom_headers.X-Source", "terraform"),
					testAccCheckWebhookCustomHeaders(resourceName, initialHeaders),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccWebhookIntegrationConfigWithCustomHeaders(name, value, &updatedHeaders),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "custom_headers.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_headers.Authorization", "Bearer updated"),
					resource.TestCheckResourceAttr(resourceName, "custom_headers.X-Trace-ID", suffix),
					resource.TestCheckNoResourceAttr(resourceName, "custom_headers.X-Source"),
					testAccCheckWebhookCustomHeaders(resourceName, updatedHeaders),
				),
			},
			{
				Config: testAccWebhookIntegrationConfigWithCustomHeaders(name, value, &emptyHeaders),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "custom_headers.%", "0"),
					testAccCheckWebhookCustomHeaders(resourceName, emptyHeaders),
				),
			},
			{
				Config:             testAccWebhookIntegrationConfigWithCustomHeaders(name, value, &emptyHeaders),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Integration_Mattermost_CustomValue_Clear(t *testing.T) {
	mwURL, ok := provideracctest.OptionalEnv("UPTIMEROBOT_TEST_MATTERMOST_WEBHOOK_URL")
	if !ok {
		t.Skip("set UPTIMEROBOT_TEST_MATTERMOST_WEBHOOK_URL to run this test")
	}
	name := provideracctest.RandomName("acc-mattermost")
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
`, provideracctest.ProviderConfig(), name, mwURL)

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
`, provideracctest.ProviderConfig(), name, mwURL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { provideracctest.PreCheck(t) },
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
			{
				Config:             cfgClear,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Integration_PagerDuty_RegionAndAutoResolve(t *testing.T) {
	name := provideracctest.RandomName("acc-pagerduty")
	resourceName := "uptimerobot_integration.test"
	key1 := acctest.RandStringFromCharSet(32, acctest.CharSetAlphaNum)
	key2 := acctest.RandStringFromCharSet(32, acctest.CharSetAlphaNum)

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
`, provideracctest.ProviderConfig(), name, key1)

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
`, provideracctest.ProviderConfig(), name, key2)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { provideracctest.PreCheck(t) },
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
			{
				Config:             cfg2,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Integration_Pushover_Priority_RoundTrip(t *testing.T) {
	userKey, ok := provideracctest.OptionalEnv("UPTIMEROBOT_TEST_PUSHOVER_USER_KEY")
	if !ok {
		t.Skip("set UPTIMEROBOT_TEST_PUSHOVER_USER_KEY to run this test")
	}
	name := provideracctest.RandomName("acc-pushover")
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
`, provideracctest.ProviderConfig(), name, userKey)

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
`, provideracctest.ProviderConfig(), name, userKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { provideracctest.PreCheck(t) },
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
			{
				Config:             cfg2,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Integration_Webhook_DuplicateConflict(t *testing.T) {
	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	namePrimary := provideracctest.RandomName("acc-webhook-dup-primary")
	nameDuplicate := provideracctest.RandomName("acc-webhook-dup-duplicate")
	value := fmt.Sprintf("https://httpbin.org/anything?dup=%s", suffix)

	cfgSingle := fmt.Sprintf(`
%s
resource "uptimerobot_integration" "primary" {
  name                     = %q
  type                     = "webhook"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
}
`, provideracctest.ProviderConfig(), namePrimary, value)

	cfgDuplicate := fmt.Sprintf(`
%s
resource "uptimerobot_integration" "primary" {
  name                     = %q
  type                     = "webhook"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
}

resource "uptimerobot_integration" "duplicate" {
  name                     = %q
  type                     = "webhook"
  value                    = %q
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
  send_as_json             = true
  send_as_query_string     = false
  send_as_post_parameters  = false
}
`, provideracctest.ProviderConfig(), namePrimary, value, nameDuplicate, value)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		CheckDestroy:             provideracctest.CheckIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: cfgSingle,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_integration.primary", "name", namePrimary),
					resource.TestCheckResourceAttr("uptimerobot_integration.primary", "type", "webhook"),
				),
			},
			{
				Config:      cfgDuplicate,
				ExpectError: regexp.MustCompile(`(?i)integration already exists|already exists|status code 409`),
			},
			{
				Config:             cfgSingle,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
