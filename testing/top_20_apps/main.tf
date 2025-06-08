terraform {
  required_providers {
    itunessearchapi = {
      source = "neilmartin83/itunessearchapi"
    }
  }
}

data "itunessearchapi_content" "apps" {
  for_each      = toset(var.app_store_urls)
  app_store_url = each.value
}


resource "local_file" "app_icons" {
  for_each = data.itunessearchapi_content.apps

  content_base64 = each.value.results[0].artwork_base64
  filename       = "${path.module}/icons/${each.value.results[0].track_id}.png"
}

resource "local_file" "directories" {
  content  = ""
  filename = "${path.module}/.terraform/tmp/.keep"

  provisioner "local-exec" {
    command = <<EOT
      mkdir -p ${path.module}/icons
    EOT
  }
}

resource "local_file" "apps_info" {
  content = jsonencode({
    titles = local.apps_info
  })
  filename = "${path.module}/top20_mobile_device_apps_info.json"
}
