//go:build acceptance

package content_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/provider"
)

// providerFactories returns a map of provider factories for acceptance tests.
var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"itunessearchapi": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccContentDataSource_BundleID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "itunessearchapi_content" "test" {
  bundle_ids = ["com.apple.Pages"]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.itunessearchapi_content.test", "results.#", "1"),
					resource.TestCheckResourceAttr("data.itunessearchapi_content.test", "results.0.bundle_id", "com.apple.Pages"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.track_id"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.track_name"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.track_view_url"),
				),
			},
		},
	})
}

func TestAccContentDataSource_MultipleBundleIDs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "itunessearchapi_content" "test" {
  bundle_ids = ["com.apple.Pages", "com.apple.Keynote"]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.itunessearchapi_content.test", "results.#", "2"),
				),
			},
		},
	})
}

func TestAccContentDataSource_TermSearch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "itunessearchapi_content" "test" {
  term    = "Pages"
  media   = "software"
  country = "us"
  limit   = 3
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.#"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.track_name"),
				),
			},
		},
	})
}

func TestAccContentDataSource_TermSearchWithEntity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "itunessearchapi_content" "test" {
  term    = "Apple"
  media   = "software"
  entity  = "software"
  country = "us"
  limit   = 2
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.#"),
				),
			},
		},
	})
}

func TestAccContentDataSource_AppStoreURL(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "itunessearchapi_content" "test" {
  app_store_urls = ["https://apps.apple.com/us/app/pages/id361309726"]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.itunessearchapi_content.test", "results.#", "1"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.track_name"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.artwork_base64"),
				),
			},
		},
	})
}

func TestAccContentDataSource_WithCountry(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "itunessearchapi_content" "test" {
  bundle_ids = ["com.apple.Pages"]
  country    = "gb"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.itunessearchapi_content.test", "results.#", "1"),
					resource.TestCheckResourceAttr("data.itunessearchapi_content.test", "results.0.bundle_id", "com.apple.Pages"),
				),
			},
		},
	})
}

func TestAccContentDataSource_ResultFieldsPopulated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "itunessearchapi_content" "test" {
  bundle_ids = ["com.apple.Pages"]
  country    = "us"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.itunessearchapi_content.test", "results.#", "1"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.track_id"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.track_name"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.bundle_id"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.seller_name"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.kind"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.description"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.release_date"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.currency"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.artwork_url"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.artwork_base64"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.track_view_url"),
				),
			},
		},
	})
}

func TestAccContentDataSource_MusicSearch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "itunessearchapi_content" "test" {
  term    = "Beatles"
  media   = "music"
  country = "us"
  limit   = 5
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.#"),
					resource.TestCheckResourceAttrSet("data.itunessearchapi_content.test", "results.0.track_name"),
				),
			},
		},
	})
}
