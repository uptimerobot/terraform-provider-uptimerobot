package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

/*
	Config addition rules:

	- Config helpers for the common, repeatable configs like HTTP base monitor, headers, tags, MWs, etc.
	They reduces duplication and makes refactors, such as adding a timeout = 30 to HTTP easy to be performed.

	- Inline configs only when the test’s readability depends on seeing the exact HCL schema in the test.
	For example, negative cases that assert a specific validation error, or tiny one-off / one time scenarios.

	- Alert contacts currently lack resource/management functionality, so the related
	acceptance tests only run when the environment variable UPTIMEROBOT_TEST_ALERT_CONTACT_ID
	is set. Obtain the contact ID via API GET /user/alert-contacts and export it before running
	them to unlock those tests locally or in CI/CD.

*/

// ---------- Stable time helpers for maintenance windows ----------

// mwDateTimeIn returns date and time in format of strings as YYYY-MM-DD, HH:mm:ss for "now + d",
// aligned to a full minute to avoid server-side normalization mismatches.
func mwDateTimeIn(d time.Duration) (string, string) {
	t := time.Now().UTC().Add(d).Truncate(time.Minute)
	return t.Format("2006-01-02"), t.Format("15:04:05")
}

// twoMWDateTimes returns two future timestamps used by tests (5m, 10m).
func twoMWDateTimes() (d1, t1, d2, t2 string) {
	d1, t1 = mwDateTimeIn(5 * time.Minute)
	d2, t2 = mwDateTimeIn(10 * time.Minute)
	return
}

// ------------------------ Config helpers ------------------------

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

func testAccMonitorResourceConfigWithTags(name string, tags []string) string {
	tagsStr := ""
	if tags != nil {
		if len(tags) == 0 {
			tagsStr = `
    tags = []`
		} else {
			tagsStr = fmt.Sprintf(`
    tags = [%s]`, joinQuotedStrings(tags))
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
`, name, tagsStr)
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
	method := `http_method_type = "GET"`
	if headers != nil {
		method = `http_method_type = "POST"`
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
  interval = 300
  timeout  = 30
  %s%s
}
`, name, method, hdr)
}

func testAccMonitorResourceConfigWithBody(name string, body string) string {
	// body should be an HCL expression, e.g. ` + "`jsonencode({foo=\"bar\", n=1})` or `null`" + `
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
  custom_http_headers = { "content-type" = "application/x-www-form-urlencoded" }%s
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
  url              = "https://example.com/echo"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "GET"
}
`, name)
}

//nolint:unparam // name kept for symmetry with other helpers & future reuse
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

func testAccMonitorResourceConfigWithSSLPeriod(name string, days []int) string {
	cfg := ""
	if days != nil {
		if len(days) == 0 {
			cfg = `
  config = {
    ssl_expiration_period_days = []
  }`
		} else {
			cfg = fmt.Sprintf(`
  config = {
    ssl_expiration_period_days = [%s]
  }`, joinInts(days))
		}
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = "https://example.com"
  type     = "HTTP"
  interval = 300
  timeout  = 30%s
}
`, name, cfg)
}

// ---------- MW helpers that embed STABLE (literal) date/time ----------

func testAccConfigMonitorWithTwoMWs(sfx string) string {
	d1, t1, d2, t2 := twoMWDateTimes()
	return fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "a" {
  name      = "%[1]s-a"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 15
}

resource "uptimerobot_maintenance_window" "b" {
  name      = "%[1]s-b"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 20
}

resource "uptimerobot_monitor" "test" {
  name     = "%[1]s-monitor"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  maintenance_window_ids = [
    uptimerobot_maintenance_window.a.id,
    uptimerobot_maintenance_window.b.id,
  ]
}
`, sfx, d1, t1, d2, t2)
}

func testAccConfigMonitorWithOneMW(sfx string) string {
	d1, t1, d2, t2 := twoMWDateTimes()
	return fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "a" {
  name      = "%[1]s-a"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 15
}

resource "uptimerobot_maintenance_window" "b" {
  name      = "%[1]s-b"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 20
}

resource "uptimerobot_monitor" "test" {
  name     = "%[1]s-monitor"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  maintenance_window_ids = [
    uptimerobot_maintenance_window.b.id,
  ]
}
`, sfx, d1, t1, d2, t2)
}

func testAccConfigMonitorNoMW(sfx string) string {
	d1, t1, d2, t2 := twoMWDateTimes()
	return fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "a" {
  name      = "%[1]s-a"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 15
}

resource "uptimerobot_maintenance_window" "b" {
  name      = "%[1]s-b"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 20
}

resource "uptimerobot_monitor" "test" {
  name     = "%[1]s-monitor"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  maintenance_window_ids = []
}
`, sfx, d1, t1, d2, t2)
}

// -------------------------- Helpers --------------------------

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

// ---------------------- Acceptance tests ----------------------

func TestAccMonitorResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
			{
				Config:             testAccMonitorResourceConfigWithAlertContactObjects("test-monitor-alerts", nil),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccMonitorResource_AlertContacts_ExplicitEmpty(t *testing.T) {
	id := mustAlertContactID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// 1) Start with one contact assigned
			{
				Config: testAccMonitorResourceConfigWithAlertContactObjects("test-monitor-contacts-empty", []string{id}),
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
			// 2) Explicitly set to empty list. Plan should exist and clears server
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
  name     = "test-monitor-contacts-empty"
  url      = "https://example.com"
  type     = "HTTP"
  interval = 300
  timeout  = 30
  assigned_alert_contacts = [] // explicit empty
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// 3) Apply explicit empty. State should be an empty set
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
  name     = "test-monitor-contacts-empty"
  url      = "https://example.com"
  type     = "HTTP"
  interval = 300
  timeout  = 30
  assigned_alert_contacts = []
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts.#", "0"),
				),
			},
			// 4) Idempotent re-plan with explicit empty
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
  name     = "test-monitor-contacts-empty"
  url      = "https://example.com"
  type     = "HTTP"
  interval = 300
  timeout  = 30
  assigned_alert_contacts = []
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// 5) Remove the attribute entirely. Attribute should be omitted in state
			{
				Config: testAccMonitorResourceConfigWithAlertContactObjects("test-monitor-contacts-empty", nil),
				Check:  resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts"),
			},
		},
	})
}

// TestAccMonitorResource_Tags tests the specific case where tags
// are added to an existing monitor that was initially created without any.
func TestAccMonitorResource_Tags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "tags"),
				),
			},
			// Step 2: Add tags to existing monitor - this should NOT fail
			{
				Config: testAccMonitorResourceConfigWithTags("test-monitor-tags", []string{"production", "web"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-tags"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "tags.#", "2"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "tags.*", "production"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "tags.*", "web"),
				),
			},
			// Step 3: Remove tags - set back to empty
			{
				Config: testAccMonitorResourceConfigWithTags("test-monitor-tags", []string{}),
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
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// 1) Create without headers
			{
				Config: testAccMonitorResourceConfig(name),
				Check:  resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "custom_http_headers"),
			},
			// 2) Add a header
			{
				Config: testAccMonitorResourceConfigWithHeaders(name, map[string]string{"x-test": "one"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.%", "1"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.x-test", "one"),
				),
			},
			// 3) Change header value
			{
				Config: testAccMonitorResourceConfigWithHeaders(name, map[string]string{"x-test": "two"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.%", "1"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.x-test", "two"),
				),
			},
			// 4) Clear by removing the block entirely
			{
				Config: testAccMonitorResourceConfig(name),
				Check:  resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "custom_http_headers"),
			},
			// 5) Import to ensure state matches API with no headers
			{
				ResourceName:            "uptimerobot_monitor.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeout", "status", "custom_http_headers"},
			},
		},
	})
}

func TestAccMonitorResource_CustomHTTPHeaders_ContentTypeWithBody(t *testing.T) {
	name := "test-monitor-headers-ct"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// 1) Create with POST and JSON body with header 'Content-Type'
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com/echo"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  post_value_data  = jsonencode({foo="bar"})
  custom_http_headers = { "content-type" = "application/json" }
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "POST"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "RAW_JSON"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.%", "1"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.content-type", "application/json"),
				),
			},
			// 2) Change Content-Type value
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com/echo"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  post_value_data  = jsonencode({foo="bar"})
  custom_http_headers = { "content-type" = "application/x-www-form-urlencoded" }
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.%", "1"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "custom_http_headers.content-type", "application/x-www-form-urlencoded"),
				),
			},
			// 3) Remove headers while body remains
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com/echo"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  post_value_data  = jsonencode({foo="bar"})
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "custom_http_headers"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "RAW_JSON"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "post_value_data"),
				),
			},
		},
	})
}

// TestAccMonitorResource_MaintenanceWindows tests the specific case where maintenance window IDs
// are added to an existing monitor that was initially created without any.
func TestAccMonitorResource_MaintenanceWindows(t *testing.T) {
	sfx := "acc-mw"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: attach two MWs
				Config: testAccConfigMonitorWithTwoMWs(sfx),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids.#", "2"),
					resource.TestCheckTypeSetElemAttrPair(
						"uptimerobot_monitor.test", "maintenance_window_ids.*",
						"uptimerobot_maintenance_window.a", "id",
					),
					resource.TestCheckTypeSetElemAttrPair(
						"uptimerobot_monitor.test", "maintenance_window_ids.*",
						"uptimerobot_maintenance_window.b", "id",
					),
				),
			},
			{
				// Step 2: keep only MW b
				Config: testAccConfigMonitorWithOneMW(sfx),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(
						"uptimerobot_monitor.test", "maintenance_window_ids.*",
						"uptimerobot_maintenance_window.b", "id",
					),
				),
			},
			{
				// Step 3: detach all
				Config: testAccConfigMonitorNoMW(sfx),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "maintenance_window_ids.#", "0"),
				),
			},
		},
	})
}

func TestAccMonitorResource_SuccessHTTPResponseCodes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// 1) Create with attr omitted. Defaults may be set on server, attribute is ABSENT in state
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes("test-monitor-response-codes", nil),
				Check:  resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "success_http_response_codes"),
			},

			// 2) Set custom codes
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes("test-monitor-response-codes", []string{"200", "201", "202"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.#", "3"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "200"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "201"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "202"),
				),
			},

			// 3) Omit attr (nil). PRESERVE existing custom values on server and still PRESENT in state
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes("test-monitor-response-codes", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.#", "3"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "200"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "201"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "202"),
				),
			},

			// 4) Explicit empty []. Provider sends empty slice and server resets to defaults, and attr ABSENT in state
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes("test-monitor-response-codes", []string{}),
				Check:  resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.#", "0"),
			},
			// 5) Idempotent re-plan with omit
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
	const name = "test-newfields"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) threshold only
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name                    = %q
  url                     = "https://example.com"
  type                    = "HTTP"
  interval                = 300
  timeout                 = 30
  response_time_threshold = 5000
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "response_time_threshold", "5000"),
				),
			},
			// 2) change threshold
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name                    = %q
  url                     = "https://example.com"
  type                    = "HTTP"
  interval                = 300
  timeout                 = 30
  response_time_threshold = 3000
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "response_time_threshold", "3000"),
				),
			},
			// 3) add regional_data as well
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name                    = %q
  url                     = "https://example.com"
  type                    = "HTTP"
  interval                = 300
  timeout                 = 30
  response_time_threshold = 3000
  regional_data           = "eu"
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "response_time_threshold", "3000"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "regional_data", "eu"),
				),
			},
			// 4) idempotency re-plan
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name                    = %q
  url                     = "https://example.com"
  type                    = "HTTP"
  interval                = 300
  timeout                 = 30
  response_time_threshold = 3000
  regional_data           = "eu"
}`, name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccMonitorResource_InvalidMonitorType tests that invalid monitor types are rejected.
func TestAccMonitorResource_InvalidMonitorType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// 1) POST + JSON
			{
				Config: testAccMonitorResourceConfigWithBody(name, `jsonencode({hello="world"})`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "POST"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "RAW_JSON"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "post_value_data"),
				),
			},
			// 2) Switch to GET with no body
			{
				Config: testAccMonitorResourceConfigGetNoBody(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "GET"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
				),
			},
			// 3) URL change only with GET and no body to verift URL update
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com" // change URL only
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "GET"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "GET"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", "https://example.com"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
				),
			},
			// 4) Switch back to POST with no body, keep new url – should remain clean and stable
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "POST"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", "https://example.com"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
				),
			},
			// 5) Idempotent re-plan on step 5 confog
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "https://example.com"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
}
`, name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_HTTP_DefaultMethod_GET(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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

func TestAcc_Monitor_CheckSSLErrors_DefaultFalse(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorResourceConfig("acc-sslerrs-default"),
				Check:  resource.TestCheckResourceAttr("uptimerobot_monitor.test", "check_ssl_errors", "false"),
			},
			// idempotency
			{
				Config:             testAccMonitorResourceConfig("acc-sslerrs-default"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_CheckSSLErrors_ExplicitTrue(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
  name             = "acc-sslerrs-true"
  url              = "https://example.com"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  check_ssl_errors = true
}
`,
				Check: resource.TestCheckResourceAttr("uptimerobot_monitor.test", "check_ssl_errors", "true"),
			},
			// flip back to false
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
  name             = "acc-sslerrs-true"
  url              = "https://example.com"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  check_ssl_errors = false
}
`,
				Check: resource.TestCheckResourceAttr("uptimerobot_monitor.test", "check_ssl_errors", "false"),
			},
		},
	})
}

func TestAcc_Monitor_Config_SSLExpirationPeriodDays(t *testing.T) {
	name := "acc-ssl-period"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{ // apply [0, 7, 30]
				Config: testAccMonitorResourceConfigWithSSLPeriod(name, []int{0, 7, 30}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "config.ssl_expiration_period_days.#", "3"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "config.ssl_expiration_period_days.*", "0"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "config.ssl_expiration_period_days.*", "7"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "config.ssl_expiration_period_days.*", "30"),
				),
			},
			{ // change to [1, 14]
				Config: testAccMonitorResourceConfigWithSSLPeriod(name, []int{1, 14}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "config.ssl_expiration_period_days.#", "2"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "config.ssl_expiration_period_days.*", "1"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "config.ssl_expiration_period_days.*", "14"),
				),
			},
			{ // remove config → attribute omitted
				Config: testAccMonitorResourceConfig(name),
				Check:  resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "config.ssl_expiration_period_days"),
			},
			{ // idempotent re-plan
				Config:             testAccMonitorResourceConfig(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_Config_SSLExpirationPeriodDays_Invalid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // out of range
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
  name     = "acc-ssl-period-invalid"
  url      = "https://example.com"
  type     = "HTTP"
  interval = 300
  timeout  = 30
  config = {
    ssl_expiration_period_days = [-1, 366]
  }
}
`,
				ExpectError: regexp.MustCompile(
					`(?s)` +
						`Attribute config\.ssl_expiration_period_days\[Value\(-1\)\] value must be between[\s\S]*0 and 365, got: -1` +
						`[\s\S]*` +
						`Attribute config\.ssl_expiration_period_days\[Value\(366\)\] value must be between[\s\S]*0 and 365, got: 366`,
				),
			},
			{ // > 10 items
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
  name     = "acc-ssl-period-too-many"
  url      = "https://example.com"
  type     = "HTTP"
  interval = 300
  timeout  = 30
  config = {
    ssl_expiration_period_days = [0,1,2,3,4,5,6,7,8,9,10]
  }
}
`,
				ExpectError: regexp.MustCompile(
					`Attribute config\.ssl_expiration_period_days (?:set|value) must contain at most 10\s+elements(?:, got: \d+)?`,
				),
			},
		},
	})
}

func TestAcc_Monitor_Name_HTMLNormalization(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}

	resourceName := "uptimerobot_monitor.test"

	cfgEncoded := `
resource "uptimerobot_monitor" "test" {
  name     = "A &amp; B <C>"
  type     = "HTTP"
  url      = "https://example.com/health"
  interval = 300
}
`
	cfgPlain := `
resource "uptimerobot_monitor" "test" {
  name     = "A & B <C>"
  type     = "HTTP"
  url      = "https://example.com/health"
  interval = 300
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create with encoded HCL, so state is encoded and we don't unescape on normal Read
			{
				Config: cfgEncoded,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "A &amp; B <C>"),
				),
			},
			// 2) Import will import unescaped to plain in state
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: false, // don't compare against encoded config
			},
			// 3) Switch config to plain - refresh reads encoded state, so plan should show a diff
			{
				Config:             cfgPlain,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// 4) Apply plain - state plain
			{
				Config: cfgPlain,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "A & B <C>"),
				),
			},
			// 5) Re-plan plain - clean
			{
				Config:             cfgPlain,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
