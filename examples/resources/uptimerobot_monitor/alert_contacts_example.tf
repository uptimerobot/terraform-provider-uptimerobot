# Example demonstrating adding alert contacts to an existing monitor
# This example shows the fix for the issue where adding alert contacts
# to a monitor that was initially created without any would fail

terraform {
  required_providers {
    uptimerobot = {
      source = "uptimerobot/uptimerobot"
    }
  }
}

provider "uptimerobot" {
  api_key = var.uptimerobot_api_key
}

# Step 1: Create a monitor without alert contacts initially
resource "uptimerobot_monitor" "website" {
  name     = "My Website"
  type     = "http"
  url      = "https://example.com"
  interval = 300
  
  # Initially created without assigned_alert_contacts
  # This used to cause issues when adding them later
}

# Later, you can add alert contacts to the existing monitor
# This configuration change used to produce an "invalid plan" error
# but now works correctly after the fix
resource "uptimerobot_monitor" "website_with_contacts" {
  name     = "My Website"
  type     = "http"
  url      = "https://example.com"
  interval = 300
  
  # Adding alert contacts to an existing monitor now works
  assigned_alert_contacts = [
    "contact-123",
    "contact-456"
  ]
}

# You can also remove alert contacts by omitting the field
# or setting it to null
resource "uptimerobot_monitor" "website_no_contacts" {
  name     = "My Website"
  type     = "http"
  url      = "https://example.com"
  interval = 300
  
  # No assigned_alert_contacts field = remove all alert contacts
}

variable "uptimerobot_api_key" {
  description = "UptimeRobot API key"
  type        = string
  sensitive   = true
}
