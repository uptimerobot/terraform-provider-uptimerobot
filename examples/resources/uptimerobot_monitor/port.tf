resource "uptimerobot_monitor" "database_port" {
  name     = "Database Port Check"
  type     = "port"
  url      = "db.example.com"
  port     = 5432
  interval = 300

  # Port monitoring timeout
  timeout = 10

  tags = ["database", "infrastructure"]
}

resource "uptimerobot_monitor" "redis_port" {
  name     = "Redis Port Check"
  type     = "port"
  url      = "redis.example.com"
  port     = 6379
  interval = 60

  tags = ["redis", "cache"]
}

resource "uptimerobot_monitor" "ssh_port" {
  name     = "SSH Port Check"
  type     = "port"
  url      = "server.example.com"
  port     = 22
  interval = 900

  tags = ["ssh", "server"]
}
