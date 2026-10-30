# Let UptimeRobot derive a slow response threshold for each region from that region's own response time history.
# Do not set response_time_threshold or region_data.thresholds together with mode "auto".
# A region without enough history yet sends no slow response alerts; auto_thresholds shows the progress.
resource "uptimerobot_monitor" "auto_threshold" {
  name     = "Website with automatic thresholds"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  response_time_threshold_mode = "auto"
}

# uptimerobot_monitor.auto_threshold.auto_thresholds lists each region with its status, threshold_ms, usual_ms and
# days_collected.

# Thresholds you set yourself.
resource "uptimerobot_monitor" "manual_threshold" {
  name     = "Website with a fixed threshold"
  type     = "HTTP"
  url      = "https://example.org"
  interval = 300

  response_time_threshold_mode = "manual"
  response_time_threshold      = 3000
}
