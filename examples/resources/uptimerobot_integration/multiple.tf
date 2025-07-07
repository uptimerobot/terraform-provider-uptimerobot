resource "uptimerobot_monitor" "website" {
  name     = "My Website"
  type     = "http"
  url      = "https://example.com"
  interval = 300
}

# Email integration for general alerts
resource "uptimerobot_integration" "team_email" {
  name                     = "Team Email"
  type                     = "email"
  value                    = "alerts@example.com"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

# Slack integration for team notifications
resource "uptimerobot_integration" "team_slack" {
  name                     = "Team Slack"
  type                     = "slack"
  value                    = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
  custom_value             = "#monitoring"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

# SMS for critical down events only
resource "uptimerobot_integration" "emergency_sms" {
  name                     = "Emergency SMS"
  type                     = "sms"
  value                    = "+1234567890"
  enable_notifications_for = 2 # Down events only
  ssl_expiration_reminder  = false
}

# Webhook for external systems
resource "uptimerobot_integration" "webhook" {
  name                     = "External System Webhook"
  type                     = "webhook"
  value                    = "https://api.example.com/webhook/uptime"
  custom_value             = "POST"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true

  send_as_json = true
  post_value = jsonencode({
    message    = "Monitor $monitorFriendlyName is $alertType"
    timestamp  = "$alertDateTime"
    monitor_id = "$monitorID"
  })
}
