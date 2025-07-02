resource "uptimerobot_integration" "telegram_bot" {
  name                     = "Telegram Alerts"
  type                     = "telegram"
  value                    = "123456789:ABCdefGHIjklMNOpqrsTUVwxyz" # Bot token
  custom_value             = "-987654321"                           # Chat ID
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}

resource "uptimerobot_integration" "telegram_personal" {
  name                     = "Personal Telegram"
  type                     = "telegram"
  value                    = var.telegram_bot_token
  custom_value             = var.telegram_chat_id
  enable_notifications_for = 2 # Down events only
  ssl_expiration_reminder  = false
}

variable "telegram_bot_token" {
  description = "Telegram bot token"
  type        = string
  sensitive   = true
}

variable "telegram_chat_id" {
  description = "Telegram chat ID"
  type        = string
  sensitive   = true
}