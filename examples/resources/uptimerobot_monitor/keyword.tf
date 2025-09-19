resource "uptimerobot_monitor" "api_health" {
  name     = "API Health Check"
  type     = "KEYWORD"
  url      = "https://api.example.com/health"
  interval = 60

  # Look for "healthy" in the response
  keyword_type  = "exists"
  keyword_value = "healthy"

  # Case insensitive search (default)
  keyword_case_type = "CaseInsensitive"

  # Custom HTTP method
  http_method_type = "GET"

  # Expected HTTP response codes
  success_http_response_codes = ["200", "201"]

  # Custom headers
  custom_http_headers = {
    "Authorization" = "Bearer ${var.api_token}"
    "User-Agent"    = "UptimeRobot-Monitor"
  }

  tags = ["api", "critical"]
}
