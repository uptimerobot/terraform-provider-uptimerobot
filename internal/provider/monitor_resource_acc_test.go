//go:build acceptance

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
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
	url := testAccUniqueURL(name)
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = "%s"
    type         = "HTTP"
    interval     = 300
	timeout   	 = 30
}
`, name, url)
}

func testAccMonitorResourceConfigWithURL(name, url string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = "%s"
    type         = "HTTP"
    interval     = 300
	timeout   	 = 30
}
`, name, url)
}

func testAccMonitorResourceConfigWithPause(name string, isPaused *bool) string {
	url := testAccUniqueURL(name)
	pause := ""
	if isPaused != nil {
		pause = fmt.Sprintf("\n  is_paused = %t", *isPaused)
	}

	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = "%s"
  type     = "HTTP"
  interval = 300
  timeout  = 30%s
}
`, name, url, pause)
}

func testAccMonitorResourceConfigWithTags(name string, tags []string) string {
	url := testAccUniqueURL(name)
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
    url          = "%s"
    type         = "HTTP"
    interval     = 300%s
	timeout      = 30
}
`, name, url, tagsStr)
}

func testAccMonitorResourceConfigWithGroupID(name string, groupID int) string {
	url := testAccUniqueURL(name)
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name      = %q
  url       = "%s"
  type      = "HTTP"
  interval  = 300
  timeout   = 30
  group_id  = %d
}
`, name, url, groupID)
}

// nolint:unparam // kept for symmetry with other helpers & future reuse
func testAccMonitorResourceConfigWithSuccessHTTPResponseCodes(name string, responseCodes []string) string {
	url := testAccUniqueURL(name)
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
    url          = "%s"
    type         = "HTTP"
    interval     = 300%s
    timeout      = 30
}
`, name, url, responseCodesStr)
}

func testAccMonitorResourceConfigWithHeaders(name string, headers map[string]string) string {
	url := testAccUniqueURL(name)
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
  url      = "%s"
  type     = "HTTP"
  interval = 300
  timeout  = 30
  %s%s
}
`, name, url, method, hdr)
}

func testAccMonitorResourceConfigWithBody(name string, body string) string {
	url := fmt.Sprintf("%s/echo", testAccUniqueURL(name))
	// body should be an HCL expression, e.g. ` + "`jsonencode({foo=\"bar\", n=1})` or `null`" + `
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "%s"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  post_value_data  = %s
}
`, name, url, body)
}

func testAccMonitorResourceConfigWithKV(name string, kv map[string]string) string {
	url := fmt.Sprintf("%s/echo", testAccUniqueURL(name))
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
  url              = "%s"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  custom_http_headers = { "content-type" = "application/x-www-form-urlencoded" }%s
}
`, name, url, body)
}

func testAccMonitorResourceConfigPostNoBody(name string) string {
	url := fmt.Sprintf("%s/echo", testAccUniqueURL(name))
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "%s"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  // no post_value_data / post_value_kv on purpose
}
`, name, url)
}

func testAccMonitorResourceConfigGetNoBody(name string) string {
	return testAccMonitorResourceConfigGetNoBodyAtURL(name, testAccUniqueURL(name))
}

func testAccMonitorResourceConfigGetNoBodyAtURL(name, url string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "%s"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "GET"
}
`, name, url)
}

//nolint:unparam // name kept for symmetry with other helpers & future reuse
func testAccMonitorResourceConfigWithAlertContactObjects(name string, ids []string) string {
	url := testAccUniqueURL(name)
	ac := ""
	if len(ids) > 0 {
		ac = "\n  assigned_alert_contacts = ["
		for i, id := range ids {
			if i > 0 {
				ac += ","
			}
			ac += fmt.Sprintf(`{ alert_contact_id = %q, threshold = 0, recurrence = 0 }`, id)
		}
		ac += "]"
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = "%s"
  type     = "HTTP"
  interval = 300%s
  timeout  = 30
}
`, name, url, ac)
}

func testAccMonitorResourceConfigWithSSLPeriod(name string, days []int) string {
	url := testAccUniqueURL(name)
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
  url      = "%s"
  type     = "HTTP"
  interval = 300
  timeout  = 30
%s
}
`, name, url, cfg)
}

func testAccMonitorResourceConfigWithAPIAssertions(name, logic, comparison, targetExpr string) string {
	url := testAccUniqueURL(name)
	target := ""
	if strings.TrimSpace(targetExpr) != "" {
		target = "\n          target     = " + targetExpr
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = "%s"
  type     = "API"
  interval = 300
  timeout  = 30

  config = {
    api_assertions = {
      logic = %q
      checks = [
        {
          property   = "$.status"
          comparison = %q%s
        }
      ]
    }
  }
}
`, name, url, logic, comparison, target)
}

// ---------- MW helpers that embed STABLE (literal) date/time ----------

func testAccConfigMonitorWithTwoMWs(sfx string) string {
	monitorName := fmt.Sprintf("%s-monitor", sfx)
	url := testAccUniqueURL(monitorName)
	d1, t1, d2, t2 := twoMWDateTimes()
	return fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "a" {
  name      = "%[1]s-a"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 15
  auto_add_monitors = false
}

resource "uptimerobot_maintenance_window" "b" {
  name      = "%[1]s-b"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 20
  auto_add_monitors = false
}

resource "uptimerobot_monitor" "test" {
  name     = "%[1]s-monitor"
  type     = "HTTP"
  url      = "%[6]s"
  interval = 300

  maintenance_window_ids = [
    uptimerobot_maintenance_window.a.id,
    uptimerobot_maintenance_window.b.id,
  ]
}
`, sfx, d1, t1, d2, t2, url)
}

func testAccConfigMonitorWithOneMW(sfx string) string {
	monitorName := fmt.Sprintf("%s-monitor", sfx)
	url := testAccUniqueURL(monitorName)
	d1, t1, d2, t2 := twoMWDateTimes()
	return fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "a" {
  name      = "%[1]s-a"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 15
  auto_add_monitors = false
}

resource "uptimerobot_maintenance_window" "b" {
  name      = "%[1]s-b"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 20
  auto_add_monitors = false
}

resource "uptimerobot_monitor" "test" {
  name     = "%[1]s-monitor"
  type     = "HTTP"
  url      = "%[6]s"
  interval = 300

  maintenance_window_ids = [
    uptimerobot_maintenance_window.b.id,
  ]
}
`, sfx, d1, t1, d2, t2, url)
}

func testAccConfigMonitorNoMW(sfx string) string {
	monitorName := fmt.Sprintf("%s-monitor", sfx)
	url := testAccUniqueURL(monitorName)
	d1, t1, d2, t2 := twoMWDateTimes()
	return fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "a" {
  name      = "%[1]s-a"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 15
  auto_add_monitors = false
}

resource "uptimerobot_maintenance_window" "b" {
  name      = "%[1]s-b"
  interval  = "once"
  date      = %q
  time      = %q
  duration  = 20
  auto_add_monitors = false
}

resource "uptimerobot_monitor" "test" {
  name     = "%[1]s-monitor"
  type     = "HTTP"
  url      = "%[6]s"
  interval = 300

  maintenance_window_ids = []
}
`, sfx, d1, t1, d2, t2, url)
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

// testAccUniqueURL produces a stable and per-name unique URL to satisfy API
// deduplication validations for GET and HEAD monitors.
func testAccUniqueURL(name string) string {
	if v, ok := uniqueURLCache.Load(name); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	slug := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, name)
	if strings.Trim(slug, "-") == "" {
		slug = "monitor"
	}
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	url := fmt.Sprintf("https://example.com/%s-%s", slug, suffix)
	uniqueURLCache.Store(name, url)
	return url
}

var uniqueURLCache sync.Map
var uniqueDomainCache sync.Map

// testAccUniqueDomain returns a unique domain for API validations like DNS monitors.
func testAccUniqueDomain(name string) string {
	if v, ok := uniqueDomainCache.Load(name); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	slug := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, name)
	if strings.Trim(slug, "-") == "" {
		slug = "dns"
	}
	suffix := acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	domain := fmt.Sprintf("%s-%s.example.com", slug, suffix)
	uniqueDomainCache.Store(name, domain)
	return domain
}

// ---------------------- Acceptance tests ----------------------

func TestAccMonitorResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor")
	updatedName := acctest.RandomWithPrefix("test-monitor-updated")
	url := testAccUniqueURL(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMonitorResourceConfigWithURL(name, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", url),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "interval", "300"),
				),
			},
			// Update testing
			{
				Config: testAccMonitorResourceConfigWithURL(updatedName, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", updatedName),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", url),
				),
			},
			// Import testing
			{
				ResourceName:            "uptimerobot_monitor.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeout", "status", "group_id", "name", "is_paused"},
			},
		},
	})
}

func TestAccMonitorResource_IsPaused_StartStop(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-monitor-start-stop")
	paused := true
	started := false

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorResourceConfigWithPause(name, &paused),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "is_paused", "true"),
				),
			},
			{
				Config: testAccMonitorResourceConfigWithPause(name, &started),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "is_paused", "false"),
				),
			},
			{
				Config: testAccMonitorResourceConfigWithPause(name, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "is_paused"),
				),
			},
		},
	})
}

func TestAccMonitorResource_AlertContacts(t *testing.T) {
	id := mustAlertContactID(t)
	name := acctest.RandomWithPrefix("test-monitor-alerts")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Step 1: create with no contacts
			{
				Config: testAccMonitorResourceConfigWithAlertContactObjects(name, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts"),
				),
			},
			// Step 2: add one contact
			{
				Config: testAccMonitorResourceConfigWithAlertContactObjects(name, []string{id}),
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
				Config: testAccMonitorResourceConfigWithAlertContactObjects(name, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts"),
				),
			},
			{
				Config:             testAccMonitorResourceConfigWithAlertContactObjects(name, nil),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccMonitorResource_AlertContacts_ExplicitEmpty(t *testing.T) {
	id := mustAlertContactID(t)
	name := acctest.RandomWithPrefix("test-monitor-contacts-empty")
	url := testAccUniqueURL(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// 1) Start with one contact assigned
			{
				Config: testAccMonitorResourceConfigWithAlertContactObjects(name, []string{id}),
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
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = %q
  type     = "HTTP"
  interval = 300
  timeout  = 30
  assigned_alert_contacts = [] // explicit empty
}
`, name, url),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// 3) Apply explicit empty. State should be an empty set
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = %q
  type     = "HTTP"
  interval = 300
  timeout  = 30
  assigned_alert_contacts = []
}
`, name, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts.#", "0"),
				),
			},
			// 4) Idempotent re-plan with explicit empty
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = %q
  type     = "HTTP"
  interval = 300
  timeout  = 30
  assigned_alert_contacts = []
}
`, name, url),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// 5) Remove the attribute entirely. Attribute should be omitted in state
			{
				Config: testAccMonitorResourceConfigWithAlertContactObjects(name, nil),
				Check:  resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts"),
			},
		},
	})
}

func TestAccMonitorResource_AlertContacts_MissingThreshold(t *testing.T) {
	id := mustAlertContactID(t)
	name := acctest.RandomWithPrefix("test-missing-threshold")
	url := testAccUniqueURL(name)

	cfg := testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = %q
  type     = "HTTP"
  interval = 300
  timeout  = 30

  assigned_alert_contacts = [
    {
      alert_contact_id = %q
      recurrence       = 0
      # threshold omitted on purpose
    }
  ]
}
`, name, url, id)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfg,
				ExpectError: regexp.MustCompile(`attribute "threshold" is required`),
			},
		},
	})
}

func TestAccMonitorResource_AlertContacts_MissingRecurrence(t *testing.T) {
	id := mustAlertContactID(t)
	name := acctest.RandomWithPrefix("test-missing-recurrence")
	url := testAccUniqueURL(name)

	cfg := testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = %q
  type     = "HTTP"
  interval = 300
  timeout  = 30

  assigned_alert_contacts = [
    {
      alert_contact_id = %q
      threshold        = 0
      # recurrence omitted on purpose
    }
  ]
}
`, name, url, id)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfg,
				ExpectError: regexp.MustCompile(`attribute "recurrence" is required`),
			},
		},
	})
}

// TestAccMonitorResource_Tags tests the specific case where tags
// are added to an existing monitor that was initially created without any.
func TestAccMonitorResource_Tags(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-tags")
	url := testAccUniqueURL(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create monitor without tags
			{
				Config: testAccMonitorResourceConfigWithTags(name, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", url),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "interval", "300"),
					// Verify no tags are set initially
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "tags"),
				),
			},
			// Step 2: Add tags to existing monitor - this should NOT fail
			{
				Config: testAccMonitorResourceConfigWithTags(name, []string{"production", "web"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "tags.#", "2"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "tags.*", "production"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "tags.*", "web"),
				),
			},
			// Step 3: Remove tags - set back to empty
			{
				Config: testAccMonitorResourceConfigWithTags(name, []string{}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					// Verify tags are removed
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccMonitorResource_CustomHTTPHeaders(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-headers")
	url := testAccUniqueURL(name)

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
				ImportStateVerifyIgnore: []string{"timeout", "status", "custom_http_headers", "group_id", "is_paused"},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", url),
				),
			},
		},
	})
}

func TestAccMonitorResource_CustomHTTPHeaders_ContentTypeWithBody(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-headers-ct")
	url := fmt.Sprintf("%s/echo", testAccUniqueURL(name))

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
  url              = "%s"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  post_value_data  = jsonencode({foo="bar"})
  custom_http_headers = { "content-type" = "application/json" }
}
`, name, url),
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
  url              = "%s"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  post_value_data  = jsonencode({foo="bar"})
  custom_http_headers = { "content-type" = "application/x-www-form-urlencoded" }
}
`, name, url),
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
  url              = "%s"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
  post_value_data  = jsonencode({foo="bar"})
}
`, name, url),
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
	sfx := acctest.RandomWithPrefix("acc-mw")

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
	name := acctest.RandomWithPrefix("test-monitor-response-codes")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// 1) Create with attr omitted. Defaults may be set on server, attribute is ABSENT in state
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes(name, nil),
				Check:  resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "success_http_response_codes"),
			},

			// 2) Set custom codes
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes(name, []string{"200", "201", "202"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.#", "3"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "200"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "201"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "202"),
				),
			},

			// 3) Omit attr (nil). PRESERVE existing custom values on server and still PRESENT in state
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes(name, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.#", "3"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "200"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "201"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_monitor.test", "success_http_response_codes.*", "202"),
				),
			},

			// 4) Explicit empty []. Provider sends empty slice and server resets to defaults, and attr ABSENT in state
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes(name, []string{}),
				Check:  resource.TestCheckResourceAttr("uptimerobot_monitor.test", "success_http_response_codes.#", "0"),
			},
			// 5) Idempotent re-plan with omit
			{
				Config:             testAccMonitorResourceConfigWithSuccessHTTPResponseCodes(name, nil),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccMonitorResource_PortMonitorValidation tests that PORT monitors require a port number.
func TestAccMonitorResource_PortMonitorValidation(t *testing.T) {
	name := acctest.RandomWithPrefix("test-port-monitor")
	url := testAccUniqueURL(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test that PORT monitor without port fails
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = %q
    type         = "PORT"
    interval     = 300
	timeout 	 = 30
}
`, name, url),
				ExpectError: regexp.MustCompile("Port required for PORT/UDP monitor"),
			},
			// Test that PORT monitor with port succeeds
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = %q
    type         = "PORT"
    interval     = 300
    port         = 80
	timeout 	 = 30
}
`, name, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", name),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "PORT"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "port", "80"),
				),
			},
		},
	})
}

// TestAccMonitorResource_KeywordMonitorValidation tests that KEYWORD monitors require keyword fields.
func TestAccMonitorResource_KeywordMonitorValidation(t *testing.T) {
	baseName := acctest.RandomWithPrefix("test-keyword-monitor")
	url := testAccUniqueURL(baseName + "-exists")
	urlNot := testAccUniqueURL(baseName + "-not")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test that KEYWORD monitor without keywordType fails
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
	resource "uptimerobot_monitor" "test" {
	    name         = %q
	    url          = %q
	    type         = "KEYWORD"
	    interval     = 300
		timeout 	 = 30
	    keyword_case_type = "CaseInsensitive"
	    keyword_value = "test"
	}
	`, baseName, url),
				ExpectError: regexp.MustCompile("KeywordType required for KEYWORD monitor"),
			},
			// Test that KEYWORD monitor without keywordCaseType fails
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
	resource "uptimerobot_monitor" "test" {
	    name         = %q
	    url          = %q
	    type         = "KEYWORD"
	    interval     = 300
		timeout 	 = 30
	    keyword_type = "ALERT_EXISTS"
	    keyword_value = "test"
	}
	`, baseName, url),
				ExpectError: regexp.MustCompile("KeywordCaseType required for KEYWORD monitor"),
			},
			// Test that KEYWORD monitor without keywordValue fails
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
	resource "uptimerobot_monitor" "test" {
	    name         = %q
	    url          = %q
	    type         = "KEYWORD"
	    interval     = 300
		timeout 	 = 30
	    keyword_type = "ALERT_EXISTS"
	    keyword_case_type = "CaseInsensitive"
	}
	`, baseName, url),
				ExpectError: regexp.MustCompile("KeywordValue required for KEYWORD monitor"),
			},
			// Test that KEYWORD monitor with invalid keywordType fails
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
	resource "uptimerobot_monitor" "test" {
	    name         = %q
	    url          = %q
	    type         = "KEYWORD"
	    interval     = 300
		timeout 	 = 30
	    keyword_type = "INVALID_TYPE"
	    keyword_case_type = "CaseInsensitive"
	    keyword_value = "test"
	}
	`, baseName, url),
				ExpectError: regexp.MustCompile(`(?s)value must be one of:.*ALERT_EXISTS.*ALERT_NOT_EXISTS`),
			},
			// Validate both keyword types succeed
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
	resource "uptimerobot_monitor" "exists" {
	    name         = "%s-exists"
	    url          = "%s"
	    type         = "KEYWORD"
	    interval     = 300
		timeout 	 = 30
	    keyword_type = "ALERT_EXISTS"
	    keyword_case_type = "CaseInsensitive"
	    keyword_value = "test"
	}

	resource "uptimerobot_monitor" "not" {
	    name         = "%s-not"
	    url          = "%s"
	    type         = "KEYWORD"
	    interval     = 300
		timeout 	 = 30
	    keyword_type = "ALERT_NOT_EXISTS"
	    keyword_case_type = "CaseInsensitive"
	    keyword_value = "error"
	}
	`, baseName, url, baseName, urlNot),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.exists", "type", "KEYWORD"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.exists", "keyword_type", "ALERT_EXISTS"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.exists", "keyword_case_type", "CaseInsensitive"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.exists", "keyword_value", "test"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.not", "type", "KEYWORD"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.not", "keyword_type", "ALERT_NOT_EXISTS"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.not", "keyword_case_type", "CaseInsensitive"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.not", "keyword_value", "error"),
				),
			},
		},
	})
}

// TestAccMonitorResource_NewMonitorTypes tests the new monitor types.
func TestAccMonitorResource_NewMonitorTypes(t *testing.T) {
	hbName := acctest.RandomWithPrefix("acc-hb-newtypes")
	dnsName := acctest.RandomWithPrefix("acc-dns-newtypes")
	udpName := acctest.RandomWithPrefix("acc-udp-newtypes")
	hbURL := testAccUniqueURL(hbName)
	dnsDomain := testAccUniqueDomain(dnsName)
	udpDomain := testAccUniqueDomain(udpName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test HEARTBEAT monitor
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "` + hbName + `"
    url          = "` + hbURL + `"
    type         = "HEARTBEAT"
    interval     = 300
    grace_period = 60
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", hbName),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HEARTBEAT"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "grace_period", "60"),
				),
			},
			// Test DNS monitor
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "` + dnsName + `"
    url          = "` + dnsDomain + `"
    type         = "DNS"
    interval     = 300
	config      = {
		dns_records = {}
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", dnsName),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "DNS"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", dnsDomain),
				),
			},
			// Test UDP monitor
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "` + udpName + `"
    url          = "` + udpDomain + `"
    type         = "UDP"
    interval     = 300
    port         = 53
    config = {
      udp = {
        payload = "ping"
        packet_loss_threshold = 50
      }
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", udpName),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "UDP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", udpDomain),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "port", "53"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "config.udp.packet_loss_threshold", "50"),
				),
			},
		},
	})
}

func TestAcc_Monitor_API_ConfigAssertions_RoundTrip(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-api-assert")
	res := "uptimerobot_monitor.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorResourceConfigWithAPIAssertions(name, "AND", "equals", `jsonencode("ok")`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "type", "API"),
					resource.TestCheckResourceAttr(res, "config.api_assertions.logic", "AND"),
					resource.TestCheckResourceAttr(res, "config.api_assertions.checks.#", "1"),
					resource.TestCheckResourceAttr(res, "config.api_assertions.checks.0.comparison", "equals"),
				),
			},
			{
				Config: testAccMonitorResourceConfigWithAPIAssertions(name, "OR", "not_equals", `jsonencode("down")`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "type", "API"),
					resource.TestCheckResourceAttr(res, "config.api_assertions.logic", "OR"),
					resource.TestCheckResourceAttr(res, "config.api_assertions.checks.#", "1"),
					resource.TestCheckResourceAttr(res, "config.api_assertions.checks.0.comparison", "not_equals"),
				),
			},
			{
				Config:             testAccMonitorResourceConfigWithAPIAssertions(name, "OR", "not_equals", `jsonencode("down")`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_API_RequiresConfigOnCreate(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-api-reqcfg")
	url := testAccUniqueURL(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = %q
  type     = "API"
  interval = 300
  timeout  = 30
}
`, name, url),
				ExpectError: regexp.MustCompile(`(?i)config.*required.*dns/api`),
			},
		},
	})
}

func TestAcc_Monitor_Config_APIAssertions_ForbiddenOnHTTP(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-apiassert-http")
	url := testAccUniqueURL(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = %q
  type     = "HTTP"
  interval = 300
  timeout  = 30

  config = {
    api_assertions = {
      logic = "AND"
      checks = [
        {
          property   = "$.status"
          comparison = "equals"
          target     = jsonencode("ok")
        }
      ]
    }
  }
}
`, name, url),
				ExpectError: regexp.MustCompile(`(?i)api_assertions[\s\S]*only[\s\S]*api monitors?`),
			},
		},
	})
}

// TestAccMonitorResource_NewFields tests the new fields added to the monitor resource.
func TestAccMonitorResource_NewFields(t *testing.T) {
	name := acctest.RandomWithPrefix("test-newfields")
	url := testAccUniqueURL(name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) threshold only
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name                    = %q
  url                     = "%s"
  type                    = "HTTP"
  interval                = 300
  timeout                 = 30
  response_time_threshold = 5000
}`, name, url),
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
  url                     = "%s"
  type                    = "HTTP"
  interval                = 300
  timeout                 = 30
  response_time_threshold = 3000
}`, name, url),
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
  url                     = "%s"
  type                    = "HTTP"
  interval                = 300
  timeout                 = 30
  response_time_threshold = 3000
  regional_data           = "eu"
}`, name, url),
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
  url                     = "%s"
  type                    = "HTTP"
  interval                = 300
  timeout                 = 30
  response_time_threshold = 3000
  regional_data           = "eu"
}`, name, url),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccMonitorResource_InvalidMonitorType tests that invalid monitor types are rejected.
func TestAccMonitorResource_InvalidMonitorType(t *testing.T) {
	name := acctest.RandomWithPrefix("test-invalid-monitor")
	url := testAccUniqueURL(name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test invalid monitor type
			{
				Config: testAccProviderConfig() + `
resource "uptimerobot_monitor" "test" {
    name         = "` + name + `"
    url          = "` + url + `"
    type         = "INVALID_TYPE"
    interval     = 300
	timeout 	 = 30
}
`,
				ExpectError: regexp.MustCompile(`(?s)value must be one of:.*HTTP.*KEYWORD.*PING.*PORT.*HEARTBEAT.*DNS.*API.*UDP`),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_UsesTimeout(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-http")
	url := testAccUniqueURL(name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  type     = "HTTP"
  url      = "%s"
  interval = 300
  timeout  = 30
}
`, name, url),
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
	name := acctest.RandomWithPrefix("acc-http-no-timeout")
	url := testAccUniqueURL(name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  type     = "HTTP"
  url      = "%s"
  interval = 300
  // timeout omitted on purpose
}
`, name, url),
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

func TestAcc_Monitor_DNS_IgnoreTimeoutAndGrace_And_PING_UsesTimeout(t *testing.T) {
	dnsName := "acc-dns"
	dnsDomain := testAccUniqueDomain(dnsName)
	pingName := "acc-ping"
	pingURL := testAccUniqueURL(pingName)
	pingConfig := testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "ping" {
  name     = %q
  type     = "PING"
  url      = %q
  interval = 300
}
	`, pingName, pingURL)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// DNS with neither timeout nor grace_period
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "dns" {
  name     = %q
  type     = "DNS"
  url      = %q
  interval = 300
  config  = {
	dns_records = {}
  }
}
`, dnsName, dnsDomain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.dns", "type", "DNS"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.dns", "timeout"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.dns", "grace_period"),
				),
			},
			{
				// PING defaults timeout to 30 and never uses grace_period
				Config: pingConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.ping", "type", "PING"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.ping", "timeout", "30"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.ping", "grace_period"),
				),
			},
			{
				// Ensure no drift after default timeout is materialized by API.
				Config:             pingConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_Heartbeat_UsesGrace(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-heartbeat")
	url := testAccUniqueURL(name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "uptimerobot_monitor" "hb" {
  name         = %q
  type         = "HEARTBEAT"
  url          = "%s"
  interval     = 300
  grace_period = 120
}
`, name, url),
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
	baseName := acctest.RandomWithPrefix("hb-bounds")
	baseURL := testAccUniqueURL(baseName)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create both boundaries in one apply to avoid update-timing flakes.
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "hb_min" {
  name         = "%s-min"
  type         = "HEARTBEAT"
  url          = "%s/min"
  interval     = 300
  grace_period = 0
}

resource "uptimerobot_monitor" "hb_max" {
  name         = "%s-max"
  type         = "HEARTBEAT"
  url          = "%s/max"
  interval     = 300
  grace_period = 86400
}
`, baseName, baseURL, baseName, baseURL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.hb_min", "grace_period", "0"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.hb_max", "grace_period", "86400"),
				),
			},
		},
	})
}

func TestAcc_Monitor_Heartbeat_Grace_Invalid(t *testing.T) {
	baseName := acctest.RandomWithPrefix("hb-bad")
	baseURL := testAccUniqueURL(baseName)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "hb" {
  name         = "%s-low"
  type         = "HEARTBEAT"
  url          = "%s"
  interval     = 300
  grace_period = -1
}
`, baseName, baseURL),
				ExpectError: regexp.MustCompile(`must be between 0 and 86400`),
			},
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "hb" {
  name         = "%s-high"
  type         = "HEARTBEAT"
  url          = "%s"
  interval     = 300
  grace_period = 86401
}
`, baseName, baseURL),
				ExpectError: regexp.MustCompile(`must be between 0 and 86400`),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_PostBody_RoundTrip(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-body")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create with body
			{
				Config: testAccMonitorResourceConfigWithBody(name, `jsonencode({foo="bar", n=1})`),
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
				Config: testAccMonitorResourceConfigWithBody(name, `jsonencode({foo="baz", m=2})`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "post_value_type", "RAW_JSON"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor.test", "post_value_data"),
				),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_PostBody_ClearByRemoving(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-body-clear")
	url := fmt.Sprintf("%s/echo", testAccUniqueURL(name))
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
  url              = "%s"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
}
`, name, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
				),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_GetHead_NoBodyAllowed(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-get-body-error")
	url := testAccUniqueURL(name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "%s"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "GET"
  post_value_data  = jsonencode({oops="nope"})
}`, name, url),
				ExpectError: regexp.MustCompile(`Request body not allowed for GET/HEAD`),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_PostKV_RoundTrip(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-kv")
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
	name := acctest.RandomWithPrefix("acc-post-nobody")
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
	name := acctest.RandomWithPrefix("acc-method-switch")
	// Keep one URL across steps to avoid backend URL normalization drift.
	url := fmt.Sprintf("%s/echo", testAccUniqueURL(name))
	postNoBodyConfig := testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = "%s"
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  http_method_type = "POST"
}
`, name, url)

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
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", url),
				),
			},
			// 2) Switch to GET with no body on the same URL to clear any payload
			{
				Config: testAccMonitorResourceConfigGetNoBodyAtURL(name, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "GET"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", url),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
				),
			},
			// 3) Switch back to POST with no body on the same URL – should remain clean and stable
			{
				Config: postNoBodyConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "POST"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", url),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_type"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_data"),
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "post_value_kv"),
				),
			},
			// 4) Idempotent re-plan on POST/no-body config
			{
				Config:             postNoBodyConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_HTTP_DefaultMethod_GET(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-default-method")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorResourceConfig(name),
				Check:  resource.TestCheckResourceAttr("uptimerobot_monitor.test", "http_method_type", "GET"),
			},
		},
	})
}

func TestAcc_Monitor_HTTP_Body_Switch_JSON_KV(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-switch-json-kv")
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
	name := acctest.RandomWithPrefix("acc-planonly-from-scratch")
	// This specifically exercises ModifyPlan with a null state on first create.
	// It should produce a non-empty plan and not panic / error.
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             testAccMonitorResourceConfig(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAcc_Monitor_CheckSSLErrors_DefaultFalse(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-sslerrs-default")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorResourceConfig(name),
				Check:  resource.TestCheckResourceAttr("uptimerobot_monitor.test", "check_ssl_errors", "false"),
			},
			// idempotency
			{
				Config:             testAccMonitorResourceConfig(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_CheckSSLErrors_ExplicitTrue(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-sslerrs-true")
	url := testAccUniqueURL(name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = %q
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  check_ssl_errors = true
}
`, name, url),
				Check: resource.TestCheckResourceAttr("uptimerobot_monitor.test", "check_ssl_errors", "true"),
			},
			// flip back to false
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name             = %q
  url              = %q
  type             = "HTTP"
  interval         = 300
  timeout          = 30
  check_ssl_errors = false
}
`, name, url),
				Check: resource.TestCheckResourceAttr("uptimerobot_monitor.test", "check_ssl_errors", "false"),
			},
		},
	})
}

func TestAcc_Monitor_GroupID_SetAndPlanStable(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-group-id")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorResourceConfigWithGroupID(name, 0),
				Check:  resource.TestCheckResourceAttr("uptimerobot_monitor.test", "group_id", "0"),
			},
			{
				Config:             testAccMonitorResourceConfigWithGroupID(name, 0),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_Config_SSLExpirationPeriodDays(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-ssl-period")
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
			{ // remove config block from TF management
				Config:             testAccMonitorResourceConfigWithSSLPeriod(name, nil),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_Config_SSLExpirationPeriodDays_Invalid(t *testing.T) {
	baseName := acctest.RandomWithPrefix("acc-ssl-period")
	baseURL := testAccUniqueURL(baseName)
	invalidName := acctest.RandomWithPrefix("acc-ssl-period-invalid")
	tooManyName := acctest.RandomWithPrefix("acc-ssl-period-too-many")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // out of range
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = "%s"
  type     = "HTTP"
  interval = 300
  timeout  = 30
  config = {
    ssl_expiration_period_days = [-1, 366]
  }
}
`, invalidName, baseURL),
				ExpectError: regexp.MustCompile(
					`(?s)` +
						`Attribute config\.ssl_expiration_period_days\[Value\(-1\)\] value must be between[\s\S]*0 and 365, got: -1` +
						`[\s\S]*` +
						`Attribute config\.ssl_expiration_period_days\[Value\(366\)\] value must be between[\s\S]*0 and 365, got: 366`,
				),
			},
			{ // > 10 items
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = "%s"
  type     = "HTTP"
  interval = 300
  timeout  = 30
  config = {
    ssl_expiration_period_days = [0,1,2,3,4,5,6,7,8,9,10]
  }
}
`, tooManyName, baseURL),
				ExpectError: regexp.MustCompile(
					`Attribute config\.ssl_expiration_period_days (?:set|value) must contain at most 10\s+elements(?:, got: \d+)?`,
				),
			},
		},
	})
}

func TestAcc_Monitor_NameURL_HTMLNormalization(t *testing.T) {
	testAccPreCheck(t)

	resourceName := "uptimerobot_monitor.test"
	name := fmt.Sprintf("A & B <C> %s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
	url := fmt.Sprintf("%s/health?a=1&b=2", testAccUniqueURL(name))

	cfgPlain := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  type     = "HTTP"
  url      = "%s"
  interval = 300
}
`, name, url)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfgPlain,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "url", url),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"group_id",
					"is_paused",
				},
			},
			{Config: cfgPlain, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}

func TestAcc_Monitor_Import_NameURL_HTMLNormalizationFromAPI(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}

	apiKey := os.Getenv("UPTIMEROBOT_API_KEY")
	if apiKey == "" {
		t.Skip("UPTIMEROBOT_API_KEY not set")
	}

	apiClient := client.NewClient(apiKey)
	if apiURL := os.Getenv("UPTIMEROBOT_API_URL"); apiURL != "" {
		apiClient.SetBaseURL(apiURL)
	}
	apiClient.SetUserAgent("terraform-provider-uptimerobot/acc-test")
	apiClient.AddHeader("X-Terraform-Provider", "uptimerobot/acc-test")

	// Create a monitor via API with intentionally escaped inputs to simulate
	// out-of-band creation (UI, direct API usage, other tools).
	suffix := acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	rawURL := fmt.Sprintf("%s/health?a=1&amp;b=2", testAccUniqueURL("acc-import-html-normalization-"+suffix))
	rawName := fmt.Sprintf("A &amp; B <C> %s", suffix)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	timeout := 30
	grace := 0
	method := "GET"
	var created *client.Monitor
	{
		backoff := 500 * time.Millisecond
		for attempt := 0; attempt < 4; attempt++ {
			m, err := apiClient.CreateMonitor(ctx, &client.CreateMonitorRequest{
				Type:           client.MonitorTypeHTTP,
				Name:           rawName,
				URL:            rawURL,
				Interval:       300,
				Timeout:        &timeout,
				GracePeriod:    &grace,
				HTTPMethodType: method,
				Tags:           []string{},
			})
			if err == nil {
				created = m
				break
			}

			// If the POST request timed out mid executio, the monitor might still have been created.
			// Attempt to find it before retrying create to avoid duplicates.
			if existing, findErr := apiClient.FindExistingMonitorByNameAndURL(ctx, rawName, rawURL); findErr == nil && existing != nil {
				created = existing
				break
			}

			if attempt == 3 {
				t.Fatalf("failed to create monitor via API: %v", err)
			}
			time.Sleep(backoff)
			if backoff < 4*time.Second {
				backoff *= 2
				if backoff > 4*time.Second {
					backoff = 4 * time.Second
				}
			}
		}
	}

	// The create endpoint may return before the monitor is consistently visible for GETs.
	// Wait for a few consecutive successes so ImportStateVerify doesn't race eventual consistency.
	{
		deadline := time.Now().Add(90 * time.Second)
		backoff := 500 * time.Millisecond
		okCount := 0
		for {
			if time.Now().After(deadline) {
				t.Fatalf("monitor %d not visible via API after create", created.ID)
			}
			_, err := apiClient.GetMonitor(ctx, created.ID)
			if err == nil {
				okCount++
				if okCount >= 3 {
					break
				}
				time.Sleep(1 * time.Second)
				continue
			}
			if !client.IsNotFound(err) {
				t.Fatalf("failed to fetch created monitor %d: %v", created.ID, err)
			}
			okCount = 0
			time.Sleep(backoff)
			if backoff < 5*time.Second {
				backoff *= 2
				if backoff > 5*time.Second {
					backoff = 5 * time.Second
				}
			}
		}
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_ = apiClient.DeleteMonitor(ctx, created.ID)
		_ = apiClient.WaitMonitorDeleted(ctx, created.ID, 90*time.Second)
	})

	plainName := unescapeHTML(rawName)
	plainURL := unescapeHTML(rawURL)

	resourceName := "uptimerobot_monitor.test"
	cfg := testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = %q
  type     = "HTTP"
  interval = 300
  timeout  = 30
}
`, plainName, plainURL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             cfg,
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStateId:      fmt.Sprintf("%d", created.ID),
				ImportStatePersist: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", plainName),
					resource.TestCheckResourceAttr(resourceName, "url", plainURL),
				),
			},
			{Config: cfg, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}

func TestAccMonitorResource_KeywordCaseType_Semantics(t *testing.T) {
	name := acctest.RandomWithPrefix("kct-semantics")
	url := testAccUniqueURL(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
	resource "uptimerobot_monitor" "test" {
	  name          = %q
	  url           = %q
	  type          = "KEYWORD"
	  interval      = 300
	  timeout       = 30
	  keyword_type  = "ALERT_EXISTS"
	  keyword_case_type = "CaseSensitive"
	  keyword_value = "ok"
	}
	`, name, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "keyword_case_type", "CaseSensitive"),
				),
			},
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
	resource "uptimerobot_monitor" "test" {
  name              = %q
  url               = %q
  type              = "KEYWORD"
	  interval          = 300
	  timeout           = 30
	  keyword_type      = "ALERT_EXISTS"
	  keyword_value     = "ok"
	  keyword_case_type = "CaseInsensitive"
	}
	`, name, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "keyword_case_type", "CaseInsensitive"),
				),
			},
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
	resource "uptimerobot_monitor" "test" {
	  name          = %q
	  url           = %q
	  type          = "KEYWORD"
	  interval      = 300
	  timeout       = 30
	  keyword_type  = "ALERT_EXISTS"
	  keyword_case_type = "CaseInsensitive"
	  keyword_value = "ok"
	}
	`, name, url),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// Config

func TestAcc_Monitor_Config_SSLDays_Semantics(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-ssl-config")
	res := "uptimerobot_monitor.test"
	url := testAccUniqueURL(name)

	cfgSet := `
resource "uptimerobot_monitor" "test" {
  name     = "` + name + `"
  type     = "HTTP"
  url      = "` + url + `"
  interval = 300
  timeout  = 30

  config = {
    ssl_expiration_period_days = [3, 5, 30, 69]
  }
}
`
	// Empty block present -> mirror remote into state, send nothing
	cfgPreserve := `
resource "uptimerobot_monitor" "test" {
  name     = "` + name + `"
  type     = "HTTP"
  url      = "` + url + `"
  interval = 300
  timeout  = 30

  config = {}
}
`
	// Explicit clear on server
	cfgClear := `
resource "uptimerobot_monitor" "test" {
  name     = "` + name + `"
  type     = "HTTP"
  url      = "` + url + `"
  interval = 300
  timeout  = 30

  config = {
    ssl_expiration_period_days = []
  }
}
`
	// Omit block entirely -> config is unmanaged for non-DNS types
	cfgOmit := `
resource "uptimerobot_monitor" "test" {
  name     = "` + name + `"
  type     = "HTTP"
  url      = "` + url + `"
  interval = 300
  timeout  = 30
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Set concrete days
			{
				Config: cfgSet,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.ssl_expiration_period_days.#", "4"),
					resource.TestCheckTypeSetElemAttr(res, "config.ssl_expiration_period_days.*", "3"),
					resource.TestCheckTypeSetElemAttr(res, "config.ssl_expiration_period_days.*", "5"),
					resource.TestCheckTypeSetElemAttr(res, "config.ssl_expiration_period_days.*", "30"),
					resource.TestCheckTypeSetElemAttr(res, "config.ssl_expiration_period_days.*", "69"),
				),
			},
			// Preserve with empty block: plan should be empty, state mirrors remote
			{
				Config:             cfgPreserve,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: cfgPreserve,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.ssl_expiration_period_days.#", "4"),
				),
			},
			// Clear on server
			{
				Config: cfgClear,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.ssl_expiration_period_days.#", "0"),
				),
			},
			// Omit block: preserve previously known remote SSL days
			{
				Config: cfgOmit,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.ssl_expiration_period_days.#", "0"),
				),
			},
		},
	})
}

func TestAcc_Monitor_Config_DNSRecords_Manage(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-dns-config")
	domain := testAccUniqueDomain(name)
	res := "uptimerobot_monitor.test"

	cfgAandCNAME := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300

  config = {
    dns_records = {
    	a     = ["93.184.216.34"]
    	cname = []
    }
  }
}
`, name, domain)
	cfgPreserve := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300

  config = {}
}
`, name, domain)
	cfgChange := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300

  config = {
    dns_records = {
    	a   = ["93.184.216.34"]
    	txt = ["v=spf1 include:%s ~all"]
    }
  }
}
`, name, domain, domain)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfgAandCNAME,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.dns_records.a.#", "1"),
					resource.TestCheckResourceAttr(res, "config.dns_records.cname.#", "0"),
				),
			},
			// Empty block preserve (mirror) should yield empty plan
			{
				Config:             cfgPreserve,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: cfgChange,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.dns_records.a.#", "1"),
					resource.TestCheckResourceAttr(res, "config.dns_records.cname.#", "0"),
					resource.TestCheckResourceAttr(res, "config.dns_records.txt.#", "1"),
				),
			},
		},
	})
}

func TestAcc_Monitor_Config_DNSRecords_ForbiddenOnHTTP(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-http-dns")
	url := testAccUniqueURL(name)
	cfg := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "`+name+`"
  type     = "HTTP"
  url      = "%s"
  interval = 300

  config = {
    dns_records = {
		a = ["1.2.3.4"]
    }
  }
}
`, url)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfg,
				ExpectError: regexp.MustCompile(`(?i)dns_records[\s\S]*only[\s\S]*dns monitors?`),
			},
		},
	})
}

func TestAcc_Monitor_Config_SSLDays_Validators(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-ssl-validate")
	url := testAccUniqueURL(name)

	tooMany := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "`+name+`"
  type     = "HTTP"
  url      = "%s"
  interval = 300
  timeout  = 30

  config = {
    # 11 items (max 10)
    ssl_expiration_period_days = [0,1,2,3,4,5,6,7,8,9,10]
  }
}
`, url)
	outOfRange := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "`+name+`"
  type     = "HTTP"
  url      = "%s"
  interval = 300
  timeout  = 30

  config = {
    ssl_expiration_period_days = [400]
  }
}
`, url)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      tooMany,
				ExpectError: regexp.MustCompile(`(?i)at most 10`),
			},
			{
				Config:      outOfRange,
				ExpectError: regexp.MustCompile(`(?i)[\s\S]*between\s*0\s*and\s*365`),
			},
		},
	})
}

func TestAcc_Monitor_Config_IPVersion_SetAndUpdate(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("acc-ip-version")
	url := testAccUniqueURL(name)
	res := "uptimerobot_monitor.test"

	cfgIPv4 := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "HTTP"
  url      = "%s"
  interval = 300

  config = {
    ip_version = "ipv4Only"
  }
}
`, name, url)

	cfgIPv6 := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "HTTP"
  url      = "%s"
  interval = 300

  config = {
    ip_version = "ipv6Only"
  }
}
`, name, url)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfgIPv4,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.ip_version", "ipv4Only"),
				),
			},
			{
				Config: cfgIPv6,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.ip_version", "ipv6Only"),
				),
			},
			{
				Config:             cfgIPv6,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_Config_IPVersion_SetAndUpdate_API(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("acc-api-ip-version")
	url := testAccUniqueURL(name)
	res := "uptimerobot_monitor.test"

	cfgIPv4 := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "API"
  url      = "%s"
  interval = 300
  timeout  = 30

  config = {
    ip_version = "ipv4Only"
    api_assertions = {
      logic = "AND"
      checks = [{
        property   = "$.status"
        comparison = "is_not_null"
      }]
    }
  }
}
`, name, url)

	cfgIPv6 := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "API"
  url      = "%s"
  interval = 300
  timeout  = 30

  config = {
    ip_version = "ipv6Only"
    api_assertions = {
      logic = "AND"
      checks = [{
        property   = "$.status"
        comparison = "is_not_null"
      }]
    }
  }
}
`, name, url)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfgIPv4,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "type", "API"),
					resource.TestCheckResourceAttr(res, "config.ip_version", "ipv4Only"),
				),
			},
			{
				Config: cfgIPv6,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.ip_version", "ipv6Only"),
				),
			},
			{
				Config:             cfgIPv6,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_Config_IPVersion_AllowedForPingAndPort(t *testing.T) {
	t.Parallel()

	pingName := acctest.RandomWithPrefix("acc-ping-ipv")
	portName := acctest.RandomWithPrefix("acc-port-ipv")
	pingURL := testAccUniqueURL(pingName)
	portURL := testAccUniqueURL(portName)

	cfgInitial := fmt.Sprintf(`
resource "uptimerobot_monitor" "ping" {
  name     = "%s"
  type     = "PING"
  url      = "%s"
  interval = 300

  config = {
    ip_version = "ipv4Only"
  }
}

resource "uptimerobot_monitor" "port" {
  name     = "%s"
  type     = "PORT"
  url      = "%s"
  interval = 300
  timeout  = 30
  port     = 443

  config = {
    ip_version = "ipv6Only"
  }
}
`, pingName, pingURL, portName, portURL)

	cfgSwap := fmt.Sprintf(`
resource "uptimerobot_monitor" "ping" {
  name     = "%s"
  type     = "PING"
  url      = "%s"
  interval = 300

  config = {
    ip_version = "ipv6Only"
  }
}

resource "uptimerobot_monitor" "port" {
  name     = "%s"
  type     = "PORT"
  url      = "%s"
  interval = 300
  timeout  = 30
  port     = 443

  config = {
    ip_version = "ipv4Only"
  }
}
`, pingName, pingURL, portName, portURL)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfgInitial,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.ping", "config.ip_version", "ipv4Only"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.port", "config.ip_version", "ipv6Only"),
				),
			},
			{
				Config: cfgSwap,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.ping", "config.ip_version", "ipv6Only"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.port", "config.ip_version", "ipv4Only"),
				),
			},
		},
	})
}

func TestAcc_Monitor_Config_IPVersion_AllowsAPI_RejectsUDP(t *testing.T) {
	t.Parallel()

	apiName := acctest.RandomWithPrefix("acc-api-ipv")
	udpName := acctest.RandomWithPrefix("acc-udp-ipv")
	apiURL := testAccUniqueURL(apiName)
	udpURL := testAccUniqueURL(udpName)

	cfgAPIAllowed := fmt.Sprintf(`
resource "uptimerobot_monitor" "api" {
  name     = "%s"
  type     = "API"
  url      = "%s"
  interval = 300
  timeout  = 30

  config = {
    ip_version = "ipv4Only"
    api_assertions = {
      logic = "AND"
      checks = [{
        property   = "$.status"
        comparison = "is_not_null"
      }]
    }
  }
}
`, apiName, apiURL)

	cfgUnsupportedRejected := fmt.Sprintf(`
resource "uptimerobot_monitor" "udp" {
  name     = "%s"
  type     = "UDP"
  url      = "%s"
  port     = 53
  interval = 300

  config = {
    ip_version = "ipv4Only"
    udp = {
      packet_loss_threshold = 100
    }
  }
}
`, udpName, udpURL)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfgUnsupportedRejected,
				ExpectError: regexp.MustCompile(`(?i)ip_version[\s\S]*only[\s\S]*HTTP/KEYWORD/PING/PORT/API`),
			},
			{
				Config: cfgAPIAllowed,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.api", "config.ip_version", "ipv4Only"),
				),
			},
		},
	})
}

func TestAcc_Monitor_Config_IPVersion_Validators(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("acc-ip-version-validate")
	httpURL := testAccUniqueURL(name)
	dnsDomain := testAccUniqueDomain(name)

	cfgInvalidForDNS := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300

  config = {
    dns_records = {}
    ip_version  = "ipv4Only"
  }
}
`, name, dnsDomain)

	cfgMismatchIPv4 := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "HTTP"
  url      = "https://[2606:4700:4700::1111]/health"
  interval = 300

  config = {
    ip_version = "ipv4Only"
  }
}
`, name)

	cfgMismatchIPv6 := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "HTTP"
  url      = "%s"
  interval = 300

  config = {
    ip_version = "ipv6Only"
  }
}
`, name, strings.Replace(httpURL, "example.com", "1.1.1.1", 1))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfgInvalidForDNS,
				ExpectError: regexp.MustCompile(`(?i)ip_version[\s\S]*only[\s\S]*HTTP/KEYWORD/PING/PORT/API`),
			},
			{
				Config:      cfgMismatchIPv4,
				ExpectError: regexp.MustCompile(`(?i)incompatible ip_version[\s\S]*ipv4[\s\S]*IPv6`),
			},
			{
				Config:      cfgMismatchIPv6,
				ExpectError: regexp.MustCompile(`(?i)incompatible ip_version[\s\S]*ipv6[\s\S]*IPv4`),
			},
		},
	})
}

func TestAcc_Monitor_Config_DNSRecords_EmptyList_StaysEmpty(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-dns-empty")
	domain := testAccUniqueDomain(name)
	res := "uptimerobot_monitor.test"

	cfgEmpty := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300

  config = {
    dns_records = {
      cname = []   # explicitly managed empty list
    }
  }
}
`, name, domain)
	cfgNonEmpty := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300

  config = {
    dns_records = {
      cname = ["foo.%s."]
    }
  }
}
`, name, domain, domain)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with empty list -> should be an empty set in state, not null
			{
				Config: cfgEmpty,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.dns_records.cname.#", "0"),
				),
			},
			// Re-apply same config -> empty plan and still empty set
			{
				Config:             cfgEmpty,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Flip to non-empty and back to [] to ensure it remains empty set (not null)
			{
				Config: cfgNonEmpty,
				Check:  resource.TestCheckResourceAttr(res, "config.dns_records.cname.#", "1"),
			},
			{
				Config: cfgEmpty,
				Check:  resource.TestCheckResourceAttr(res, "config.dns_records.cname.#", "0"),
			},
		},
	})
}

func TestAcc_Monitor_Config_DNSRecords_EmptyList_A_StaysEmpty(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-dns-empty-a")
	domain := testAccUniqueDomain(name)
	res := "uptimerobot_monitor.test"

	cfgEmpty := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300

  config = {
    dns_records = {
      a     = []              # explicitly managed empty list for A
      cname = ["foo.%s."]     # keep a real record so the monitor is fully valid
    }
  }
}
`, name, domain, domain)

	cfgNonEmpty := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300

  config = {
    dns_records = {
      a     = ["93.184.216.34"]
      cname = ["foo.%s."]
    }
  }
}
`, name, domain, domain)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Start with a = []
			{
				Config: cfgEmpty,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.dns_records.a.#", "0"),
					resource.TestCheckResourceAttr(res, "config.dns_records.cname.#", "1"),
				),
			},
			// Re-apply same config, have to be empty
			{
				Config:             cfgEmpty,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Make A filled
			{
				Config: cfgNonEmpty,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.dns_records.a.#", "1"),
					resource.TestCheckResourceAttr(res, "config.dns_records.cname.#", "1"),
				),
			},
			// Back to empty. Have to go back to an empty set
			{
				Config: cfgEmpty,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.dns_records.a.#", "0"),
					resource.TestCheckResourceAttr(res, "config.dns_records.cname.#", "1"),
				),
			},
			{
				Config:             cfgEmpty,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_Config_DNSRecords_OmitConfig_Preserves(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-dns-omit-config")
	domain := testAccUniqueDomain(name)
	res := "uptimerobot_monitor.test"

	cfgWithRecords := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300

  config = {
    dns_records = {
      a   = ["93.184.216.34"]
      txt = ["v=spf1 include:%s ~all"]
    }
  }
}
`, name, domain, domain)

	cfgOmitConfig := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s-updated"
  type     = "DNS"
  url      = "%s"
  interval = 300
  # config omitted on purpose to preserve remote dns_records
}
`, name, domain)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create with explicit dns_records
			{
				Config: cfgWithRecords,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config.dns_records.a.#", "1"),
					resource.TestCheckResourceAttr(res, "config.dns_records.txt.#", "1"),
				),
			},
			// 2) Update without config, so server have to keep dns_records
			{
				Config: cfgOmitConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "name", name+"-updated"),
					resource.TestCheckResourceAttr(res, "config.dns_records.a.#", "1"),
					resource.TestCheckResourceAttr(res, "config.dns_records.txt.#", "1"),
				),
			},
			// 3) Re-plan same config to check for no drift
			{
				Config:             cfgOmitConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_Config_DNSRecords_ConfigWithoutRecords_AllowsEmptyConfig(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-dns-norecords")
	domain := testAccUniqueDomain(name)
	res := "uptimerobot_monitor.test"

	cfgMissing := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300

  # config block present but dns_records omitted on purpose
  config = {}
}
`, name, domain)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfgMissing,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "name", name),
				),
			},
			{
				Config:             cfgMissing,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAcc_Monitor_Config_DNSRequiresConfigOnCreate(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-dns-requires-config")
	domain := testAccUniqueDomain(name)

	cfgMissingConfig := fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
  name     = "%s"
  type     = "DNS"
  url      = "%s"
  interval = 300
}
`, name, domain)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfgMissingConfig,
				ExpectError: regexp.MustCompile("config.*required.*DNS/API/UDP monitors.*create"),
			},
		},
	})
}
