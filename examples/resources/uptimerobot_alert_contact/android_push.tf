resource "uptimerobot_alert_contact" "android_phone" {
  name                    = "On-call Android"
  type                    = "mobile_app"
  notification_events     = "up_and_down"
  ssl_expiration_reminder = true

  one_signal_subscription_id = var.android_one_signal_subscription_id
  one_signal_user_id         = var.android_one_signal_user_id
  device_fingerprint         = var.android_device_fingerprint
  push_token                 = var.android_push_token

  android_push_up_channel   = "uptimerobot-up"
  android_push_down_channel = "uptimerobot-down"
}

variable "android_one_signal_subscription_id" {
  description = "OneSignal subscription ID for the Android device"
  type        = string
  sensitive   = true
}

variable "android_one_signal_user_id" {
  description = "OneSignal user ID for the Android device"
  type        = string
  sensitive   = true
}

variable "android_device_fingerprint" {
  description = "Device fingerprint for the Android device"
  type        = string
  sensitive   = true
}

variable "android_push_token" {
  description = "Push token for the Android device"
  type        = string
  sensitive   = true
  default     = null
}
