terraform {
  required_providers {
    itunessearchapi = {
      source = "neilmartin83/itunessearchapi"
    }
  }
}

provider "itunessearchapi" {}

data "itunessearchapi_software" "example" {
  term    = "evernote"
  country = "us"
  limit   = 3
}

output "software_results" {
  value = data.itunessearchapi_software.example.results
}
