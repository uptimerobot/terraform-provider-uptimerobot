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
  value                    = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
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

### Email Integration

```terraform
resource "uptimerobot_integration" "team_email" {
  name                     = "Team Email"
  type                     = "email"
  value                    = "alerts@example.com"
  enable_notifications_for = 1 # All notifications
  ssl_expiration_reminder  = true
}

resource "uptimerobot_integration" "oncall_email" {
  name                     = "On-Call Engineer"
  type                     = "email"
  value                    = "oncall@example.com"
  enable_notifications_for = 2 # Down events only
  ssl_expiration_reminder  = false
}

resource "uptimerobot_integration" "devops_email" {
  name                     = "DevOps Team"
  type                     = "email"
  value                    = "devops@example.com"
  enable_notifications_for = 1 # All notifications
  ssl_expiration_reminder  = true
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

### SMS Integration

```terraform
resource "uptimerobot_integration" "emergency_sms" {
  name                     = "Emergency SMS"
  type                     = "sms"
  value                    = "+1234567890" # Phone number
  enable_notifications_for = 2             # Down events only
  ssl_expiration_reminder  = false         # Usually not needed for SMS
}

resource "uptimerobot_integration" "oncall_sms" {
  name                     = "On-Call SMS"
  type                     = "sms"
  value                    = var.oncall_phone
  enable_notifications_for = 2 # Critical down events only
  ssl_expiration_reminder  = false
}

variable "oncall_phone" {
  description = "On-call engineer phone number"
  type        = string
  sensitive   = true
}
```

### Multiple Integrations

```terraform
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
```

## Integration Types

- `slack` - Slack webhook integration
- `email` - Email notifications
- `webhook` - Custom webhook integration
- `sms` - SMS notifications
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
- `type` (String) The type of the integration (slack, email, webhook, sms, discord, telegram, pushover, pushbullet).
- `value` (String) The value for the integration (e.g. webhook URL, email address).

### Optional

- `custom_value` (String) The custom value for the integration. Only valid for slack (#channel), telegram (chat_id), and pushover (device name).
- `post_value` (String) The POST value to send with the webhook. Only valid for webhook integrations.
- `send_as_json` (Boolean) Whether to send the webhook payload as JSON. Only valid for webhook integrations.
- `send_as_query_string` (Boolean) Whether to send the webhook payload as query string. Only valid for webhook integrations.

### Read-Only

- `id` (String) The ID of this integration.
