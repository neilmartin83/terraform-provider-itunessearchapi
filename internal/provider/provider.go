package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/client"
	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/resources/content"
)

// Ensure ITunesProvider satisfies the provider.Provider interface.
var _ provider.Provider = &ITunesProvider{}

// ITunesProvider defines the provider implementation.
type ITunesProvider struct {
	client  *client.Client
	version string
}

func (p *ITunesProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "itunessearchapi"
	resp.Version = p.version
}

func (p *ITunesProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with the iTunes Search API: https://performance-partners.apple.com/search-api",
		Attributes:  map[string]schema.Attribute{},
	}
}

func (p *ITunesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	clientObj := client.NewClient()
	clientObj.SetLogger(NewTerraformLogger())

	p.client = clientObj
	resp.DataSourceData = clientObj
	resp.ResourceData = clientObj
}

func (p *ITunesProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func (p *ITunesProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		content.NewContentDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ITunesProvider{
			version: version,
		}
	}
}
