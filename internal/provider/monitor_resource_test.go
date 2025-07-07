package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccMonitorResourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor" "test" {
    name         = %q
    url          = "https://example.com"
    type         = "1"
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
    type         = "1"
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
    type         = "1"
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
    type         = "1"
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
    type         = "1"
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
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "1"),
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
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "1"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "url", "https://example.com"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "interval", "300"),
					// Verify no alert contacts are set initially
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts"),
				),
			},
			// Step 2: Add alert contacts to existing monitor - this should NOT fail
			{
				Config: testAccMonitorResourceConfigWithAlertContacts("test-monitor-alerts", []string{"contact-123", "contact-456"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "name", "test-monitor-alerts"),
					// Verify alert contacts are now set
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts.#", "2"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts.0", "contact-123"),
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "assigned_alert_contacts.1", "contact-456"),
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
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "1"),
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
					resource.TestCheckNoResourceAttr("uptimerobot_monitor.test", "tags"),
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
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "1"),
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
					resource.TestCheckResourceAttr("uptimerobot_monitor.test", "type", "1"),
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
