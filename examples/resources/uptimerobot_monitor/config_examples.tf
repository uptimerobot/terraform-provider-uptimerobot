

# Set specific days
resource "uptimerobot_monitor" "set_days" {
  name     = "HTTP set days"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  config = {
    ssl_expiration_period_days = [3, 5, 30, 69]
  }
}

# Preserve remote values (no change)
resource "uptimerobot_monitor" "preserve" {
  name     = "HTTP preserve"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  config = {}
}

# Clear days on server
resource "uptimerobot_monitor" "clear" {
  name     = "HTTP clear"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  config = {
    ssl_expiration_period_days = []
  }
}
