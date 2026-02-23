resource "uptimerobot_monitor" "api_assertions_null_checks" {
  name     = "API assertions null checks"
  type     = "API"
  url      = "https://example.com/api/status"
  interval = 300
  timeout  = 30

  config = {
    ip_version = "ipv6Only"

    api_assertions = {
      logic = "AND"
      checks = [
        {
          property   = "$.result.value"
          comparison = "is_not_null"
          # target must be omitted for is_null/is_not_null
        },
        {
          property   = "$.result.error"
          comparison = "is_null"
          # target must be omitted for is_null/is_not_null
        },
      ]
    }
  }
}
