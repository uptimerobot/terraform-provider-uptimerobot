data "uptimerobot_ip_ranges" "checker_europe_ipv4" {
  regions     = ["EUROPE"]
  services    = ["checker"]
  ip_versions = ["ipv4"]
}

output "uptimerobot_checker_europe_ipv4_prefixes" {
  value = data.uptimerobot_ip_ranges.checker_europe_ipv4.ipv4_prefixes
}
