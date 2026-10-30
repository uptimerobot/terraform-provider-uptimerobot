resource "uptimerobot_monitor_group" "services" {
  name = "Services"
}

resource "uptimerobot_monitor" "service" {
  name     = "Service"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  group_id = uptimerobot_monitor_group.services.id
}

resource "uptimerobot_psp" "grouped" {
  name        = "Service Status"
  monitor_ids = [uptimerobot_monitor.service.id]

  custom_settings = {
    page = {
      show_monitor_groups = true
    }
  }
}
