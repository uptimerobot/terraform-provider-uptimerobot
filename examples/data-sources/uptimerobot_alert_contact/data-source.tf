data "uptimerobot_alert_contact" "mobile" {
  name = "My Phone"
  type = "mobile_app"
}

resource "uptimerobot_monitor" "website" {
  name     = "Example Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  assigned_alert_contacts = [
    {
      alert_contact_id = data.uptimerobot_alert_contact.mobile.id
      threshold        = 0
      recurrence       = 0
    }
  ]
}
