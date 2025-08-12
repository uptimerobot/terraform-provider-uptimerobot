package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccMonitorResourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = "https://example.com"
    type         = "HTTP"
    interval     = 300
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
}
`, name, maintenanceWindowsStr)
}

func testAccMonitorResourceConfigWithSuccessHTTPResponseCodes(name string, responseCodes []string) string {
	responseCodesStr := ""
	if len(responseCodes) > 0 {
		responseCodesStr = fmt.Sprintf(`
    success_http_response_codes = [%s]`, joinQuotedStrings(responseCodes))
	}

	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = "https://example.com"
    type         = "HTTP"
    interval     = 300%s
}
`, name, responseCodesStr)
}

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
				ResourceName:      "uptimerobot_monitor.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccMonitorResource_AlertContacts tests the specific case where alert contacts
// are added to an existing monitor that was initially created without any.
func TestAccMonitorResource_AlertContacts(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create monitor without alert contacts
			{
				Config: testAccMonitorResourceConfigWithAlertContacts("test-monitor-alerts", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-alerts"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", "https://example.com"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "interval", "300"),
					// Verify no alert contacts are set initially
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts"),
				),
			},
			// Step 2: Add alert contacts to existing monitor - this should NOT fail
			{
				Config: testAccMonitorResourceConfigWithAlertContacts("test-monitor-alerts", []string{"7589476"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-alerts"),
					// Verify alert contacts are now set
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts.#", "1"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts.0", "7589476"),
				),
			},
			// Step 3: Remove alert contacts - set back to empty
			{
				Config: testAccMonitorResourceConfigWithAlertContacts("test-monitor-alerts", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-alerts"),
					// Verify alert contacts are removed
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
			// Step 3: Set to empty list (should revert to defaults?)
			{
				Config: testAccMonitorResourceConfigWithSuccessHTTPResponseCodes("test-monitor-response-codes", []string{}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-response-codes"),
					// This might expose the issue - what happens with empty vs default?
				),
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
}
`,
				ExpectError: regexp.MustCompile(`(?s)value must be one of:.*HTTP.*KEYWORD.*PING.*PORT.*HEARTBEAT.*DNS`),
			},
		},
	})
}
