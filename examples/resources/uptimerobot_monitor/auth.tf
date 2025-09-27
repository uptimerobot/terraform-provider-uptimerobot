resource "uptimerobot_monitor" "protected_api" {
  name     = "Protected API Endpoint"
  type     = "HTTP"
  url      = "https://api.example.com/protected"
  interval = 300
  timeout  = 30

  # HTTP Basic Authentication
  auth_type     = "HTTP_BASIC"
  http_username = "monitor_user"
  http_password = var.monitor_password

  # Custom HTTP method
  http_method_type = "POST"

  # POST data
  post_value_type = "JSON"
  post_value_data = jsonencode({
    action = "health_check"
    source = "uptime_monitor"
  })

  # Custom headers
  custom_http_headers = {
    "Content-Type" = "application/json"
    "X-API-Key"    = var.api_key
  }

  # Expect specific response codes
  success_http_response_codes = ["200", "202"]

  tags = ["api", "authenticated"]
}

variable "monitor_password" {
  description = "Password for monitor authentication"
  type        = string
  sensitive   = true
}

variable "api_key" {
  description = "API key for monitoring"
  type        = string
  sensitive   = true
}
