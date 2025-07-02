resource "uptimerobot_integration" "pushbullet_alerts" {
  name                     = "Pushbullet Alerts"
  type                     = "pushbullet"
  value                    = "o.XXXXXXXXXXXXXXXXXXXXX" # Access token
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

resource "uptimerobot_integration" "pushbullet_critical" {
  name                     = "Critical Pushbullet"
  type                     = "pushbullet"
  value                    = var.pushbullet_access_token
  enable_notifications_for = 2 # Down events only
  ssl_expiration_reminder  = false
}

variable "pushbullet_access_token" {
  description = "Pushbullet access token"
  type        = string
  sensitive   = true
}