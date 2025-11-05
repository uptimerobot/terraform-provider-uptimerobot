# Set specific days for SSL expiration period days
resource "uptimerobot_monitor" "set_days" {
  name     = "HTTP set days"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  config = {
    ssl_expiration_period_days = [3, 5, 30, 69] # (max 10, 0..365)
  }
}

# Preserve remote values but manage the block. Nothing will be send
resource "uptimerobot_monitor" "preserve" {
  name     = "HTTP preserve"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  # Empty block present, provider reads current remote values into state,
  # but does NOT update the server
  config = {}
}

# Clear days on server - send an explicit empty list
resource "uptimerobot_monitor" "clear" {
  name     = "HTTP clear"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  config = {
    # Empty list means clear on server
    ssl_expiration_period_days = []
  }
}

# UI managed SSL days. It will ignore drift if management is preferred via dashboard
resource "uptimerobot_monitor" "ui_driven_ssl" {
  name     = "UI-driven SSL days"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  lifecycle {
    # Ignore any config changes (made in UI or elsewhere)
    ignore_changes = [config]
  }

  # Optionally an empty block may be left so Terraform mirrors
  # current remote values into state without changing them
  config = {}
}

# DNS monitor. Manage DNS record lists. Only for type=DNS
resource "uptimerobot_monitor" "dns_records" {
  name     = "example.org DNS"
  type     = "DNS"
  url      = "example.org"
  interval = 300

  config = {
    dns_records = {
      # Provide only the record lists you want to manage.
      # Omit an attribute to preserve it on server and set [] to clear them.
      a     = ["93.184.216.34"]
      cname = [] # clear on server
    }
  }
}

# DNS monitor on create require config
resource "uptimerobot_monitor" "dns" {
  name     = "example.org DNS"
  type     = "DNS"
  url      = "example.org"
  interval = 300

  config = {
    dns_records = {} # required for DNS when config is present (API needs config on create)
  }
}
