data "uptimerobot_tags" "all" {}

data "uptimerobot_tags" "production" {
  name = "production"
}

output "all_tag_ids" {
  value = data.uptimerobot_tags.all.ids
}

output "production_tag_ids" {
  value = data.uptimerobot_tags.production.ids
}
