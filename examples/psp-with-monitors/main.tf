# Create monitors for different services
resource "uptimerobot_monitor" "api" {
  name     = "API Service"
  url      = "https://api.example.com"
  type     = 1   # HTTP(s)
  interval = 300 # 5 minutes
}

resource "uptimerobot_monitor" "web" {
  name     = "Web Frontend"
  url      = "https://www.example.com"
  type     = 1 # HTTP(s)
  interval = 300
}

resource "uptimerobot_monitor" "database" {
  name     = "Database Health"
  url      = "https://db.example.com"
  type     = 1  # HTTP(s)
  interval = 60 # 1 minute
}

# Create a public status page for all services
resource "uptimerobot_psp" "main_status" {
  name = "Example.com Status"
  type = "status" # Standard status page
  monitors = [
    uptimerobot_monitor.api.id,
    uptimerobot_monitor.web.id,
    uptimerobot_monitor.database.id
  ]

  # Optional: Customize the status page
  custom_domain   = "status.example.com"
  sort            = "status-desc" # Sort by status, critical first
  theme           = "light"
  hide_urls       = false
  all_time_uptime = true

  # Optional: Add custom branding
  custom_css = <<-EOT
    .header { 
      background-color: #4CAF50;
    }
    .status-good {
      color: #4CAF50;
    }
  EOT

  tags = ["production", "public"]
}

# Create a separate status page for API services only
resource "uptimerobot_psp" "api_status" {
  name = "API Status"
  type = "status"
  monitors = [
    uptimerobot_monitor.api.id,
    uptimerobot_monitor.database.id
  ]

  # Make it password protected
  password = var.api_status_password

  # Different theme and sorting
  theme = "dark"
  sort  = "name-asc" # Sort alphabetically

  tags = ["api", "internal"]
}

# Create a minimal status page for internal use
resource "uptimerobot_psp" "internal" {
  name = "Internal Systems Status"
  type = "status"
  monitors = [
    uptimerobot_monitor.database.id
  ]

  # Hide URLs for security
  hide_urls = true

  # Add custom HTML for internal documentation
  custom_html = <<-EOT
    <div class="notice">
      For internal use only. Contact DevOps team for issues.
      <br>
      Emergency: +1 (555) 123-4567
    </div>
  EOT

  tags = ["internal", "devops"]
}

# Variables
variable "api_status_password" {
  description = "Password for the API status page"
  type        = string
  sensitive   = true
}

# Outputs
output "main_status_url" {
  description = "URL of the main status page"
  value       = uptimerobot_psp.main_status.custom_domain != "" ? "https://${uptimerobot_psp.main_status.custom_domain}" : uptimerobot_psp.main_status.default_url
}

output "api_status_url" {
  description = "URL of the API status page"
  value       = uptimerobot_psp.api_status.default_url
  sensitive   = true # Since this page is password protected
}
