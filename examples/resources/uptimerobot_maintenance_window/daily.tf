resource "uptimerobot_maintenance_window" "daily_backup" {
  name     = "Daily Backup Window"
  interval = "daily"
  duration = 30
  time     = "03:00:00"

  auto_add_monitors = true
}

resource "uptimerobot_maintenance_window" "daily_updates" {
  name     = "Daily Security Updates"
  interval = "daily"
  duration = 15
  time     = "04:30:00"

  auto_add_monitors = false
}
