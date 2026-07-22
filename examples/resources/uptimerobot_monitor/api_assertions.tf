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
          # API Internal validates the documented safe RFC 9535 subset.
          property   = "$.items[?(@.status == 'active')].enabled"
          comparison = "is_boolean"
        },
        {
          property   = "headers.Content-Type"
          comparison = "contains"
          target     = jsonencode("application/json")
        },
        {
          property   = "headers.X-Request-Id"
          comparison = "exists"
        },
        {
          property   = "status_code"
          comparison = "less_than"
          target     = jsonencode(500)
        },
        {
          property   = "body"
          comparison = "not_contains"
          target     = jsonencode("fatal")
        },
      ]
    }
  }
}
