package itunessearchapi

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func New() provider.Provider {
	return &iTunesProvider{}
}

type iTunesProvider struct{}

func (p *iTunesProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "itunessearchapi"
}

func (p *iTunesProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with the iTunes Search API: https://performance-partners.apple.com/search-api.",
	}
}

func (p *iTunesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
}

func (p *iTunesProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *iTunesProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewContentDataSource,
	}
}
