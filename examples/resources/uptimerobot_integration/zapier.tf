resource "uptimerobot_integration" "zapier" {
  name                     = "Zapier Alerts"
  type                     = "zapier"
  value                    = "https://hooks.zapier.com/hooks/catch/123/abc"
  enable_notifications_for = 1
  ssl_expiration_reminder  = false
}
