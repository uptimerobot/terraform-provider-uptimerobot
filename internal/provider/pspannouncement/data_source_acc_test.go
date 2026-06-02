//go:build acceptance

package pspannouncement_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func testAccPSPAnnouncementDataSourceResourceConfig(name string) string {
	return provideracctest.ProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_psp" "test" {
  name         = %q
  subscription = true
}

resource "uptimerobot_psp_announcement" "test" {
  psp_id     = tonumber(uptimerobot_psp.test.id)
  title      = "Data source maintenance"
  content    = "We will perform scheduled maintenance."
  status     = "pending"
  type       = "maintenance"
  start_date = "2030-01-01T00:00:00Z"
  end_date   = "2030-01-01T01:00:00Z"
}
`, name)
}

func testAccPSPAnnouncementDataSourceConfig(name string) string {
	return testAccPSPAnnouncementDataSourceResourceConfig(name) + `
data "uptimerobot_psp_announcement" "by_id" {
  psp_id = tonumber(uptimerobot_psp.test.id)
  id     = uptimerobot_psp_announcement.test.id

  depends_on = [uptimerobot_psp_announcement.test]
}

data "uptimerobot_psp_announcement" "by_title" {
  psp_id = tonumber(uptimerobot_psp.test.id)
  title  = uptimerobot_psp_announcement.test.title

  depends_on = [uptimerobot_psp_announcement.test]
}
`
}

func testAccCheckPSPAnnouncementVisibleInList(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		pspID, err := testAccParseInt64Attr(rs.Primary.Attributes, "psp_id")
		if err != nil {
			return err
		}
		announcementID, err := testAccParseInt64Value(rs.Primary.ID, "announcement ID")
		if err != nil {
			return err
		}

		apiClient := provideracctest.APIClient()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var lastListErr error
		for {
			announcements, err := apiClient.ListAllPSPAnnouncements(ctx, pspID)
			if err != nil {
				lastListErr = err
			} else {
				for _, announcement := range announcements {
					if announcement.ID == announcementID {
						return nil
					}
				}
			}

			select {
			case <-ctx.Done():
				if lastListErr != nil {
					return fmt.Errorf("PSP announcement %d was not visible in list endpoint before ctx.Done; last apiClient.ListAllPSPAnnouncements error: %v: %w", announcementID, lastListErr, ctx.Err())
				}
				return fmt.Errorf("PSP announcement %d was not visible in list endpoint before timeout: %w", announcementID, ctx.Err())
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func testAccParseInt64Attr(attrs map[string]string, key string) (int64, error) {
	value := attrs[key]
	if value == "" {
		return 0, fmt.Errorf("missing %s in state", key)
	}
	return testAccParseInt64Value(value, key)
}

func testAccParseInt64Value(value, label string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse %s %q: %w", label, value, err)
	}
	return parsed, nil
}

func TestAccPSPAnnouncementDataSource(t *testing.T) {
	if os.Getenv("UPTIMEROBOT_TEST_PSP_ANNOUNCEMENT") != "1" {
		t.Skip("set UPTIMEROBOT_TEST_PSP_ANNOUNCEMENT=1 to run PSP announcement acceptance tests")
	}

	name := provideracctest.RandomName("acc-psp-ann-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		CheckDestroy:             provideracctest.CheckPSPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPSPAnnouncementDataSourceResourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_psp_announcement.test", "title", "Data source maintenance"),
					testAccCheckPSPAnnouncementVisibleInList("uptimerobot_psp_announcement.test"),
				),
			},
			{
				Config: testAccPSPAnnouncementDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_psp_announcement.by_id",
						"id",
						"uptimerobot_psp_announcement.test",
						"id",
					),
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_psp_announcement.by_title",
						"id",
						"uptimerobot_psp_announcement.test",
						"id",
					),
					resource.TestCheckResourceAttrPair(
						"data.uptimerobot_psp_announcement.by_title",
						"psp_id",
						"uptimerobot_psp_announcement.test",
						"psp_id",
					),
					resource.TestCheckResourceAttr("data.uptimerobot_psp_announcement.by_title", "title", "Data source maintenance"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp_announcement.by_title", "content", "We will perform scheduled maintenance."),
					resource.TestCheckResourceAttr("data.uptimerobot_psp_announcement.by_title", "status", "pending"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp_announcement.by_title", "type", "maintenance"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp_announcement.by_title", "start_date", "2030-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp_announcement.by_title", "end_date", "2030-01-01T01:00:00Z"),
					resource.TestCheckResourceAttr("data.uptimerobot_psp_announcement.by_title", "is_pinned", "false"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_psp_announcement.by_title", "creation_date"),
				),
			},
		},
	})
}
