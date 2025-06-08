locals {
  apps_info = [
    for app in data.itunessearchapi_content.apps : {
      app_store_url = app.results[0].track_view_url
      display_name  = app.results[0].track_name
      description   = app.results[0].description
    }
  ]
}
