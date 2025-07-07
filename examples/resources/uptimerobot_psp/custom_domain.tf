resource "uptimerobot_monitor" "website" {
  name     = "Website"
  type     = "http"
  url      = "https://example.com"
  interval = 300
}

resource "uptimerobot_psp" "custom_domain_status" {
  name          = "Example.com Status"
  custom_domain = "status.example.com"
  monitor_ids = [
    uptimerobot_monitor.website.id,
  ]

  # SEO settings
  no_index = false

  # Hide direct URLs for security
  hide_url_links = true
}
