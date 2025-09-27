resource "uptimerobot_monitor" "server_ping" {
  name     = "Server Ping Check"
  type     = "PING"
  url      = "server.example.com"
  interval = 300

  tags = ["ping", "server", "network"]
}

resource "uptimerobot_monitor" "gateway_ping" {
  name     = "Gateway Ping"
  type     = "PING"
  url      = "gateway.example.com"
  interval = 60

  tags = ["gateway", "critical", "network"]
}
