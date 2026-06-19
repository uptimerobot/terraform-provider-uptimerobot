resource "uptimerobot_psp" "all_monitors" {
  name              = "All Services Status"
  auto_add_monitors = true
  monitor_sort      = "status_down_up_paused"
}
