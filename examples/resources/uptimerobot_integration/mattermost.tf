resource "uptimerobot_integration" "mattermost" {
  name                     = "Mattermost Alert"
  type                     = "mattermost"
  value                    = "https://mattermost.example/hooks/xxx"
  custom_value             = "Important alert"
  enable_notifications_for = 2
  ssl_expiration_reminder  = true
}
