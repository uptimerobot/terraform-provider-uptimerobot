//go:build acceptance

package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func testAccMonitorGroupConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor_group" "test" {
  name = %q
}
`, name)
}

func testAccMonitorGroupWithMonitorConfig(groupName, monitorName string) string {
	url := testAccUniqueURL(monitorName)
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor_group" "test" {
  name = %q
}

resource "uptimerobot_monitor" "test" {
  name     = %q
  url      = %q
  type     = "HTTP"
  interval = 300
  timeout  = 30
  group_id = tonumber(uptimerobot_monitor_group.test.id)
}
`, groupName, monitorName, url)
}

func testAccMonitorGroupDeleteMoveConfig(sourceName, fallbackName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor_group" "fallback" {
  name = %q
}

resource "uptimerobot_monitor_group" "source" {
  name                  = %q
  monitors_new_group_id = tonumber(uptimerobot_monitor_group.fallback.id)
}
`, fallbackName, sourceName)
}

func testAccMonitorGroupFallbackOnlyConfig(fallbackName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_monitor_group" "fallback" {
  name = %q
}
`, fallbackName)
}

func TestAccMonitorGroup_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-monitor-group")
	renamed := name + "-renamed"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorGroupConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptimerobot_monitor_group.test", "id"),
					resource.TestCheckResourceAttr("uptimerobot_monitor_group.test", "name", name),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor_group.test", "created_at"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor_group.test", "updated_at"),
				),
			},
			{
				Config: testAccMonitorGroupConfig(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor_group.test", "name", renamed),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor_group.test", "created_at"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor_group.test", "updated_at"),
				),
			},
			{
				Config:             testAccMonitorGroupConfig(renamed),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "uptimerobot_monitor_group.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The API may briefly return stale values immediately after rename.
				ImportStateVerifyIgnore: []string{"name", "updated_at"},
			},
		},
	})
}

func TestAccMonitorGroup_DeleteMovesExternalMonitorToConfiguredGroup(t *testing.T) {
	sourceName := acctest.RandomWithPrefix("acc-monitor-group-source")
	fallbackName := acctest.RandomWithPrefix("acc-monitor-group-fallback")
	monitorName := acctest.RandomWithPrefix("acc-monitor-group-move")
	var monitorID int64

	cleanupMonitor := func() {
		if monitorID == 0 {
			return
		}
		apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		_ = apiClient.DeleteMonitor(ctx, monitorID)
		_ = apiClient.WaitMonitorDeleted(ctx, monitorID, 90*time.Second)
		monitorID = 0
	}
	t.Cleanup(cleanupMonitor)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorGroupDeleteMoveConfig(sourceName, fallbackName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptimerobot_monitor_group.fallback", "id"),
					resource.TestCheckResourceAttrSet("uptimerobot_monitor_group.source", "id"),
					testAccCreateExternalMonitorInGroup(&monitorID, monitorName, "uptimerobot_monitor_group.source"),
				),
			},
			{
				Config: testAccMonitorGroupFallbackOnlyConfig(fallbackName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("uptimerobot_monitor_group.fallback", "id"),
					testAccCheckExternalMonitorMovedAndCleanup(&monitorID, "uptimerobot_monitor_group.fallback"),
				),
			},
		},
	})
}

func testAccCreateExternalMonitorInGroup(monitorID *int64, name, groupResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *monitorID != 0 {
			return nil
		}

		groupID, err := testAccStateResourceIntID(s, groupResource)
		if err != nil {
			return err
		}
		groupIDInt := int(groupID)
		timeout := 30

		apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		monitor, err := apiClient.CreateMonitor(ctx, &client.CreateMonitorRequest{
			Name:                     name,
			URL:                      testAccUniqueURL(name),
			Type:                     client.MonitorTypeHTTP,
			Interval:                 300,
			Timeout:                  &timeout,
			Tags:                     []string{},
			SSLExpirationReminder:    false,
			DomainExpirationReminder: false,
			FollowRedirections:       false,
			GroupID:                  &groupIDInt,
		})
		if err != nil {
			return fmt.Errorf("creating external monitor in source group: %w", err)
		}
		*monitorID = monitor.ID

		if monitor.GroupID != groupID {
			return fmt.Errorf("external monitor group_id = %d, want %d", monitor.GroupID, groupID)
		}
		return nil
	}
}

func testAccCheckExternalMonitorMovedAndCleanup(monitorID *int64, fallbackResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *monitorID == 0 {
			return fmt.Errorf("external monitor was not created")
		}

		fallbackID, err := testAccStateResourceIntID(s, fallbackResource)
		if err != nil {
			return err
		}

		apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		var monitor *client.Monitor
		for i := 0; i < 20; i++ {
			monitor, err = apiClient.GetMonitor(ctx, *monitorID)
			if err != nil {
				return fmt.Errorf("reading external monitor after source group delete: %w", err)
			}
			if monitor.GroupID == fallbackID {
				break
			}
			time.Sleep(2 * time.Second)
		}
		if monitor == nil || monitor.GroupID != fallbackID {
			gotGroupID := int64(0)
			if monitor != nil {
				gotGroupID = monitor.GroupID
			}
			return fmt.Errorf("external monitor group_id = %d, want fallback group %d", gotGroupID, fallbackID)
		}

		if err := apiClient.DeleteMonitor(ctx, *monitorID); err != nil {
			return fmt.Errorf("cleaning up external monitor: %w", err)
		}
		if err := apiClient.WaitMonitorDeleted(ctx, *monitorID, 90*time.Second); err != nil {
			return fmt.Errorf("waiting for external monitor cleanup: %w", err)
		}
		*monitorID = 0
		return nil
	}
}

func testAccStateResourceIntID(s *terraform.State, resourceName string) (int64, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return 0, fmt.Errorf("resource %s not found in state", resourceName)
	}
	id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing %s ID %q: %w", resourceName, rs.Primary.ID, err)
	}
	return id, nil
}

func TestAccMonitorGroup_WithMonitorAssignment(t *testing.T) {
	groupName := acctest.RandomWithPrefix("acc-monitor-group")
	monitorName := acctest.RandomWithPrefix("acc-monitor-group-monitor")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMonitorGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorGroupWithMonitorConfig(groupName, monitorName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_monitor_group.test", "name", groupName),
					resource.TestCheckResourceAttrPair(
						"uptimerobot_monitor.test",
						"group_id",
						"uptimerobot_monitor_group.test",
						"id",
					),
				),
			},
			{
				Config:             testAccMonitorGroupWithMonitorConfig(groupName, monitorName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
