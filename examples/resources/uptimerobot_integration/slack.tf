resource "uptimerobot_integration" "team_slack" {
  name                     = "Team Slack"
  type                     = "slack"
  value                    = "https://hooks.slack.com/services/*****REDACTED*****"
  custom_value             = "#monitoring" # Slack channel
  enable_notifications_for = 1             # All notifications
  ssl_expiration_reminder  = true
}

resource "uptimerobot_integration" "critical_slack" {
  name                     = "Critical Alerts Slack"
  type                     = "slack"
  value                    = var.critical_slack_webhook
  custom_value             = "#alerts" # Different channel for critical alerts
  enable_notifications_for = 2         # Down events only
  ssl_expiration_reminder  = false
}

variable "critical_slack_webhook" {
  description = "Slack webhook URL for critical alerts"
  type        = string
  sensitive   = true
}
