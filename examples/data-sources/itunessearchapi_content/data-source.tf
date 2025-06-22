# Search for content using a term and country
data "itunessearchapi_content" "term_search" {
  term    = "Microsoft Word"
  country = "gb"
  media   = "software"
  limit   = 3
}

# Lookup content by App Store URLs
data "itunessearchapi_content" "app_lookups" {
  app_store_urls = [
    "https://apps.apple.com/gb/app/messenger/id1480068668?mt=12"
  ]
  country = "gb"
}

# Outputs for term search results
output "term_search_results" {
  value = data.itunessearchapi_content.term_search.results
}

# Outputs for lookups
output "lookup_results" {
  value = {
    # Results by URL - using track_id to identify specific apps
    by_url = {
      for result in data.itunessearchapi_content.app_lookups.results :
      result.track_view_url => result
      if contains([1480068668], result.track_id) # Filter for Messenger app
    }

    # Results by ID
    by_id = {
      for result in data.itunessearchapi_content.app_lookups.results :
      result.track_id => result
      if contains([462054704], result.track_id) # Filter for specific ID
    }
  }
}
