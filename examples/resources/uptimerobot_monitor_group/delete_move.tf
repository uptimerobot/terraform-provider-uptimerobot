resource "uptimerobot_monitor_group" "fallback" {
  name = "Fallback"
}

resource "uptimerobot_monitor_group" "temporary" {
  name = "Temporary"

  # When this group is destroyed, move any remaining monitors to fallback.
  # If omitted, UptimeRobot moves monitors to the default group.
  monitors_new_group_id = tonumber(uptimerobot_monitor_group.fallback.id)
}
