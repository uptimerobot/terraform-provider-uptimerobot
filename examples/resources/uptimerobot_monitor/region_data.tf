resource "uptimerobot_monitor" "multi_region" {
  name     = "Multi-region Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  timeout  = 30

  region_data = {
    regions = ["na", "eu", "as"]

    thresholds = {
      na = 3000
      eu = 4000
      as = 6000
    }
  }
}
