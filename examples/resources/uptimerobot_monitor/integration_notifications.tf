data "uptimerobot_integration" "pagerduty" {
  name = "Production PagerDuty"
  type = "pagerduty"
}

resource "uptimerobot_monitor" "website_with_pagerduty" {
  name     = "Example Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  timeout  = 30

  assigned_alert_contacts = [
    {
      alert_contact_id = data.uptimerobot_integration.pagerduty.id
      threshold        = 5
      recurrence       = 30
    }
  ]
}
