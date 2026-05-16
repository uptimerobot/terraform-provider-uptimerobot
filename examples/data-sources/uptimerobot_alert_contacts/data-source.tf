data "uptimerobot_alert_contacts" "mobile_devices" {
  type   = "mobile_app"
  status = "active"
}

output "mobile_alert_contact_ids" {
  value = data.uptimerobot_alert_contacts.mobile_devices.ids
}
