resource "uptimerobot_maintenance_window" "weekly_maintenance" {
  name     = "Weekly Maintenance"
  interval = "weekly"
  duration = 60
  time     = "02:00:00"
  days     = [7] # Sunday

  # Automatically add new monitors to this maintenance window
  auto_add_monitors = true
}

resource "uptimerobot_monitor" "website" {
  name     = "Example Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
}

resource "uptimerobot_maintenance_window" "business_hours_maintenance" {
  name     = "Business Hours Maintenance"
  interval = "weekly"
  duration = 120 # 2 hours
  time     = "09:00:00"
  days     = [2, 4] # Tuesday and Thursday

  auto_add_monitors = false
  monitor_ids       = [uptimerobot_monitor.website.id]
}
