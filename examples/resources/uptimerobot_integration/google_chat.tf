resource "uptimerobot_integration" "gchat" {
  name                     = "GChat Prod Alerts"
  type                     = "googlechat"
  value                    = "https://chat.googleapis.com/v1/spaces/AAA/messages?key=...&token=..."
  custom_value             = "Prod alert"
  enable_notifications_for = 1    # 1=UpAndDown, 2=Down, 3=Up, 4=None
  ssl_expiration_reminder  = true
}