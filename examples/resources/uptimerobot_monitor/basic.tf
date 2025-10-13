resource "uptimerobot_monitor" "website" {
  name     = "My Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  # Optional: SSL certificate expiration monitoring
  ssl_expiration_reminder = true

  # Optional: to send reminder on selected days (0...365)
  config = {
    ssl_expiration_period_days = [20, 30, 44, 52, 67]
  }

  # Optional: SSL check errors
  check_ssl_errors = true

  # Optional: Follow HTTP redirects
  follow_redirections = true

  # Optional: Custom timeout (default is 30 seconds)
  timeout = 30

  # Optional: Tags for organization
  tags = ["production", "web"]
}
