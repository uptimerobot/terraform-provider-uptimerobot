data "uptimerobot_psp" "production" {
  name = "Example.com Status"
}

data "uptimerobot_psp_announcement" "maintenance" {
  psp_id = tonumber(data.uptimerobot_psp.production.id)
  title  = "Scheduled maintenance"
}

output "maintenance_announcement_status" {
  value = data.uptimerobot_psp_announcement.maintenance.status
}

output "maintenance_announcement_is_pinned" {
  value = data.uptimerobot_psp_announcement.maintenance.is_pinned
}
