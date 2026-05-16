data "uptimerobot_all_alert_contacts" "assignable" {
  status = "active"
}

data "uptimerobot_all_alert_contacts" "mobile_devices" {
  type   = "mobile_app_old"
  status = "active"
}

resource "uptimerobot_monitor" "example" {
  name     = "Example website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  assigned_alert_contacts = [
    for id in data.uptimerobot_all_alert_contacts.mobile_devices.ids : {
      alert_contact_id = id
      threshold        = 0
      recurrence       = 0
    }
  ]
}
