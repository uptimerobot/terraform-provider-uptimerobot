data "uptimerobot_current_user" "current" {}

output "uptimerobot_plan" {
  value = data.uptimerobot_current_user.current.plan
}

output "uptimerobot_monitor_usage" {
  value = {
    used  = data.uptimerobot_current_user.current.monitors_count
    limit = data.uptimerobot_current_user.current.monitor_limit
  }
}
