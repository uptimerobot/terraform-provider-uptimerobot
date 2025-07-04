---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "uptimerobot_maintenance_window Resource - uptimerobot"
subcategory: ""
description: |-
  Manages an UptimeRobot maintenance window.
---

# uptimerobot_maintenance_window (Resource)

Manages an UptimeRobot maintenance window.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `duration` (Number) Duration of the maintenance window in minutes
- `interval` (String) Interval of maintenance window (once, daily, weekly, monthly)
- `name` (String) Name of the maintenance window
- `time` (String) Time of the maintenance window (format: HH:mm)

### Optional

- `auto_add_monitors` (Boolean) Automatically add new monitors to maintenance window
- `date` (String) Date of the maintenance window (format: YYYY-MM-DD)
- `days` (List of Number) Days to run maintenance window on (1-7, 1 = Monday)

### Read-Only

- `id` (String) Maintenance window identifier
- `status` (String) Status of the maintenance window
