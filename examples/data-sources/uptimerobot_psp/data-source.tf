data "uptimerobot_psp" "production" {
  name = "Example.com Status"
}

resource "uptimerobot_psp_announcement" "maintenance" {
  psp_id = tonumber(data.uptimerobot_psp.production.id)
  title  = "Scheduled maintenance"
  status = "ACTIVE"
  type   = "MAINTENANCE"

  content = "Maintenance is scheduled for tonight."
}

output "status_page_url_key" {
  value = data.uptimerobot_psp.production.url_key
}
