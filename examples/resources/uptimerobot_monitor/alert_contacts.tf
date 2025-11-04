# Alert Contacts
resource "uptimerobot_monitor" "website_with_contacts" {
  name     = "My Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  timeout  = 30

  # Set exact contacts and their semantics
  assigned_alert_contacts = [
    {
      alert_contact_id = "123",
      threshold        = 0,
      recurrence       = 0
    }, # immediate, no repeats
    {
      alert_contact_id = "456",
      threshold        = 3,
      recurrence       = 15
    }, # after 3m, repeat every 15m
  ]
}


# You can also remove alert contacts by omitting the field
# or setting it to null
resource "uptimerobot_monitor" "website_no_contacts" {
  name     = "My Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  timeout  = 30

  # No assigned_alert_contacts field = remove all alert contacts
}
