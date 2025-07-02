resource "uptimerobot_monitor" "internal_api" {
  name     = "Internal API"
  type     = "http"
  url      = "https://internal-api.example.com"
  interval = 300
}

resource "uptimerobot_psp" "internal_status" {
  name = "Internal Services Status"
  monitor_ids = [
    uptimerobot_monitor.internal_api.id,
  ]

  # Password protection for internal use
  custom_settings = {
    features = {
      show_monitor_url = "0" # Hide URLs
    }
  }

  # Prevent search engine indexing
  no_index = true

  # Hide URL links for security
  hide_url_links = true
}
