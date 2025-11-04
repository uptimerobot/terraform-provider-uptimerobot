resource "uptimerobot_integration" "pushover_alerts" {
  name                     = "Pushover Alerts"
  type                     = "pushover"
  priority                 = "Normal" 
  value                    = "uQiRzpo4DXghDmr9QzzfQu27cmVRsG" # User key
  custom_value             = "azGDORePK8gMaC0QOYAMyEEuzJnyUi" # Device name (optional)
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

resource "uptimerobot_integration" "pushover_emergency" {
  name                     = "Emergency Pushover"
  type                     = "pushover"
  priority                 = "Emergency" 
  value                    = var.pushover_user_key
  custom_value             = var.pushover_device
  enable_notifications_for = 2 # Down events only
  ssl_expiration_reminder  = false
}

variable "pushover_user_key" {
  description = "Pushover user key"
  type        = string
  sensitive   = true
}

variable "pushover_device" {
  description = "Pushover device name (optional)"
  type        = string
  default     = ""
}