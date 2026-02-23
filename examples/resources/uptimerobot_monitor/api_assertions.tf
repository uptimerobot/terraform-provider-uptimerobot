resource "uptimerobot_monitor" "api_assertions" {
  name     = "API assertions"
  type     = "API"
  url      = "https://example.com/api/health"
  interval = 300
  timeout  = 30

  config = {
    # API monitors can also force IP family now.
    ip_version = "ipv4Only"

    api_assertions = {
      logic = "AND"
      checks = [
        {
          property   = "$.status"
          comparison = "equals"
          target     = jsonencode("ok")
        },
        {
          property   = "$.count"
          comparison = "greater_than"
          target     = jsonencode(0)
        },
      ]
    }
  }
}
