package itunessearchapi

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/client"
	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/resources/content"
)

type iTunesProviderModel struct {
	RequestsPerMinute types.Int64 `tfsdk:"requests_per_minute"`
}

type iTunesProvider struct {
	client *client.Client
}

// Configure configures the iTunes Search API provider with settings from the Terraform configuration.
// It initializes the API client with user-specified settings such as rate limiting.
//
// Parameters:
//   - ctx: Context for configuration operations
//   - req: Contains the provider configuration from Terraform
//   - resp: Used to return configuration results and any diagnostics
//
// The function will populate resp.Diagnostics with any errors encountered during configuration.
func (p *iTunesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config iTunesProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := p.client.Configure(ctx, client.Config{
		RequestsPerMinute: config.RequestsPerMinute.ValueInt64(),
	}); err != nil {
		resp.Diagnostics.AddError("Client Configuration Error", err.Error())
		return
	}
}

// Metadata returns the provider type name.
//
// Parameters:
//   - ctx: Unused context
//   - req: Unused metadata request
//   - resp: Response containing the provider type name
func (p *iTunesProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "itunessearchapi"
}

// Schema defines the provider-level schema for configuration data.
//
// Parameters:
//   - ctx: Unused context
//   - req: Unused schema request
//   - resp: Response containing the provider schema
//
// The schema defines available provider configuration options, including:
//   - requests_per_minute: Optional rate limiting configuration
func (p *iTunesProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with the iTunes Search API: https://performance-partners.apple.com/search-api.",
		Attributes: map[string]schema.Attribute{
			"requests_per_minute": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of API requests per minute (default: 20)",
			},
		},
	}
}

// Resources returns a list of supported provider resources.
//
// Parameters:
//   - ctx: Unused context
//
// Currently returns nil as this provider does not implement any resources.
func (p *iTunesProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

// DataSources returns a list of supported provider data sources.
//
// Parameters:
//   - ctx: Unused context
//
// Returns a list of functions that create new instances of supported data sources:
//   - content: iTunes content search and lookup functionality
func (p *iTunesProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource {
			return content.NewContentDataSource(p.client)
		},
	}
}

// New creates a new instance of the iTunes Search API provider.
//
// Returns a configured provider instance with an initialized but unconfigured client.
// The client will be configured when the provider's Configure method is called.
func New() provider.Provider {
	return &iTunesProvider{
		client: client.NewClient(),
	}
}
