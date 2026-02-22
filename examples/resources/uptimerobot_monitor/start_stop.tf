resource "uptimerobot_monitor" "start_stop" {
  name      = "Start Stop Monitor"
  type      = "HTTP"
  url       = "https://example.com/health"
  interval  = 300
  timeout   = 30
  is_paused = true
}

