resource "uptimerobot_monitor" "metadata" {
  name     = "API health with metadata"
  type     = "HTTP"
  url      = "https://example.com/health"
  interval = 300
  timeout  = 30

  custom_fields = {
    environment = "production"
    team        = "platform"
  }
}
