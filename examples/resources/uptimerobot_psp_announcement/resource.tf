resource "uptimerobot_psp" "public_status" {
  name          = "Example.com Status"
  subscription  = true
  monitor_ids   = []
  homepage_link = "https://example.com"
}

resource "uptimerobot_psp_announcement" "maintenance" {
  psp_id     = tonumber(uptimerobot_psp.public_status.id)
  title      = "Scheduled maintenance"
  content    = "We will perform scheduled maintenance on the API cluster."
  status     = "pending"
  type       = "maintenance"
  start_date = "2030-01-01T00:00:00Z"
  end_date   = "2030-01-01T02:00:00Z"
}
