resource "uptimerobot_monitor" "website" {
  name     = "My Website"
  type     = "http"
  url      = "https://example.com"
  interval = 300

  # Optional: SSL certificate expiration monitoring
  ssl_expiration_reminder = true

  # Optional: Follow HTTP redirects
  follow_redirections = true

  # Optional: Custom timeout (default is 30 seconds)
  timeout = 30

  # Optional: Tags for organization
  tags = ["production", "web"]
}
