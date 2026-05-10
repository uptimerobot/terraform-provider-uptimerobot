resource "uptimerobot_monitor_group" "production" {
  name = "Production"
}

resource "uptimerobot_monitor" "api" {
  name     = "Production API"
  url      = "https://api.example.com/health"
  type     = "HTTP"
  interval = 300
  timeout  = 30
  group_id = tonumber(uptimerobot_monitor_group.production.id)
}
