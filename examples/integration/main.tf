terraform {
  required_providers {
    uptimerobot = {
      source = "uptimerobot/uptimerobot"
    }
  }
}

provider "uptimerobot" {
  # Configure the provider here
}

# Create a monitor first
resource "uptimerobot_monitor" "website" {
  friendly_name = "My Website"
  type         = "http"
  url          = "https://example.com"
  interval     = 300
}

# 1. Slack Integration
resource "uptimerobot_integration" "slack" {
  friendly_name             = "Team Slack"
  type                     = "slack"
  value                    = "https://hooks.slack.com/services/XXXXX/YYYYY/ZZZZZ"
  custom_value             = "#monitoring"  # Slack channel
  enable_notifications_for = 1  # 1 for all notifications
  ssl_expiration_reminder  = true
}

# 2. Email Integration
resource "uptimerobot_integration" "email" {
  friendly_name             = "Team Email"
  type                     = "email"
  value                    = "alerts@example.com"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

# 3. Webhook Integration
resource "uptimerobot_integration" "webhook" {
  friendly_name             = "API Webhook"
  type                     = "webhook"
  value                    = "https://api.example.com/webhook"
  custom_value             = "POST"  # HTTP method
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
  
  # Webhook-specific fields
  send_as_json            = true
  send_as_query_string    = false
  post_value              = jsonencode({
    message = "Alert: $monitorURL is $alertType"
    status  = "$monitorStatusFullName"
    time    = "$alertDateTime"
  })
}

# 4. SMS Integration
resource "uptimerobot_integration" "sms" {
  friendly_name             = "Emergency SMS"
  type                     = "sms"
  value                    = "+1234567890"  # Phone number
  enable_notifications_for = 2  # 2 for down events only
  ssl_expiration_reminder  = false
}

# 5. Discord Integration
resource "uptimerobot_integration" "discord" {
  friendly_name             = "Team Discord"
  type                     = "discord"
  value                    = "https://discord.com/api/webhooks/XXXXX/YYYYY"
  custom_value             = "Monitoring Alerts"  # Discord webhook name
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

# 6. Telegram Integration
resource "uptimerobot_integration" "telegram" {
  friendly_name             = "Telegram Alerts"
  type                     = "telegram"
  value                    = "bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"  # Bot token
  custom_value             = "-123456789"  # Chat ID
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

# 7. Pushover Integration
resource "uptimerobot_integration" "pushover" {
  friendly_name             = "Pushover Alerts"
  type                     = "pushover"
  value                    = "uQiRzpo4DXghDmr9QzzfQu27cmVRsG"  # User key
  custom_value             = "azGDORePK8gMaC0QOYAMyEEuzJnyUi"  # Application API token
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

# 8. Pushbullet Integration
resource "uptimerobot_integration" "pushbullet" {
  friendly_name             = "Pushbullet Alerts"
  type                     = "pushbullet"
  value                    = "o.XXXXXXXXXXXXXXXXXXXXX"  # Access token
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}
