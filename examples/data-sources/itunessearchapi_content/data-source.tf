# Search for content using a term and country
data "itunessearchapi_content" "example" {
  term    = "Microsoft Word"
  country = "gb"
  media   = "software"
  limit   = 3
}

# Lookup content by App Store URL
data "itunessearchapi_content" "example_by_url" {
  app_store_url = "https://apps.apple.com/gb/app/messenger/id1480068668?mt=12"
}

# Lookup content by specific iTunes ID and country
data "itunessearchapi_content" "example_by_id" {
  id      = "462054704"
  country = "gb"
}

# Output the results of the content search
output "content_results" {
  value = data.itunessearchapi_content.example.results
}

# Output the results of the content lookup by URL
output "content_by_url" {
  value = data.itunessearchapi_content.example_by_url.results
}

# Output the results of the content lookup by ID
output "content_by_id" {
  value = data.itunessearchapi_content.example_by_id.results
}
