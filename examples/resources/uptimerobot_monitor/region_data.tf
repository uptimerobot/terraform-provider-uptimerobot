# Requires an account with monitor-location-settings support.
# Accounts without that feature cannot use region_data.
resource "uptimerobot_monitor" "multi_region" {
  name     = "Multi-region Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  timeout  = 30

  region_data = {
    regions = ["na", "eu", "as"]

    # Optional: let UptimeRobot choose the monitoring region automatically.
    # When omitted or false, the configured regions are used manually.
    # auto_select = true

    thresholds = {
      na = 3000
      eu = 4000
      as = 6000
    }
  }
}
