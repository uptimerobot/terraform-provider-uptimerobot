terraform {
  required_providers {
    uptimerobot = {
      source = "uptimerobot/uptimerobot"
      version = "1.0.0"
    }
  }
}

provider "uptimerobot" {
  api_key  = "524ce7d7fe8f647ddbbc419f"
  api_url = "http://localhost:3000"
}