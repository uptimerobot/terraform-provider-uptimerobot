resource "uptimerobot_psp_announcement" "service_update" {
  psp_id  = tonumber(uptimerobot_psp.public_status.id)
  title   = "Service update"
  content = "All systems are operational."
  status  = "published"
  type    = "info"
  # start_date is assigned by the API when omitted.
}
