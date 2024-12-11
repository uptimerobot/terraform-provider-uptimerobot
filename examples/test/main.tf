terraform {
  required_providers {
    uptimerobot = {
      source  = "local/providers/uptimerobot"
      version = "0.1.0"
    }
  }
}

provider "uptimerobot" {
  api_key = "your-api-key" # Replace with your actual API key
}

# Test Monitor Resource
resource "uptimerobot_monitor" "test" {
  name     = "Test Monitor"
  url      = "https://example.com"
  type     = "http"
  interval = 300

  http_method = "GET"
  timeout     = 30
}

# Test PSP Resource
resource "uptimerobot_psp" "test" {
  name     = "Test PSP"
  type     = "public"
  sort     = "name-asc"
  monitors = [uptimerobot_monitor.test.id]
}
