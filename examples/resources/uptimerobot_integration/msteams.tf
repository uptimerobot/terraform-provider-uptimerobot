resource "uptimerobot_integration" "msteams" {
  name                     = "Teams On-call"
  type                     = "msteams"
  value                    = "https://contoso.webhook.office.com/webhookb2/..."
  enable_notifications_for = 2
  ssl_expiration_reminder  = true
}
