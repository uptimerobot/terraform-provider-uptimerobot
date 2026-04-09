terraform {
  required_providers {
    uptimerobot = {
      source  = "local/providers/uptimerobot"
      version = "1.0.0"
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
  type     = "HTTP"
  interval = 300

  http_method = "GET"
  timeout     = 30
}

# Test Heartbeat Monitor Resource
resource "uptimerobot_monitor" "heartbeat" {
  name         = "Test Heartbeat"
  type         = "HEARTBEAT"
  interval     = 300
  grace_period = 300
}

# Test PSP Resource
resource "uptimerobot_psp" "test" {
  name     = "Test PSP"
  type     = "public"
  sort     = "name-asc"
  monitors = [uptimerobot_monitor.test.id]
}

output "heartbeat_url" {
  value = uptimerobot_monitor.heartbeat.url
}
