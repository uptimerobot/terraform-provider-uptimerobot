resource "uptimerobot_monitor" "website" {
  name     = "Website"
  type     = "http"
  url      = "https://example.com"
  interval = 300
}

resource "uptimerobot_psp" "branded_status" {
  name = "Example.com Status"
  monitor_ids = [
    uptimerobot_monitor.website.id,
  ]

  # Custom branding
  logo = "https://example.com/logo.png"
  icon = "https://example.com/favicon.ico"

  # Custom styling
  custom_settings = {
    colors = {
      main = "#4CAF50"
      text = "#333333"
      link = "#2196F3"
    }

    font = {
      family = "Arial, sans-serif"
    }

    page = {
      theme   = "light"
      layout  = "default"
      density = "comfortable"
    }

    features = {
      show_uptime_percentage = "1"
      show_overall_uptime    = "1"
      show_bars              = "1"
      enable_floating_status = "1"
    }
  }

  # Google Analytics
  ga_code = "UA-XXXXXXXXX-X"

  # Cookie consent
  show_cookie_bar                = true
  use_small_cookie_consent_modal = true
}
