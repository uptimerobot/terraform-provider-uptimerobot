data "uptimerobot_psp" "production" {
  name = "Example.com Status"
}

resource "uptimerobot_psp_announcement" "maintenance" {
  psp_id     = tonumber(data.uptimerobot_psp.production.id)
  title      = "Scheduled maintenance"
  content    = "Maintenance is scheduled for tonight."
  status     = "pending"
  type       = "maintenance"
  start_date = "2030-01-01T00:00:00Z"
}

output "status_page_url_key" {
  value = data.uptimerobot_psp.production.url_key
}
