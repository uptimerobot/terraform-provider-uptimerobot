resource "uptimerobot_alert_contact" "ios_phone" {
  name                    = "On-call iPhone"
  type                    = "mobile_app_old"
  notification_events     = "down"
  ssl_expiration_reminder = false

  one_signal_subscription_id = var.ios_one_signal_subscription_id
  one_signal_user_id         = var.ios_one_signal_user_id
  device_fingerprint         = var.ios_device_fingerprint
  push_token                 = var.ios_push_token
}

variable "ios_one_signal_subscription_id" {
  description = "OneSignal subscription ID for the iOS device"
  type        = string
  sensitive   = true
}

variable "ios_one_signal_user_id" {
  description = "OneSignal user ID for the iOS device"
  type        = string
  sensitive   = true
}

variable "ios_device_fingerprint" {
  description = "Device fingerprint for the iOS device"
  type        = string
  sensitive   = true
}

variable "ios_push_token" {
  description = "Push token for the iOS device"
  type        = string
  sensitive   = true
  default     = null
}
