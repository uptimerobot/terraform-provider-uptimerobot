---
page_title: "uptimerobot_integration Resource - uptimerobot"
subcategory: ""
description: |-
  Manages an integration in UptimeRobot.
---

# uptimerobot_integration (Resource)

Manages an integration in UptimeRobot.

## Example Usage

### Slack Integration

```terraform
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
```

### Webhook Integration

```terraform
resource "uptimerobot_integration" "api_webhook" {
  name                     = "API Webhook"
  type                     = "webhook"
  value                    = "https://api.example.com/webhook/uptime"
  custom_value             = "POST" # HTTP method
  enable_notifications_for = 1      # All notifications
  ssl_expiration_reminder  = true

  # Send as JSON payload
  send_as_json         = true
  send_as_query_string = false

  post_value = jsonencode({
    message    = "Alert: $monitorURL is $alertType"
    status     = "$monitorStatusFullName"
    timestamp  = "$alertDateTime"
    monitor_id = "$monitorID"
    alert_type = "$alertType"
  })
}

resource "uptimerobot_integration" "simple_webhook" {
  name                     = "Simple Webhook"
  type                     = "webhook"
  value                    = "https://hooks.example.com/uptime"
  custom_value             = "GET" # Simple GET request
  enable_notifications_for = 2     # Down events only
  ssl_expiration_reminder  = false

  # Send as query string
  send_as_json         = false
  send_as_query_string = true
}
```

### Discord Integration

```terraform
resource "uptimerobot_integration" "team_discord" {
  name                     = "Team Discord"
  type                     = "discord"
  value                    = "https://discord.com/api/webhooks/123456789/abcdefghijklmnopqrstuvwxyz"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

resource "uptimerobot_integration" "critical_discord" {
  name                     = "Critical Discord"
  type                     = "discord"
  value                    = var.discord_webhook_url
  enable_notifications_for = 2 # Down events only
  ssl_expiration_reminder  = false
}

variable "discord_webhook_url" {
  description = "Discord webhook URL for notifications"
  type        = string
  sensitive   = true
}
```

### Telegram Integration

```terraform
resource "uptimerobot_integration" "telegram_bot" {
  name                     = "Telegram Alerts"
  type                     = "telegram"
  value                    = "123456789:ABCdefGHIjklMNOpqrsTUVwxyz" # Bot token
  custom_value             = "-987654321"                           # Chat ID
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

resource "uptimerobot_integration" "telegram_personal" {
  name                     = "Personal Telegram"
  type                     = "telegram"
  value                    = var.telegram_bot_token
  custom_value             = var.telegram_chat_id
  enable_notifications_for = 2 # Down events only
  ssl_expiration_reminder  = false
}

variable "telegram_bot_token" {
  description = "Telegram bot token"
  type        = string
  sensitive   = true
}

variable "telegram_chat_id" {
  description = "Telegram chat ID"
  type        = string
  sensitive   = true
}
```

### Pushover Integration

```terraform
resource "uptimerobot_integration" "pushover_alerts" {
  name                     = "Pushover Alerts"
  type                     = "pushover"
  value                    = "uQiRzpo4DXghDmr9QzzfQu27cmVRsG" # User key
  custom_value             = "azGDORePK8gMaC0QOYAMyEEuzJnyUi" # Device name (optional)
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

resource "uptimerobot_integration" "pushover_emergency" {
  name                     = "Emergency Pushover"
  type                     = "pushover"
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
```

### Pushbullet Integration

```terraform
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
```

### Multiple Integrations

```terraform
resource "uptimerobot_monitor" "website" {
  name     = "My Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
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
```

## Integration Types

- `slack` - Slack webhook integration
- `webhook` - Custom webhook integration
- `discord` - Discord webhook integration
- `telegram` - Telegram bot integration
- `pushover` - Pushover notifications
- `pushbullet` - Pushbullet notifications

## Notification Levels

- `1` - All notifications (up, down, paused, etc.)
- `2` - Down notifications only
- `3` - Custom notification settings

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `enable_notifications_for` (Number) Enable notifications for specific events (1 for all, 2 for down only, 3 for custom).
- `name` (String) The name of the integration.
- `ssl_expiration_reminder` (Boolean) Whether to enable SSL expiration reminders.
- `type` (String) The type of the integration (slack, webhook, discord, telegram, pushover, pushbullet, msteams, zapier, pagerduty, googlechat, splunk, mattermost).
- `value` (String, Sensitive) The value for the integration (e.g. webhook URL).

### Optional

- `custom_value` (String) The custom value for the integration. Only valid for slack (#channel), telegram (chat_id), and pushover (device name). Not used for webhook integrations (webhook settings are stored in dedicated fields).
- `post_value` (String) The POST value to send with the webhook. Only valid for webhook integrations.
- `send_as_json` (Boolean) Whether to send the webhook payload as JSON. Only valid for webhook integrations.
- `send_as_post_parameters` (Boolean) Whether to send the webhook payload as POST parameters. Only valid for webhook integrations.
- `send_as_query_string` (Boolean) Whether to send the webhook payload as query string. Only valid for webhook integrations.

### Read-Only

- `id` (String) The ID of this integration.
