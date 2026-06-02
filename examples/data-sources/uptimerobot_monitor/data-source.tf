data "uptimerobot_monitor" "api" {
  name = "Production API"
}

resource "uptimerobot_psp" "production" {
  name        = "Production Status"
  monitor_ids = [data.uptimerobot_monitor.api.id]
}

output "api_monitor_tags" {
  value = data.uptimerobot_monitor.api.tags
}
