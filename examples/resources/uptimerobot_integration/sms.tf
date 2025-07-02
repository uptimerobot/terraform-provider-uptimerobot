resource "uptimerobot_integration" "emergency_sms" {
  name                     = "Emergency SMS"
  type                     = "sms"
  value                    = "+1234567890" # Phone number
  enable_notifications_for = 2             # Down events only
  ssl_expiration_reminder  = false         # Usually not needed for SMS
}

resource "uptimerobot_integration" "oncall_sms" {
  name                     = "On-Call SMS"
  type                     = "sms"
  value                    = var.oncall_phone
  enable_notifications_for = 2 # Critical down events only
  ssl_expiration_reminder  = false
}

variable "oncall_phone" {
  description = "On-call engineer phone number"
  type        = string
  sensitive   = true
}
