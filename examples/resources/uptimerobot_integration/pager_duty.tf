resource "uptimerobot_integration" "pagerduty" {
  name  = "PD Incidents"
  type  = "pagerduty"
  value = var.pagerduty_integration_key # must be >= 32 chars

  location     = "eu" # or "us"
  auto_resolve = true

  enable_notifications_for = 2
  ssl_expiration_reminder  = true
}
