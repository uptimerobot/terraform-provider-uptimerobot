data "uptimerobot_tag" "production" {
  name = "production"
}

resource "uptimerobot_psp" "production_status" {
  name = "Production Services"

  tag_ids = [
    tonumber(data.uptimerobot_tag.production.id),
  ]
}
