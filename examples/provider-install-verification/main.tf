terraform {
  required_providers {
    uptimerobot = {
      source = "uptimerobot/uptimerobot"
    }
  }
}

provider "uptimerobot" {
  api_key = "your-api-key" # Replace with your actual API key
}

resource "uptimerobot_monitor" "example" {
  name     = "Example Monitor"
  url      = "https://example.com"
  type     = "http"
  interval = 300
}

resource "uptimerobot_maintenance_window" "example" {
  name       = "Weekly Maintenance"
  type       = "weekly"
  start_time = 1672531200 # Unix timestamp
  duration   = 60         # minutes
  monitors   = [123, 456] # monitor IDs

  repeat   = "weekly"
  week_day = 0 # Sunday

  description = "Weekly maintenance window for system updates"
  tags        = ["maintenance", "weekly"]
}