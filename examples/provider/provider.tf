terraform {
  required_providers {
    uptimerobot = {
      source  = "uptimerobot/uptimerobot"
      version = "~> 0.1.0"
    }
  }
}

provider "uptimerobot" {
  api_key = var.uptimerobot_api_key
  # api_url = "https://api.uptimerobot.com/v3" # Optional: Custom API endpoint
}

variable "uptimerobot_api_key" {
  description = "UptimeRobot API key"
  type        = string
  sensitive   = true
}
