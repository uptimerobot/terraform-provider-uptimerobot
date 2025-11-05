# Set specific days for SSL expiration period days
resource "uptimerobot_monitor" "set_days" {
  name     = "HTTP set days"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  config = {
    ssl_expiration_period_days = [3, 5, 30, 69] # max 10 items in range 0..365
  }
}

# Preserve remote values but manage the block. Nothing will be sent
resource "uptimerobot_monitor" "preserve" {
  name     = "HTTP preserve"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  # Empty block present - provider will read current remote values into state
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
    ssl_expiration_period_days = [] # empty list means clear on server
  }
}

# UI-managed SSL days. Ignore drift if management is preferred via dashboard
resource "uptimerobot_monitor" "ui_driven_ssl" {
  name     = "UI-driven SSL days"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  lifecycle {
    ignore_changes = [config]
  }

  # Optional to keep an empty block so Terraform will mirror current remote values
  # into state without changing them
  config = {}
}

# DNS monitor - manage DNS record lists. Only for type=DNS.
resource "uptimerobot_monitor" "dns_records" {
  name     = "example.org DNS"
  type     = "DNS"
  url      = "example.org"
  interval = 300

  config = {
    dns_records = {
      # Provide only record lists you want to manage.
      # Omit an attribute to preserve it on the server; set [] to clear it.
      a     = ["93.184.216.34"]
      cname = [] # clear on server
    }
  }
}

# DNS on CREATE — API requires config.dns_records, which may be empty
resource "uptimerobot_monitor" "dns" {
  name     = "example.org DNS (create)"
  type     = "DNS"
  url      = "example.org"
  interval = 300

  config = {
    dns_records = {} # required for DNS on create
  }
}

# DNS on UPDATE - to preserve server values, omit the config block entirely
resource "uptimerobot_monitor" "dns_preserve" {
  name     = "example.org DNS (preserve)"
  type     = "DNS"
  url      = "example.org"
  interval = 300

  # No config block - provider will preserves server-side DNS records
}
