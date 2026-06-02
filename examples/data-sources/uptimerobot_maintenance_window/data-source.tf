data "uptimerobot_maintenance_window" "weekly" {
  name = "Weekly Maintenance"
}

resource "uptimerobot_monitor" "website" {
  name     = "Example Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  maintenance_window_ids = [
    tonumber(data.uptimerobot_maintenance_window.weekly.id),
  ]
}
