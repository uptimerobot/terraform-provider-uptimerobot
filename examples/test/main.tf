terraform {
  required_providers {
    uptimerobot = {
      source  = "uptimerobot/uptimerobot"
      version = "~> 1.9.1"
    }
  }
}

provider "uptimerobot" {
  api_key = var.uptimerobot_api_key
}

variable "uptimerobot_api_key" {
  description = "UptimeRobot API key"
  type        = string
  sensitive   = true
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
  name         = "Test PSP"
  monitor_sort = "friendly_name_asc"
  monitor_ids  = [uptimerobot_monitor.test.id]
}

output "heartbeat_url" {
  value = uptimerobot_monitor.heartbeat.url
}
