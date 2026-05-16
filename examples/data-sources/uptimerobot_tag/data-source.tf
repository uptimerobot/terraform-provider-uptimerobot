data "uptimerobot_tag" "production" {
  name = "production"
}

output "production_tag_id" {
  value = data.uptimerobot_tag.production.id
}
