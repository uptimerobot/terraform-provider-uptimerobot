resource "uptimerobot_maintenance_window" "server_migration" {
  name     = "Server Migration"
  interval = "once"
  duration = 240 # 4 hours
  date     = "2024-12-15"
  time     = "01:00:00"

  auto_add_monitors = false
}

resource "uptimerobot_maintenance_window" "emergency_patch" {
  name     = "Emergency Security Patch"
  interval = "once"
  duration = 60
  date     = "2024-08-20"
  time     = "23:00:00"

  auto_add_monitors = true
}
