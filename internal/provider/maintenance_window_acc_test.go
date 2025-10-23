package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccMWWeeklyWithDaysCfg(name string, days string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "test" {
  name     = %q
  interval = "weekly"
  time     = "02:00:00"
  duration = 60
  days     = %s
}
`, name, days)
}

func testAccMWMonthlyCfg(name string, days string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "test" {
  name     = %q
  interval = "monthly"
  time     = "03:30:00"
  duration = 90
  days     = %s
}
`, name, days)
}

func testAccMWDailyWithDaysInvalidCfg(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "uptimerobot_maintenance_window" "test" {
  name     = %q
  interval = "daily"
  time     = "01:00:00"
  duration = 30
  days     = [3] // invalid: days not allowed for daily
}
`, name)
}

func TestAccMaintenanceWindow_WeeklyDays(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMaintenanceWindowDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMWWeeklyWithDaysCfg("mw-weekly", "[2,4,5]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "interval", "weekly"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_maintenance_window.test", "days.*", "2"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_maintenance_window.test", "days.*", "4"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_maintenance_window.test", "days.*", "5"),
				),
			},
			{
				// Update and remove Friday
				Config: testAccMWWeeklyWithDaysCfg("mw-weekly", "[2,4]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "interval", "weekly"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "days.#", "2"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_maintenance_window.test", "days.*", "2"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_maintenance_window.test", "days.*", "4"),
				),
			},
			{
				ResourceName:      "uptimerobot_maintenance_window.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccMaintenanceWindow_MonthlyWithLastDay(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMaintenanceWindowDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMWMonthlyCfg("mw-monthly-last", "[-1]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "interval", "monthly"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.test", "days.#", "1"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_maintenance_window.test", "days.*", "-1"),
				),
			},
			{
				ResourceName:      "uptimerobot_maintenance_window.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccMaintenanceWindow_DailyWithDays_ShouldError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccMWDailyWithDaysInvalidCfg("mw-daily-bad"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Days not allowed for this interval|only valid for interval = "weekly" or "monthly"`),
			},
		},
	})
}

func TestAccMaintenanceWindow_DriftSuppression_DailyOmitDays(t *testing.T) {
	cfgWeekly := testAccProviderConfig() + `
resource "uptimerobot_maintenance_window" "mw" {
  name     = "mw-drift"
  interval = "weekly"
  time     = "02:00:00"
  duration = 60
  days     = [2,4,5]
}
`
	cfgDaily := testAccProviderConfig() + `
resource "uptimerobot_maintenance_window" "mw" {
  name     = "mw-drift"
  interval = "daily"
  time     = "02:00:00"
  duration = 60
  // days intentionally omitted
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMaintenanceWindowDestroy,
		Steps: []resource.TestStep{
			{Config: cfgWeekly},
			{Config: cfgDaily},
			{
				Config:             cfgDaily,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false, // ensure no diff
			},
		},
	})
}

func TestAccMaintenanceWindow_WeeklyDays_DedupAndOrderIrrelevant(t *testing.T) {
	cfg := testAccProviderConfig() + `
resource "uptimerobot_maintenance_window" "mw" {
  name     = "mw-dedup"
  interval = "weekly"
  time     = "03:00:00"
  duration = 45
  days     = [4,2,2,7]
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMaintenanceWindowDestroy,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.mw", "interval", "weekly"),
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.mw", "days.#", "3"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_maintenance_window.mw", "days.*", "2"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_maintenance_window.mw", "days.*", "4"),
					resource.TestCheckTypeSetElemAttr("uptimerobot_maintenance_window.mw", "days.*", "7"),
				),
			},
		},
	})
}

func TestAccMaintenanceWindow_WeeklyEmptyDays_ShouldError(t *testing.T) {
	cfg := testAccProviderConfig() + `
resource "uptimerobot_maintenance_window" "mw" {
  name     = "mw-weekly-empty"
  interval = "weekly"
  time     = "01:00:00"
  duration = 30
  days     = []
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfg,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Days cannot be empty|must set at least one`),
			},
		},
	})
}

func TestAccMaintenanceWindow_MonthlyInvalidDay_ShouldError(t *testing.T) {
	cfg := testAccProviderConfig() + `
resource "uptimerobot_maintenance_window" "mw" {
  name     = "mw-monthly-bad"
  interval = "monthly"
  time     = "01:00:00"
  duration = 30
  days     = [32] // invalid
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfg,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must be between -1 and 31`),
			},
		},
	})
}

func TestAccMaintenanceWindow_AutoAddMonitors_NullAndSet(t *testing.T) {
	cfgNull := testAccProviderConfig() + `
resource "uptimerobot_maintenance_window" "mw" {
  name     = "mw-auto-null"
  interval = "weekly"
  time     = "04:00:00"
  duration = 20
  days     = [2]
  // auto_add_monitors omitted on purpose
}
`
	cfgSet := testAccProviderConfig() + `
resource "uptimerobot_maintenance_window" "mw" {
  name     = "mw-auto-null"
  interval = "weekly"
  time     = "04:00:00"
  duration = 20
  days     = [2]
  auto_add_monitors = true
}
`
	cfgOmitAgain := testAccProviderConfig() + `
resource "uptimerobot_maintenance_window" "mw" {
  name     = "mw-auto-null"
  interval = "weekly"
  time     = "04:00:00"
  duration = 20
  days     = [2]
  // omit again; state should keep true (explicit > implicit)
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck() },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMaintenanceWindowDestroy,
		Steps: []resource.TestStep{
			{
				Config: cfgNull,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.mw", "auto_add_monitors", "false"),
				),
			},
			{
				Config: cfgSet,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptimerobot_maintenance_window.mw", "auto_add_monitors", "true"),
				),
			},
			{
				Config:             cfgOmitAgain,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
