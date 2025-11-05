resource "uptimerobot_integration" "splunk" {
  name                     = "Splunk Alerts"
  type                     = "splunk"
  value                    = "https://splunk.collector.example/services/collector/raw"
  enable_notifications_for = 1
  ssl_expiration_reminder  = true
}
