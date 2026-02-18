# Set specific days for SSL expiration period days
resource "uptimerobot_monitor" "set_days" {
  name     = "DNS set days"
  type     = "DNS"
  url      = "example.com"
  interval = 300

  config = {
    ssl_expiration_period_days = [3, 5, 30, 69] # max 10 items in range 0..365
  }
}

# Preserve remote values but manage the block. Nothing will be sent
resource "uptimerobot_monitor" "preserve" {
  name     = "DNS preserve"
  type     = "DNS"
  url      = "example.com"
  interval = 300

  # Empty block present - provider will read current remote values into state
  # but does NOT update the server
  config = {}
}

# Clear days on server - send an explicit empty list
resource "uptimerobot_monitor" "clear" {
  name     = "DNS clear"
  type     = "DNS"
  url      = "example.com"
  interval = 300

  config = {
    ssl_expiration_period_days = [] # empty list means clear on server
  }
}

# UI-managed SSL days. Ignore drift if management is preferred via dashboard
resource "uptimerobot_monitor" "ui_driven_ssl" {
  name     = "UI-driven DNS SSL days"
  type     = "DNS"
  url      = "example.com"
  interval = 300

  lifecycle {
    ignore_changes = [config]
  }

  # Optional to keep an empty block so Terraform will mirror current remote values
  # into state without changing them
  config = {}
}

# HTTP monitor with forced IPv4
resource "uptimerobot_monitor" "ipv4_only" {
  name     = "HTTP IPv4 only"
  type     = "HTTP"
  url      = "https://example.com/health"
  interval = 300

  config = {
    ip_version = "ipv4Only"
  }
}

# KEYWORD monitor with forced IPv6
resource "uptimerobot_monitor" "ipv6_only_keyword" {
  name              = "Keyword IPv6 only"
  type              = "KEYWORD"
  url               = "https://example.com/status"
  interval          = 300
  keyword_type      = "ALERT_EXISTS"
  keyword_case_type = "CaseInsensitive"
  keyword_value     = "ok"

  config = {
    ip_version = "ipv6Only"
  }
}

# PING monitor with forced IPv4
resource "uptimerobot_monitor" "ipv4_only_ping" {
  name     = "Ping IPv4 only"
  type     = "PING"
  url      = "example.com"
  interval = 300

  config = {
    ip_version = "ipv4Only"
  }
}

# PORT monitor with forced IPv6
resource "uptimerobot_monitor" "ipv6_only_port" {
  name     = "Port IPv6 only"
  type     = "PORT"
  url      = "example.com"
  port     = 443
  interval = 300

  config = {
    ip_version = "ipv6Only"
  }
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

# DNS on CREATE - config is required, even when using defaults
resource "uptimerobot_monitor" "dns" {
  name     = "example.org DNS (create)"
  type     = "DNS"
  url      = "example.org"
  interval = 300

  config = {}
}

# DNS on UPDATE - to preserve server values, omit the config block entirely
resource "uptimerobot_monitor" "dns_preserve" {
  name     = "example.org DNS (preserve)"
  type     = "DNS"
  url      = "example.org"
  interval = 300

  # No config block - provider will preserves server-side DNS records
}
