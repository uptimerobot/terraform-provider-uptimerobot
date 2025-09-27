resource "uptimerobot_monitor" "website" {
  name         = "My Website"
  type         = "HEARTBEAT"
  url          = "https://example.com"
  interval     = 300
  grace_period = 300

  # Optional: Tags for organization
  tags = ["production", "web"]
}
