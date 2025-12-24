---
page_title: "uptimerobot_monitor Resource - uptimerobot"
subcategory: ""
description: |-
  Manages an UptimeRobot monitor.
---

# uptimerobot_monitor (Resource)

Manages an UptimeRobot monitor.

## Example Usage

### Basic HTTP Monitor

```terraform
resource "uptimerobot_monitor" "website" {
  name     = "My Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  # Optional: SSL certificate expiration monitoring
  ssl_expiration_reminder = true

  # Optional: to send reminder on selected days (0...365)
  config = {
    ssl_expiration_period_days = [20, 30, 44, 52, 67]
  }

  # Optional: SSL check errors
  check_ssl_errors = true

  # Optional: Follow HTTP redirects
  follow_redirections = true

  # Optional: Custom timeout (default is 30 seconds)
  timeout = 30

  # Optional: Tags for organization
  tags = ["production", "web"]
}
```

### HTTP Monitor with Keyword Checking

```terraform
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

variable "api_token" {
  description = "API token for some system"
  type        = string
  sensitive   = true
}
```

### HTTP Monitor with Authentication

```terraform
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
```

### Port Monitor

```terraform
resource "uptimerobot_monitor" "database_port" {
  name     = "Database Port Check"
  type     = "PORT"
  url      = "db.example.com"
  port     = 5432
  interval = 300

  # Port monitoring timeout
  timeout = 10

  tags = ["database", "infrastructure"]
}

resource "uptimerobot_monitor" "redis_port" {
  name     = "Redis Port Check"
  type     = "PORT"
  url      = "redis.example.com"
  port     = 6379
  interval = 60

  tags = ["redis", "cache"]
}

resource "uptimerobot_monitor" "ssh_port" {
  name     = "SSH Port Check"
  type     = "PORT"
  url      = "server.example.com"
  port     = 22
  interval = 900
  timeout  = 30

  tags = ["ssh", "server"]
}
```

### Ping Monitor

```terraform
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
```

### Alert Contacts Example

```terraform
# Alert Contacts
resource "uptimerobot_monitor" "website_with_contacts" {
  name     = "My Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  timeout  = 30

  # Set exact contacts and their semantics
  assigned_alert_contacts = [
    {
      alert_contact_id = "123",
      threshold        = 0,
      recurrence       = 0
    }, # immediate, no repeats
    {
      alert_contact_id = "456",
      threshold        = 3,
      recurrence       = 15
    }, # after 3m, repeat every 15m
  ]
}


# You can also remove alert contacts by omitting the field
# or setting it to null
resource "uptimerobot_monitor" "website_no_contacts" {
  name     = "My Website"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300
  timeout  = 30

  # No assigned_alert_contacts field = remove all alert contacts
}
```

### Heartbeat Example

```terraform
resource "uptimerobot_monitor" "website" {
  name         = "My Website"
  type         = "HEARTBEAT"
  url          = "https://example.com"
  interval     = 300
  grace_period = 300

  # Optional: Tags for organization
  tags = ["production", "web"]
}
```

### Config Example

```terraform
# Set specific days for SSL expiration period days
resource "uptimerobot_monitor" "set_days" {
  name     = "HTTP set days"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  config = {
    ssl_expiration_period_days = [3, 5, 30, 69] # max 10 items in range 0..365
  }
}

# Preserve remote values but manage the block. Nothing will be sent
resource "uptimerobot_monitor" "preserve" {
  name     = "HTTP preserve"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  # Empty block present - provider will read current remote values into state
  # but does NOT update the server
  config = {}
}

# Clear days on server - send an explicit empty list
resource "uptimerobot_monitor" "clear" {
  name     = "HTTP clear"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  config = {
    ssl_expiration_period_days = [] # empty list means clear on server
  }
}

# UI-managed SSL days. Ignore drift if management is preferred via dashboard
resource "uptimerobot_monitor" "ui_driven_ssl" {
  name     = "UI-driven SSL days"
  type     = "HTTP"
  url      = "https://example.com"
  interval = 300

  ssl_expiration_reminder = true

  lifecycle {
    ignore_changes = [config]
  }

  # Optional to keep an empty block so Terraform will mirror current remote values
  # into state without changing them
  config = {}
}

# DNS monitor - manage DNS record lists. Only for type=DNS.
resource "uptimerobot_monitor" "dns_records" {
  name     = "example.org DNS"
  type     = "DNS"
  url      = "example.org"
  interval = 300

  config = {
    dns_records = {
      # Provide only record lists you want to manage.
      # Omit an attribute to preserve it on the server; set [] to clear it.
      a     = ["93.184.216.34"]
      cname = [] # clear on server
    }
  }
}

# DNS on CREATE — API requires config.dns_records, which may be empty
resource "uptimerobot_monitor" "dns" {
  name     = "example.org DNS (create)"
  type     = "DNS"
  url      = "example.org"
  interval = 300

  config = {
    dns_records = {} # required for DNS on create
  }
}

# DNS on UPDATE - to preserve server values, omit the config block entirely
resource "uptimerobot_monitor" "dns_preserve" {
  name     = "example.org DNS (preserve)"
  type     = "DNS"
  url      = "example.org"
  interval = 300

  # No config block - provider will preserves server-side DNS records
}
```

## Monitor Types

- `HTTP` — HTTP(s) monitoring
- `KEYWORD` — Keyword monitoring (searches for specific text)
- `PING` — Ping monitoring
- `PORT` — Port monitoring
- `HEARTBEAT` — Heartbeat monitoring
- `DNS` — DNS record monitoring

## Intervals

Common monitoring intervals:
- `60` - Every minute
- `300` - Every 5 minutes (recommended)
- `600` - Every 10 minutes
- `900` - Every 15 minutes
- `1800` - Every 30 minutes
- `3600` - Every hour

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `interval` (Number) Interval for the monitoring check (in seconds)
- `name` (String) Tip: Write names as plain text (do not use HTML entities like `&amp;`). UptimeRobot may return HTML-escaped values; the provider normalizes them to plain text on read/import.
- `type` (String) Type of the monitor (HTTP, KEYWORD, PING, PORT, HEARTBEAT, DNS)
- `url` (String) Tip: Write url as plain text (do not use HTML entities like `&amp;`). UptimeRobot may return HTML-escaped values; the provider normalizes them to plain text on read/import.

### Optional

- `assigned_alert_contacts` (Attributes Set) Alert contacts assigned to this monitor.

**Semantics**
- Terraform sends exactly what you specify; the provider does not inject hidden defaults.
- **Free plan**: set `threshold = 0`, `recurrence = 0`.
- **Paid plans**: any non-negative minutes for both fields. (see [below for nested schema](#nestedatt--assigned_alert_contacts))
- `auth_type` (String) The authentication type (HTTP_BASIC)
- `check_ssl_errors` (Boolean) If true, monitor checks SSL certificate errors (hostname mismatch, invalid chain, etc.).
- `config` (Attributes) Advanced monitor configuration.

**Semantics**
- **Omit** the block → **preserve** remote values (no change). *(Exception: DNS on create — see DNS rules.)*
- `config = {}` (empty block) → treat as **managed but keep** current remote values. *(Not allowed for DNS; include `dns_records` instead.)*
- `ssl_expiration_period_days = []` → **clear** days on the server; non-empty list sets exactly those days (max 10).

**DNS-only rules**
- For `type = "DNS"`:
  - **Create:** `config` **must** include `dns_records` (it may be empty: `dns_records = {}`).
  - **Update:** if you include `config`, you **must** include `dns_records`. To preserve server values, omit `config`.

**Validation**
- `dns_records` is only valid for DNS monitors.
- SSL settings are valid only for HTTPS URLs on HTTP/KEYWORD monitors. (see [below for nested schema](#nestedatt--config))
- `custom_http_headers` (Map of String) Custom HTTP headers as key:value. **Keys are case-insensitive.** The provider normalizes keys to **lower-case** on read and during planning to avoid false diffs. Tip: add keys in lower-case (e.g., `"content-type" = "application/json"`).
- `domain_expiration_reminder` (Boolean) Whether to enable domain expiration reminders
- `follow_redirections` (Boolean) Whether to follow redirections
- `grace_period` (Number) The grace period (in seconds). Only for HEARTBEAT monitors
- `http_method_type` (String) The HTTP method type (HEAD, GET, POST, PUT, PATCH, DELETE, OPTIONS)
- `http_password` (String, Sensitive) The password for HTTP authentication
- `http_username` (String) The username for HTTP authentication
- `keyword_case_type` (String) Case sensitivity for keyword. One of: CaseSensitive, CaseInsensitive. Omit to leave server as-is.
- `keyword_type` (String) The type of keyword check (ALERT_EXISTS, ALERT_NOT_EXISTS)
- `keyword_value` (String) The keyword to search for
- `maintenance_window_ids` (Set of Number) Today API v3 behavior on update, if maintenance_window_ids is omitted or set to [] they both clear maintenance windows.
					Recommended: To clear, set maintenance_window_ids = []. To manage them, set the exact IDs.
- `port` (Number) The port to monitor
- `post_value_data` (String) JSON body (use jsonencode). Mutually exclusive with post_value_kv.
- `post_value_kv` (Map of String) Key/Value body for application/x-www-form-urlencoded. Mutually exclusive with post_value_data.
- `regional_data` (String) Region for monitoring: na (North America), eu (Europe), as (Asia), oc (Oceania)
- `response_time_threshold` (Number) Response time threshold in milliseconds. Response time over this threshold will trigger an incident
- `ssl_expiration_reminder` (Boolean) Whether to enable SSL expiration reminders
- `success_http_response_codes` (Set of String) The expected HTTP response codes. If not set API applies defaults.
- `tags` (Set of String) Tags for the monitor. Must be lowercase. Duplicates are removed by set semantics.
- `timeout` (Number) Timeout for the check (in seconds). Not applicable for HEARTBEAT; ignored for DNS/PING. If omitted, default value 30 is used.

### Read-Only

- `id` (String) Monitor ID
- `post_value_type` (String) Computed body type used by UptimeRobot when sending the monitor request. Set automatically to RAW_JSON or KEY_VALUE.
- `status` (String) Status of the monitor

<a id="nestedatt--assigned_alert_contacts"></a>
### Nested Schema for `assigned_alert_contacts`

Required:

- `alert_contact_id` (String)
- `recurrence` (Number) Repeat interval (in minutes) for subsequent notifications **while the incident lasts**.

- **Required by the API**
- `0` = no repeat (single notification)
- Any non-negative integer (minutes) on paid plans
- `threshold` (Number) Delay (in minutes) **after the monitor is DOWN** before notifying this contact.

- **Required by the API**
- `0` = notify immediately (Free plan must use `0`)
- Any non-negative integer (minutes) on paid plans


<a id="nestedatt--config"></a>
### Nested Schema for `config`

Optional:

- `dns_records` (Attributes) DNS record lists for DNS monitors. If present on non-DNS types, validation fails. (see [below for nested schema](#nestedatt--config--dns_records))
- `ssl_expiration_period_days` (Set of Number) Reminder days before SSL expiry (0..365). Max 10 items.

- Omit the attribute → **preserve** remote values.
- Empty set `[]` → **clear** values on server.
Effective for HTTPS URLs when `ssl_expiration_reminder = true`.

<a id="nestedatt--config--dns_records"></a>
### Nested Schema for `config.dns_records`

Optional:

- `a` (Set of String)
- `aaaa` (Set of String)
- `cname` (Set of String)
- `dnskey` (Set of String)
- `ds` (Set of String)
- `mx` (Set of String)
- `ns` (Set of String)
- `nsec` (Set of String)
- `nsec3` (Set of String)
- `ptr` (Set of String)
- `soa` (Set of String)
- `spf` (Set of String)
- `srv` (Set of String)
- `txt` (Set of String)
