resource "uptimerobot_monitor" "server_ping" {
  name     = "Server Ping Check"
  type     = "PING"
  url      = "server.example.com"
  interval = 300

  # Ping timeout in seconds
  timeout = 5

  tags = ["ping", "server", "network"]
}

resource "uptimerobot_monitor" "gateway_ping" {
  name     = "Gateway Ping"
  type     = "PING"
  url      = "gateway.example.com"
  interval = 60

  # Shorter timeout for critical infrastructure
  timeout = 3

  tags = ["gateway", "critical", "network"]
}
