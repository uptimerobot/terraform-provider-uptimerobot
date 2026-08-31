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
          # jsonencode is decoded once by the provider and sent as a native
          # JSON array. Array order and nested JSON types remain significant.
          property   = "$.expected_items"
          comparison = "equals"
          target = jsonencode([
            {
              status  = "active"
              enabled = true
            },
            "complete",
            null,
          ])
        },
        {
          # An object selected from the response contains this recursive
          # object subset. Nested arrays, when present, still compare exactly.
          property   = "$.metadata"
          comparison = "contains"
          target = jsonencode({
            region = "eu"
            flags = {
              healthy = true
            }
          })
        },
        {
          # An array contains an object only when one element is exactly equal
          # to the target object; this is not object-subset matching.
          property   = "$.items"
          comparison = "contains"
          target = jsonencode({
            status  = "active"
            enabled = true
          })
        },
        {
          property   = "$.items"
          comparison = "length_greater_than"
          target     = jsonencode(0)
        },
        {
          property   = "$.metadata"
          comparison = "is_not_empty"
        },
      ]
    }
  }
}
