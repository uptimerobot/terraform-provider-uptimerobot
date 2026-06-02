data "uptimerobot_monitor_group" "production" {
  name = "Production"
}

resource "uptimerobot_monitor" "api" {
  name     = "Production API"
  type     = "HTTP"
  url      = "https://api.example.com/health"
  interval = 300
  group_id = tonumber(data.uptimerobot_monitor_group.production.id)
}

output "production_group_created_at" {
  value = data.uptimerobot_monitor_group.production.created_at
}
