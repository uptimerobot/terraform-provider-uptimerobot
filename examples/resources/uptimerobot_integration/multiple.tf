resource "uptimerobot_monitor" "website" {
  name     = "My Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  timeout  = 30
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
