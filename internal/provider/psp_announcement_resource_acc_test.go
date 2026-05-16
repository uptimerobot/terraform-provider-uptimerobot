//go:build acceptance

package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPSPAnnouncementResource(t *testing.T) {
	if os.Getenv("UPTIMEROBOT_TEST_PSP_ANNOUNCEMENT") != "1" {
		t.Skip("set UPTIMEROBOT_TEST_PSP_ANNOUNCEMENT=1 to run PSP announcement acceptance tests")
	}

	name := randomName("acc-psp-ann")
	startDate := "2030-01-01T00:00:00Z"
	endDate := "2030-01-01T01:00:00Z"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPSPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPSPAnnouncementResourceConfig(
					name,
					"Scheduled maintenance",
					"We will perform scheduled maintenance.",
					"pending",
					"maintenance",
					startDate,
					&endDate,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "title", "Scheduled maintenance"),
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "content", "We will perform scheduled maintenance."),
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "status", "pending"),
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "type", "maintenance"),
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "start_date", startDate),
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "end_date", endDate),
					resource.TestCheckResourceAttrSet("uptimerobot_psp_announcement.test", "creation_date"),
				),
			},
			{
				Config: testAccPSPAnnouncementResourceConfig(
					name,
					"Issue update",
					"We are investigating a service issue.",
					"offline",
					"issue",
					"2030-01-01T02:00:00Z",
					nil,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "title", "Issue update"),
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "content", "We are investigating a service issue."),
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "status", "offline"),
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "type", "issue"),
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "start_date", "2030-01-01T02:00:00Z"),
					resource.TestCheckNoResourceAttr("uptimerobot_psp_announcement.test", "end_date"),
				),
			},
			{
				ResourceName:      "uptimerobot_psp_announcement.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccPSPAnnouncementImportStateID,
			},
		},
	})
}

func TestAccPSPAnnouncementResource_PinUnpin(t *testing.T) {
	if os.Getenv("UPTIMEROBOT_TEST_PSP_ANNOUNCEMENT") != "1" {
		t.Skip("set UPTIMEROBOT_TEST_PSP_ANNOUNCEMENT=1 to run PSP announcement acceptance tests")
	}

	name := randomName("acc-psp-ann-pin")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPSPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPSPAnnouncementResourcePinConfig(name, pspAnnouncementBoolPtr(true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "is_pinned", "true"),
					testAccCheckPSPAnnouncementPinned("uptimerobot_psp_announcement.test", true),
				),
			},
			{
				Config: testAccPSPAnnouncementResourcePinConfig(name, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("uptimerobot_psp_announcement.test", "is_pinned"),
					testAccCheckPSPAnnouncementPinned("uptimerobot_psp_announcement.test", true),
				),
			},
			{
				Config: testAccPSPAnnouncementResourcePinConfig(name, pspAnnouncementBoolPtr(false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "is_pinned", "false"),
					testAccCheckPSPAnnouncementPinned("uptimerobot_psp_announcement.test", false),
				),
			},
			{
				Config: testAccPSPAnnouncementResourcePinConfig(name, pspAnnouncementBoolPtr(true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "is_pinned", "true"),
					testAccCheckPSPAnnouncementPinned("uptimerobot_psp_announcement.test", true),
				),
			},
		},
	})
}

func testAccPSPAnnouncementResourceConfig(
	name string,
	title string,
	content string,
	status string,
	announcementType string,
	startDate string,
	endDate *string,
) string {
	endDateConfig := ""
	if endDate != nil {
		endDateConfig = fmt.Sprintf("\n  end_date = %q", *endDate)
	}

	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name         = %q
  subscription = true
}

resource "uptimerobot_psp_announcement" "test" {
  psp_id     = tonumber(uptimerobot_psp.test.id)
  title      = %q
  content    = %q
  status     = %q
  type       = %q
  start_date = %q%s
}
`, name, title, content, status, announcementType, startDate, endDateConfig)
}

func testAccPSPAnnouncementResourcePinConfig(name string, pinned *bool) string {
	pinnedConfig := ""
	if pinned != nil {
		pinnedConfig = fmt.Sprintf("\n  is_pinned  = %t", *pinned)
	}

	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name         = %q
  subscription = true
}

resource "uptimerobot_psp_announcement" "test" {
  psp_id     = tonumber(uptimerobot_psp.test.id)
  title      = "Pinned maintenance"
  content    = "We will perform scheduled maintenance."
  status     = "pending"
  type       = "maintenance"
  start_date = "2030-01-01T00:00:00Z"%s
}
`, name, pinnedConfig)
}

func testAccPSPAnnouncementImportStateID(s *terraform.State) (string, error) {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_psp_announcement" {
			continue
		}
		pspID := rs.Primary.Attributes["psp_id"]
		if pspID == "" {
			return "", fmt.Errorf("missing psp_id in state")
		}
		return fmt.Sprintf("%s:%s", pspID, rs.Primary.ID), nil
	}
	return "", fmt.Errorf("uptimerobot_psp_announcement.test not found in state")
}

func testAccCheckPSPAnnouncementPinned(resourceName string, expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		pspID, err := strconv.ParseInt(rs.Primary.Attributes["psp_id"], 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse psp_id %q: %w", rs.Primary.Attributes["psp_id"], err)
		}
		announcementID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse announcement ID %q: %w", rs.Primary.ID, err)
		}

		apiClient := testAccAPIClient()
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		var lastPinnedID *int64
		for {
			psp, err := apiClient.GetPSP(ctx, pspID)
			if err != nil {
				return fmt.Errorf("could not read PSP %d: %w", pspID, err)
			}

			lastPinnedID = psp.PinnedAnnouncementID
			got := psp.PinnedAnnouncementID != nil && *psp.PinnedAnnouncementID == announcementID
			if got == expected {
				return nil
			}

			select {
			case <-ctx.Done():
				if lastPinnedID == nil {
					return fmt.Errorf("expected is_pinned=%t, got nil pinned_announcement_id", expected)
				}
				return fmt.Errorf("expected is_pinned=%t for announcement %d, got pinned_announcement_id=%d", expected, announcementID, *lastPinnedID)
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func pspAnnouncementBoolPtr(value bool) *bool {
	return &value
}
