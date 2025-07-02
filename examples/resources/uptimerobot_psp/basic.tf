resource "uptimerobot_monitor" "website" {
  name     = "Website"
  type     = "http"
  url      = "https://example.com"
  interval = 300
}

resource "uptimerobot_monitor" "api" {
  name     = "API"
  type     = "http"
  url      = "https://api.example.com"
  interval = 300
}

resource "uptimerobot_psp" "public_status" {
  name = "Example.com Status"
  monitor_ids = [
    uptimerobot_monitor.website.id,
    uptimerobot_monitor.api.id,
  ]
}
