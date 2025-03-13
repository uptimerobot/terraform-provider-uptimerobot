terraform {
  required_providers {
    uptimerobot = {
      source = "uptimerobot/uptimerobot"
      version = "0.0.1"
    }
  }
}

provider "uptimerobot" {
  api_key  = "524ce7d7fe8f647ddbbc419f"
  endpoint = "http://localhost:3000"
}