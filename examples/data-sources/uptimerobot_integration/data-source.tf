data "uptimerobot_integration" "webhook" {
  name = "Production Webhook"
  type = "webhook"
}

resource "uptimerobot_monitor" "website" {
  name     = "Example Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  assigned_alert_contacts = [
    {
      alert_contact_id = data.uptimerobot_integration.webhook.id
      threshold        = 0
      recurrence       = 0
    }
  ]
}
