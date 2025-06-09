package itunessearchapi

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func New() provider.Provider {
	return &iTunesProvider{}
}

type iTunesProvider struct {
	rateLimiter *rate.Limiter
	apiClient   *http.Client
	imgClient   *http.Client
}

func (p *iTunesProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "itunessearchapi"
}

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

func (p *iTunesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config iTunesProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rateLimit := int64(20)
	if !config.RequestsPerMinute.IsNull() {
		rateLimit = config.RequestsPerMinute.ValueInt64()
	}

	p.rateLimiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(rateLimit)), 1)

	transport := &http.Transport{
		MaxIdleConns:       100,
		MaxConnsPerHost:    100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: false,
		DisableKeepAlives:  false,
	}

	p.apiClient = &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	p.imgClient = &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}

func (p *iTunesProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *iTunesProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource {
			return NewContentDataSource(p)
		},
	}
}

type iTunesProviderModel struct {
	RequestsPerMinute types.Int64 `tfsdk:"requests_per_minute"`
}
