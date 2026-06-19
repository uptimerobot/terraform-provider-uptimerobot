data "uptimerobot_monitor" "api" {
  name     = "Production API"
  url      = "https://api.example.com/health"
  tags     = ["production", "api"]
  group_id = 0

  custom_fields = {
    environment = "production"
  }
}

resource "uptimerobot_psp" "production" {
  name        = "Production Status"
  monitor_ids = [data.uptimerobot_monitor.api.id]
}

output "api_monitor_tags" {
  value = data.uptimerobot_monitor.api.tags
}
