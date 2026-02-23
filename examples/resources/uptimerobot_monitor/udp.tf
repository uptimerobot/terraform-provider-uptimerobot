resource "uptimerobot_monitor" "udp_service" {
  name     = "UDP Service Check"
  type     = "UDP"
  url      = "dns.google"
  port     = 53
  interval = 300

  config = {
    ip_version = "ipv4Only"

    udp = {
      payload               = "ping"
      packet_loss_threshold = 50
    }
  }

  tags = ["udp", "network", "critical"]
}
