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
