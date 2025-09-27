# Example HTTP monitor with extended configuration
resource "uptimerobot_monitor" "http_test" {
  name                     = "HTTP Monitor Test"
  type                     = "HTTP"
  url                      = "https://google.com"
  interval                 = 300
  timeout                  = 30
  http_method_type        = "GET"
  follow_redirections     = true
  ssl_expiration_reminder = true
}

# Example Keyword monitor
resource "uptimerobot_monitor" "keyword_test" {
  name             = "Keyword Monitor Test"
  type             = "KEYWORD"
  url              = "https://example.com"
  interval         = 300
  keyword_type     = "ALERT_EXISTS"
  keyword_value    = "Example Domain"
  keyword_case_type = "CaseInsensitive"
  http_method_type = "GET"
  timeout          = 30
}

# Example Port monitor
resource "uptimerobot_monitor" "port_test" {
  name     = "Port Monitor Test"
  type     = "PORT"
  url      = "google.com"
  port     = 443
  interval = 300
  timeout  = 30
}

# Example PSP (Public Status Page)
resource "uptimerobot_psp" "test_page" {
  name           = "Test Status Page"
  monitor_ids    = [
    uptimerobot_monitor.http_test.id,
    uptimerobot_monitor.keyword_test.id,
    uptimerobot_monitor.port_test.id
  ]
  custom_domain  = "status-test.example.com"
  hide_url_links = true
}

# Example Maintenance Window (Weekly)
resource "uptimerobot_maintenance_window" "weekly_maintenance" {
  name              = "Weekly Maintenance"
  interval          = "weekly"
  time              = "22:00:00"    # 10 PM
  duration          = 120           # 2 hours
  auto_add_monitors = true
  days              = [1]          # Monday
}

# Example Maintenance Window (Once)
resource "uptimerobot_maintenance_window" "one_time_maintenance" {
  name              = "One-time Maintenance"
  interval          = "once"
  date              = "2025-03-01"
  time              = "03:00:00"    # Including seconds as required
  duration          = 180          # 3 hours
  auto_add_monitors = false
}
