# Create a maintenance window
resource "uptimerobot_maintenance_window" "weekly" {
  name       = "Weekly Maintenance"
  type       = "weekly"
  start_time = 1640995200 # Unix timestamp
  duration   = 60         # Duration in minutes
  week_day   = 6          # Saturday
  repeat     = "week"
}

# Create a monitor with the maintenance window
resource "uptimerobot_monitor" "example" {
  name     = "My Website"
  url      = "https://example.com"
  type     = 1   # HTTP(s)
  interval = 300 # 5 minutes

  # Optional settings
  http_method       = "GET"
  ignore_ssl_errors = false
  ssl_check_enabled = true
}

# Create a monitor with keyword checking
resource "uptimerobot_monitor" "with_keyword" {
  name     = "Critical Service"
  url      = "https://api.example.com"
  type     = 1  # HTTP(s)
  interval = 60 # 1 minute

  # Keyword monitoring
  keyword_type  = "exists"
  keyword_value = "success"
}

# Monthly maintenance window
resource "uptimerobot_maintenance_window" "monthly" {
  name       = "Monthly Maintenance"
  type       = "monthly"
  start_time = 1641024000 # Unix timestamp
  duration   = 120        # 2 hours
  month_day  = 1          # First day of the month
  repeat     = "month"
}
