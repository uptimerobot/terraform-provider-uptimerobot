resource "uptimerobot_integration" "team_email" {
  name                     = "Team Email"
  type                     = "email"
  value                    = "alerts@example.com"
  enable_notifications_for = 1 # All notifications
  ssl_expiration_reminder  = true
}

resource "uptimerobot_integration" "oncall_email" {
  name                     = "On-Call Engineer"
  type                     = "email"
  value                    = "oncall@example.com"
  enable_notifications_for = 2 # Down events only
  ssl_expiration_reminder  = false
}

resource "uptimerobot_integration" "devops_email" {
  name                     = "DevOps Team"
  type                     = "email"
  value                    = "devops@example.com"
  enable_notifications_for = 1 # All notifications
  ssl_expiration_reminder  = true
}
