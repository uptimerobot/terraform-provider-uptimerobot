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
  api_url = var.uptimerobot_api_url
}

variable "uptimerobot_api_key" {
  description = "UptimeRobot API key"
  type        = string
  sensitive   = true
}

variable "uptimerobot_api_url" {
  description = "UptimeRobot API URL. Override this for local API testing."
  type        = string
  default     = "https://api.uptimerobot.com/v3"
}
