data "uptimerobot_monitors" "production_api" {
  tags     = ["production", "api"]
  group_id = 0

  custom_fields = {
    environment = "production"
  }
}

locals {
  production_api_monitor_ids = toset([
    for id in data.uptimerobot_monitors.production_api.ids : tonumber(id)
  ])
}

resource "uptimerobot_psp" "production" {
  name        = "Production Status"
  monitor_ids = local.production_api_monitor_ids
}

resource "uptimerobot_maintenance_window" "production_api" {
  name     = "Production API Maintenance"
  interval = "weekly"
  duration = 60
  time     = "02:00:00"
  days     = [7]

  auto_add_monitors = false
  monitor_ids       = local.production_api_monitor_ids
}

output "production_api_monitors" {
  value = data.uptimerobot_monitors.production_api.monitors
}
