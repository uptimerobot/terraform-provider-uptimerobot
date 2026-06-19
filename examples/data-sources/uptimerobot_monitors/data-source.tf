data "uptimerobot_monitors" "production_api" {
  tags     = ["production", "api"]
  group_id = 0

  custom_fields = {
    environment = "production"
  }
}

resource "uptimerobot_psp" "production" {
  name        = "Production Status"
  monitor_ids = data.uptimerobot_monitors.production_api.ids
}

output "production_api_monitors" {
  value = data.uptimerobot_monitors.production_api.monitors
}
