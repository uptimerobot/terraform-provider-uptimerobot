resource "uptimerobot_alert_contact" "team_email" {
  name                    = "Team Email"
  type                    = "email"
  value                   = var.team_alert_email
  notification_events     = "down"
  ssl_expiration_reminder = true
  is_active               = true
}

variable "team_alert_email" {
  description = "Email address used for UptimeRobot alerts"
  type        = string
  sensitive   = true
}
