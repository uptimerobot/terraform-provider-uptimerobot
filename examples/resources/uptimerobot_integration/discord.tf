resource "uptimerobot_integration" "team_discord" {
  name                     = "Team Discord"
  type                     = "discord"
  value                    = "https://discord.com/api/webhooks/123456789/abcdefghijklmnopqrstuvwxyz"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

resource "uptimerobot_integration" "critical_discord" {
  name                     = "Critical Discord"
  type                     = "discord"
  value                    = var.discord_webhook_url
  enable_notifications_for = 2 # Down events only
  ssl_expiration_reminder  = false
}

variable "discord_webhook_url" {
  description = "Discord webhook URL for notifications"
  type        = string
  sensitive   = true
}