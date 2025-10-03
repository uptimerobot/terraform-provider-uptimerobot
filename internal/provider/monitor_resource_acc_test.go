package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

/*
	Config addition rules:

	- Config helpers for the common, repeatable configs like HTTP base monitor, headers, tags, MWs, etc.
	They reduces duplication and makes refactors, such as adding a timeout = 30 to HTTP easy to be performed.

	- Inline configs only when the test’s readability depends on seeing the exact HCL schema in the test.
	For example, negative cases that assert a specific validation error, or tiny one-off / one time scenarios.
*/

// Config helpers for tests --------------------------------------------------

func testAccMonitorResourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = "https://example.com"
    type         = "HTTP"
    interval     = 300
	timeout   	 = 30
}
`, name)
}

func testAccMonitorResourceConfigWithAlertContacts(name string, alertContacts []string) string {
	alertContactsStr := ""
	if len(alertContacts) > 0 {
		alertContactsStr = fmt.Sprintf(`
    assigned_alert_contacts = [%s]`, joinQuotedStrings(alertContacts))
	}

	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = "https://example.com"
    type         = "HTTP"
    interval     = 300%s
	timeout  	 = 30
}
`, name, alertContactsStr)
}

func testAccMonitorResourceConfigWithTags(name string, tags []string) string {
	tagsStr := ""
	if len(tags) > 0 {
		tagsStr = fmt.Sprintf(`
    tags = [%s]`, joinQuotedStrings(tags))
	}

	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = "https://example.com"
    type         = "HTTP"
    interval     = 300%s
	timeout      = 30
}
`, name, tagsStr)
}

func testAccMonitorResourceConfigWithMaintenanceWindows(name string, maintenanceWindowIDs []int) string {
	maintenanceWindowsStr := ""
	if len(maintenanceWindowIDs) > 0 {
		maintenanceWindowsStr = fmt.Sprintf(`
    maintenance_window_ids = [%s]`, joinInts(maintenanceWindowIDs))
	}

	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = "https://example.com"
    type         = "HTTP"
    interval     = 300%s
	timeout      = 30
}
`, name, maintenanceWindowsStr)
}

// nolint:unparam // kept for symmetry with other helpers & future reuse
func testAccMonitorResourceConfigWithSuccessHTTPResponseCodes(name string, responseCodes []string) string {
	var responseCodesStr string
	if responseCodes != nil {
		if len(responseCodes) == 0 {
			responseCodesStr = `
    success_http_response_codes = []`
		} else {
			responseCodesStr = fmt.Sprintf(`
    success_http_response_codes = [%s]`, joinQuotedStrings(responseCodes))
		}
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = "https://example.com"
    type         = "HTTP"
    interval     = 300%s
    timeout      = 30
}
`, name, responseCodesStr)
}

func testAccMonitorResourceConfigWithHeaders(name string, headers map[string]string) string {
	hdr := ""
	if headers != nil {
		// build a literal headers map
		hdr = "\n  custom_http_headers = {"
		first := true
		for k, v := range headers {
			if first {
				first = false
			} else {
				hdr += ","
			}
			hdr += fmt.Sprintf(` %q = %q`, k, v)
		}
		hdr += " }"
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = "https://example.com"
  type     = "HTTP"
  interval = 300%s
  timeout  = 30
}
`, name, hdr)
}

// headers block explicitly empty as {}.
func testAccMonitorResourceConfigWithEmptyHeaders(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = "https://example.com"
  type     = "HTTP"
  interval = 300
  timeout  = 30
  custom_http_headers = {}
}
`, name)
}

func testAccMonitorResourceConfigWithBody(name string, body string) string {
	// body should be an HCL expression, e.g. `jsonencode({foo="bar", n=1})` or `null`
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com/echo"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  post_value_data  = %s
}
`, name, body)
}

func testAccMonitorResourceConfigWithKV(name string, kv map[string]string) string {
	body := ""
	if kv != nil {
		body = "\n  post_value_kv = {"
		first := true
		for k, v := range kv {
			if first {
				first = false
			} else {
				body += ","
			}
			body += fmt.Sprintf(` %q = %q`, k, v)
		}
		body += " }"
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com/echo"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  custom_http_headers = { "Content-Type" = "application/x-www-form-urlencoded" }%s
}
`, name, body)
}

func testAccMonitorResourceConfigPostNoBody(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com/echo"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  // no post_value_data / post_value_kv on purpose
}
`, name)
}

func testAccMonitorResourceConfigGetNoBody(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "GET"
}
`, name)
}

func testAccMonitorResourceConfigWithAlertContactObjects(name string, ids []string) string {
	ac := ""
	if len(ids) > 0 {
		ac = "\n  assigned_alert_contacts = ["
		for i, id := range ids {
			if i > 0 {
				ac += ","
			}
			ac += fmt.Sprintf(`{ alert_contact_id = %q }`, id)
		}
		ac += "]"
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = "https://example.com"
  type     = "HTTP"
  interval = 300%s
  timeout  = 30
}
`, name, ac)
}

// Helpers ------------------------------------------------------

func joinQuotedStrings(strings []string) string {
	var result string
	for i, s := range strings {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf(`"%s"`, s)
	}
	return result
}

func joinInts(ints []int) string {
	var result string
	for i, val := range ints {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf(`%d`, val)
	}
	return result
}

func mustAlertContactID(t *testing.T) string {
	t.Helper()
	id := os.Getenv("UPTIMEROBOT_TEST_ALERT_CONTACT_ID")
	if id == "" {
		t.Skip("Set UPTIMEROBOT_TEST_ALERT_CONTACT_ID to run alert-contacts acceptance")
	}
	return id
}

// Acceptance tests ------------------------------------------------

func TestAccMonitorResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMonitorResourceConfig("test-monitor"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", "https://example.com"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "interval", "300"),
				),
			},
			// Update testing
			{
				Config: testAccMonitorResourceConfig("test-monitor-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-updated"),
				),
			},
			// Import testing
			{
				ResourceName:            "uptimerobot_monitor.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeout", "status"},
			},
		},
	})
}
func TestAccMonitorResource_AlertContacts(t *testing.T) {
	id := mustAlertContactID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Step 1: create with no contacts
			{
				Config: testAccMonitorResourceConfigWithAlertContactObjects("test-monitor-alerts", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-alerts"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts"),
				),
			},
			// Step 2: add one contact
			{
				Config: testAccMonitorResourceConfigWithAlertContactObjects("test-monitor-alerts", []string{id}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(
						"uptimerobot_monitor.test",
						"assigned_alert_contacts.*",
						map[string]string{
							"alert_contact_id": id,
							"threshold":        "0",
							"recurrence":       "0",
						},
					),
				),
			},
			// Step 3: remove contacts (attribute omitted)
			{
				Config: testAccMonitorResourceConfigWithAlertContactObjects("test-monitor-alerts", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts"),
				),
			},
		},
	})
}

// TestAccMonitorResource_Tags tests the specific case where tags
// are added to an existing monitor that was initially created without any.
func TestAccMonitorResource_Tags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create monitor without tags
			{
				Config: testAccMonitorResourceConfigWithTags("test-monitor-tags", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-tags"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", "https://example.com"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "interval", "300"),
					// Verify no tags are set initially
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "tags.#", "0"),
				),
			},
			// Step 2: Add tags to existing monitor - this should NOT fail
			{
				Config: testAccMonitorResourceConfigWithTags("test-monitor-tags", []string{"production", "web"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-tags"),
					// Verify tags are now set
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "tags.0", "production"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "tags.1", "web"),
				),
			},
			// Step 3: Remove tags - set back to empty
			{
				Config: testAccMonitorResourceConfigWithTags("test-monitor-tags", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-tags"),
					// Verify tags are removed
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccMonitorResource_CustomHTTPHeaders(t *testing.T) {
	name := "test-monitor-headers"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// 1) Create without headers
			{
				Config: testAccMonitorResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "custom_http_headers"),
				),
			},
			// 2) Add headers
			{
				Config: testAccMonitorResourceConfigWithHeaders(name, map[string]string{"cat": "meow"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.%", "1"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.cat", "meow"),
				),
			},
			// 3) Change headers to ensures that update path works
			{
				Config: testAccMonitorResourceConfigWithHeaders(name, map[string]string{"foo": "bar"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.%", "1"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "custom_http_headers.cat"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.foo", "bar"),
				),
			},
			// 4) Clear by sending {}
			{
				Config: testAccMonitorResourceConfigWithEmptyHeaders(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.%", "0"),
				),
			},
			// 5) Clear by removing the block entirely
			{
				Config: testAccMonitorResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "custom_http_headers"),
				),
			},
			// 6) Import to ensure state matches API (no headers)
			{
				ResourceName:            "uptimerobot_monitor.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeout", "status"},
			},
		},
	})
}

// TestAccMonitorResource_MaintenanceWindows tests the specific case where maintenance window IDs
// are added to an existing monitor that was initially created without any.
func TestAccMonitorResource_MaintenanceWindows(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create monitor without maintenance windows
			{
				Config: testAccMonitorResourceConfigWithMaintenanceWindows("test-monitor-maintenance", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-maintenance"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", "https://example.com"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "interval", "300"),
					// Verify no maintenance windows are set initially
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids"),
				),
			},
			// Step 2: Add maintenance windows to existing monitor - this should NOT fail
			{
				Config: testAccMonitorResourceConfigWithMaintenanceWindows("test-monitor-maintenance", []int{12345, 67890}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-maintenance"),
					// Verify maintenance windows are now set
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids.#", "2"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids.0", "12345"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids.1", "67890"),
				),
			},
			// Step 3: Remove maintenance windows - set back to empty
			{
				Config: testAccMonitorResourceConfigWithMaintenanceWindows("test-monitor-maintenance", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-maintenance"),
					// Verify maintenance windows are removed
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids"),
				),
			},
		},
	})
}

// TestAccMonitorResource_SuccessHTTPResponseCodes tests the specific case where success HTTP response codes
// are modified from their defaults to custom values.
func TestAccMonitorResource_SuccessHTTPResponseCodes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create monitor with default success HTTP response codes (should be ["2xx", "3xx"])
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes("test-monitor-response-codes", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-response-codes"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", "https://example.com"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "interval", "300"),
					// Verify default response codes are set
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.#", "2"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.0", "2xx"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.1", "3xx"),
				),
			},
			// Step 2: Update to custom response codes - this should NOT cause plan inconsistencies
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes("test-monitor-response-codes", []string{"200", "201", "202"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-response-codes"),
					// Verify custom response codes are now set
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.#", "3"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.0", "200"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.1", "201"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.2", "202"),
				),
			},
			// Step 3: Remove the attribute. Back to defaults
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes("test-monitor-response-codes", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.#", "2"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.0", "2xx"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.1", "3xx"),
				),
			},
			// Idempotency
			{
				Config:             testAccMonitorResourceConfigWithSuccessHTTPResponseCodes("test-monitor-response-codes", nil),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccMonitorResource_PortMonitorValidation tests that PORT monitors require a port number.
func TestAccMonitorResource_PortMonitorValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test that PORT monitor without port fails
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "test-port-monitor"
    url          = "https://example.com"
    type         = "PORT"
    interval     = 300
	timeout 	 = 30
}
`,
				ExpectError: regexp.MustCompile("Port required for PORT monitor"),
			},
			// Test that PORT monitor with port succeeds
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "test-port-monitor"
    url          = "https://example.com"
    type         = "PORT"
    interval     = 300
    port         = 80
	timeout 	 = 30
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-port-monitor"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "PORT"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "port", "80"),
				),
			},
		},
	})
}

// TestAccMonitorResource_KeywordMonitorValidation tests that KEYWORD monitors require keyword fields.
func TestAccMonitorResource_KeywordMonitorValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test that KEYWORD monitor without keywordType fails
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "test-keyword-monitor"
    url          = "https://example.com"
    type         = "KEYWORD"
    interval     = 300
	timeout 	 = 30
    keyword_value = "test"
}
`,
				ExpectError: regexp.MustCompile("KeywordType required for KEYWORD monitor"),
			},
			// Test that KEYWORD monitor without keywordValue fails
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "test-keyword-monitor"
    url          = "https://example.com"
    type         = "KEYWORD"
    interval     = 300
	timeout 	 = 30
    keyword_type = "ALERT_EXISTS"
}
`,
				ExpectError: regexp.MustCompile("KeywordValue required for KEYWORD monitor"),
			},
			// Test that KEYWORD monitor with invalid keywordType fails
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "test-keyword-monitor"
    url          = "https://example.com"
    type         = "KEYWORD"
    interval     = 300
	timeout 	 = 30
    keyword_type = "INVALID_TYPE"
    keyword_value = "test"
}
`,
				ExpectError: regexp.MustCompile(`(?s)value must be one of:.*ALERT_EXISTS.*ALERT_NOT_EXISTS`),
			},
			// Test that KEYWORD monitor with valid fields succeeds
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "test-keyword-monitor"
    url          = "https://example.com"
    type         = "KEYWORD"
    interval     = 300
	timeout 	 = 30
    keyword_type = "ALERT_EXISTS"
    keyword_value = "test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-keyword-monitor"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "KEYWORD"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "keyword_type", "ALERT_EXISTS"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "keyword_value", "test"),
				),
			},
			// Test ALERT_NOT_EXISTS keyword type
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "test-keyword-monitor"
    url          = "https://example.com"
    type         = "KEYWORD"
    interval     = 300
	timeout 	 = 30
    keyword_type = "ALERT_NOT_EXISTS"
    keyword_value = "error"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-keyword-monitor"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "KEYWORD"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "keyword_type", "ALERT_NOT_EXISTS"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "keyword_value", "error"),
				),
			},
		},
	})
}

// TestAccMonitorResource_NewMonitorTypes tests the new monitor types.
func TestAccMonitorResource_NewMonitorTypes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test HEARTBEAT monitor
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "test-heartbeat-monitor"
    url          = "https://example.com"
    type         = "HEARTBEAT"
    interval     = 300
    grace_period = 60
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-heartbeat-monitor"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HEARTBEAT"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "grace_period", "60"),
				),
			},
			// Test DNS monitor
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "test-dns-monitor"
    url          = "example.com"
    type         = "DNS"
    interval     = 300
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-dns-monitor"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "DNS"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", "example.com"),
				),
			},
		},
	})
}

// TestAccMonitorResource_NewFields tests the new fields added to the monitor resource.
func TestAccMonitorResource_NewFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test response_time_threshold field
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name                     = "test-response-time-monitor"
    url                      = "https://example.com"
    type                     = "HTTP"
    interval                 = 300
	timeout 				 = 30
    response_time_threshold  = 5000
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-response-time-monitor"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "response_time_threshold", "5000"),
				),
			},
			// Test regional_data field
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name          = "test-regional-monitor"
    url           = "https://example.com"
    type          = "HTTP"
    interval      = 300
	timeout 	  = 30
    regional_data = "na"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-regional-monitor"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "regional_data", "na"),
				),
			},
			// Test both new fields together
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name                     = "test-combined-monitor"
    url                      = "https://example.com"
    type                     = "HTTP"
    interval                 = 300
	timeout 	 			 = 30
    response_time_threshold  = 3000
    regional_data            = "eu"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-combined-monitor"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "response_time_threshold", "3000"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "regional_data", "eu"),
				),
			},
		},
	})
}

// TestAccMonitorResource_InvalidMonitorType tests that invalid monitor types are rejected.
func TestAccMonitorResource_InvalidMonitorType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test invalid monitor type
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "test-invalid-monitor"
    url          = "https://example.com"
    type         = "INVALID_TYPE"
    interval     = 300
	timeout 	 = 30
}
`,
				ExpectError: regexp.MustCompile(`(?s)value must be one of:.*HTTP.*KEYWORD.*PING.*PORT.*HEARTBEAT.*DNS`),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_UsesTimeout(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "uptimerobot_monitor" "test" {
  name     = "acc-http"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  timeout  = 30
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "timeout", "30"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "grace_period"),
				),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_DefaultTimeout_WhenOmitted(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
  name     = "acc-http-no-timeout"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  // timeout omitted on purpose
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					// Must be concretized by provider after apply
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "timeout", "30"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "grace_period"),
				),
			},
		},
	})
}

func TestAcc_Monitor_DNS_And_PING_IgnoreTimeoutAndGrace(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// DNS with neither timeout nor grace_period
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "dns" {
  name     = "acc-dns"
  type     = "DNS"
  url      = "example.com"
  interval = 300
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.dns", "type", "DNS"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.dns", "timeout"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.dns", "grace_period"),
				),
			},
			{
				// PING with neither timeout nor grace_period
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "ping" {
  name     = "acc-ping"
  type     = "PING"
  url      = "1.1.1.1"
  interval = 300
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.ping", "type", "PING"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.ping", "timeout"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.ping", "grace_period"),
				),
			},
		},
	})
}

func TestAcc_Monitor_Heartbeat_UsesGrace(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "uptimerobot_monitor" "hb" {
  name         = "acc-heartbeat"
  type         = "HEARTBEAT"
  url          = "https://example.com"
  interval     = 300
  grace_period = 120
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.hb", "type", "HEARTBEAT"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.hb", "grace_period", "120"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.hb", "timeout"),
				),
			},
		},
	})
}

func TestAcc_Monitor_Heartbeat_Grace_Bounds_OK(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // min=0
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "hb" {
  name         = "hb-min"
  type         = "HEARTBEAT"
  url          = "https://example.com"
  interval     = 300
  grace_period = 0
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.hb", "grace_period", "0"),
				),
			},
			{ // max=86400
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "hb" {
  name         = "hb-max"
  type         = "HEARTBEAT"
  url          = "https://example.com"
  interval     = 300
  grace_period = 86400
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.hb", "grace_period", "86400"),
				),
			},
		},
	})
}

func TestAcc_Monitor_Heartbeat_Grace_Invalid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "hb" {
  name         = "hb-bad-low"
  type         = "HEARTBEAT"
  url          = "https://example.com"
  interval     = 300
  grace_period = -1
}
`,
				ExpectError: regexp.MustCompile(`must be between 0 and 86400`),
			},
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "hb" {
  name         = "hb-bad-high"
  type         = "HEARTBEAT"
  url          = "https://example.com"
  interval     = 300
  grace_period = 86401
}
`,
				ExpectError: regexp.MustCompile(`must be between 0 and 86400`),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_PostBody_RoundTrip(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create with body
			{
				Config: testAccMonitorResourceConfigWithBody("acc-body", `jsonencode({foo="bar", n=1})`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "POST"),
					// type should be computed to RAW_JSON
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "RAW_JSON"),
					// body should be set; we don't assert exact string to avoid normalization brittleness
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "post_value_data"),
				),
			},
			// update with different body (no diff loops)
			{
				Config: testAccMonitorResourceConfigWithBody("acc-body", `jsonencode({foo="baz", m=2})`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "RAW_JSON"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "post_value_data"),
				),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_PostBody_ClearByRemoving(t *testing.T) {
	name := "acc-body-clear"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// set a body first
			{
				Config: testAccMonitorResourceConfigWithBody(name, `jsonencode({hello="world"})`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "RAW_JSON"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "post_value_data"),
				),
			},
			// remove the attribute from config → should clear on server (requires post_value_data NOT Computed)
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com/echo"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
				),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_GetHead_NoBodyAllowed(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
  name             = "acc-get-body-error"
  url              = "https://example.com"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "GET"
  post_value_data  = jsonencode({oops="nope"})
}`,
				ExpectError: regexp.MustCompile(`Request body not allowed for GET/HEAD`),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_PostKV_RoundTrip(t *testing.T) {
	name := "acc-kv"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Create with KV
			{
				Config: testAccMonitorResourceConfigWithKV(name, map[string]string{"a": "1", "b": "2"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "POST"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "KEY_VALUE"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_kv.%", "2"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_kv.a", "1"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_kv.b", "2"),
				),
			},
			// Update KV
			{
				Config: testAccMonitorResourceConfigWithKV(name, map[string]string{"a": "9"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "KEY_VALUE"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_kv.%", "1"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_kv.a", "9"),
				),
			},
			// Clear by removing post_value_kv
			{
				Config: testAccMonitorResourceConfigPostNoBody(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
				),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_Post_NoBody_StablePlan(t *testing.T) {
	name := "acc-post-nobody"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Start at GET with no body
			{Config: testAccMonitorResourceConfigGetNoBody(name)},
			// Switch to POST with no body – plan must be valid (this used to fail)
			{Config: testAccMonitorResourceConfigPostNoBody(name), PlanOnly: true, ExpectNonEmptyPlan: true},
			// Apply it – still no body attrs
			{
				Config: testAccMonitorResourceConfigPostNoBody(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "POST"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
				),
			},
			// Re-plan same config – must be idempotent
			{Config: testAccMonitorResourceConfigPostNoBody(name), PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}

func TestAcc_Monitor_HTTP_MethodSwitch_ClearsBody(t *testing.T) {
	name := "acc-method-switch"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// POST + JSON
			{
				Config: testAccMonitorResourceConfigWithBody(name, `jsonencode({hello="world"})`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "POST"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "RAW_JSON"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "post_value_data"),
				),
			},
			// Switch to GET – body must be cleared
			{
				Config: testAccMonitorResourceConfigGetNoBody(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "GET"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
				),
			},
			// Switch back to POST – still no body provided, must remain empty and stable
			{
				Config: testAccMonitorResourceConfigPostNoBody(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "POST"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
				),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_DefaultMethod_GET(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorResourceConfig("acc-default-method"),
				Check:  resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "GET"),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_Body_Switch_JSON_KV(t *testing.T) {
	name := "acc-switch-json-kv"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{ // start JSON
				Config: testAccMonitorResourceConfigWithBody(name, `jsonencode({a="1"})`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "RAW_JSON"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
				),
			},
			{ // switch to KV
				Config: testAccMonitorResourceConfigWithKV(name, map[string]string{"a": "1"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "KEY_VALUE"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_kv.%", "1"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
				),
			},
			{ // back to JSON
				Config: testAccMonitorResourceConfigWithBody(name, `jsonencode({b="2"})`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "RAW_JSON"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
				),
			},
		},
	})
}

func TestAcc_Monitor_CreatePlanOnly_NoExistingState(t *testing.T) {
	// This specifically exercises ModifyPlan with a null state on first create.
	// It should produce a non-empty plan and not panic / error.
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             testAccMonitorResourceConfig("acc-planonly-from-scratch"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
