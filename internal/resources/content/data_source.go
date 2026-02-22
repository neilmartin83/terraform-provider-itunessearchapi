package content

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/client"
	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/common"
)

var _ datasource.DataSource = &ContentDataSource{}

// ContentDataSource defines the data source implementation.
type ContentDataSource struct {
	client *client.Client
}

// NewContentDataSource returns a new instance of the content data source.
func NewContentDataSource() datasource.DataSource {
	return &ContentDataSource{}
}

// Metadata sets the data source type name.
func (d *ContentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content"
}

// Schema defines the data source schema.
func (d *ContentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Search for, or lookup content in the iTunes Store.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx),
			"app_store_urls": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of App Store URLs. Mutually exclusive with all other selectors.",
				Validators: []validator.List{
					listvalidator.ExactlyOneOf(
						path.MatchRelative().AtParent().AtName("term"),
						path.MatchRelative().AtParent().AtName("ids"),
						path.MatchRelative().AtParent().AtName("amg_artist_ids"),
						path.MatchRelative().AtParent().AtName("amg_album_ids"),
						path.MatchRelative().AtParent().AtName("amg_video_ids"),
						path.MatchRelative().AtParent().AtName("upcs"),
						path.MatchRelative().AtParent().AtName("isbns"),
						path.MatchRelative().AtParent().AtName("bundle_ids"),
					),
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(regexp.MustCompile(`^https://apps\.apple\.com/([a-z]{2})/.*?/id(\d+)`),
							"Each URL must be a valid App Store URL in the format: https://apps.apple.com/{country}/app/{app-name}/id{app-id}"),
					),
				},
			},
			"term": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Search term (e.g. app name). Mutually exclusive with lookup identifiers.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("app_store_urls"),
						path.MatchRoot("ids"),
						path.MatchRoot("amg_artist_ids"),
						path.MatchRoot("amg_album_ids"),
						path.MatchRoot("amg_video_ids"),
						path.MatchRoot("upcs"),
						path.MatchRoot("isbns"),
						path.MatchRoot("bundle_ids"),
					),
				},
			},
			"ids": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of iTunes IDs to look up specific content. Mutually exclusive with all other selectors.",
				Validators: []validator.List{
					listvalidator.ExactlyOneOf(
						path.MatchRoot("app_store_urls"),
						path.MatchRoot("term"),
						path.MatchRoot("amg_artist_ids"),
						path.MatchRoot("amg_album_ids"),
						path.MatchRoot("amg_video_ids"),
						path.MatchRoot("upcs"),
						path.MatchRoot("isbns"),
						path.MatchRoot("bundle_ids"),
					),
				},
			},
			"amg_artist_ids": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of AMG artist IDs for lookup requests.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ExactlyOneOf(
						path.MatchRoot("app_store_urls"),
						path.MatchRoot("term"),
						path.MatchRoot("ids"),
						path.MatchRoot("amg_album_ids"),
						path.MatchRoot("amg_video_ids"),
						path.MatchRoot("upcs"),
						path.MatchRoot("isbns"),
						path.MatchRoot("bundle_ids"),
					),
				},
			},
			"amg_album_ids": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of AMG album IDs for lookup requests.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ExactlyOneOf(
						path.MatchRoot("app_store_urls"),
						path.MatchRoot("term"),
						path.MatchRoot("ids"),
						path.MatchRoot("amg_artist_ids"),
						path.MatchRoot("amg_video_ids"),
						path.MatchRoot("upcs"),
						path.MatchRoot("isbns"),
						path.MatchRoot("bundle_ids"),
					),
				},
			},
			"amg_video_ids": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of AMG video IDs for lookup requests.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ExactlyOneOf(
						path.MatchRoot("app_store_urls"),
						path.MatchRoot("term"),
						path.MatchRoot("ids"),
						path.MatchRoot("amg_artist_ids"),
						path.MatchRoot("amg_album_ids"),
						path.MatchRoot("upcs"),
						path.MatchRoot("isbns"),
						path.MatchRoot("bundle_ids"),
					),
				},
			},
			"upcs": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of UPC/EAN codes for lookup requests.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ExactlyOneOf(
						path.MatchRoot("app_store_urls"),
						path.MatchRoot("term"),
						path.MatchRoot("ids"),
						path.MatchRoot("amg_artist_ids"),
						path.MatchRoot("amg_album_ids"),
						path.MatchRoot("amg_video_ids"),
						path.MatchRoot("isbns"),
						path.MatchRoot("bundle_ids"),
					),
				},
			},
			"isbns": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of ISBN codes for lookup requests.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ExactlyOneOf(
						path.MatchRoot("app_store_urls"),
						path.MatchRoot("term"),
						path.MatchRoot("ids"),
						path.MatchRoot("amg_artist_ids"),
						path.MatchRoot("amg_album_ids"),
						path.MatchRoot("amg_video_ids"),
						path.MatchRoot("upcs"),
						path.MatchRoot("bundle_ids"),
					),
				},
			},
			"bundle_ids": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of application bundle IDs for lookup requests.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ExactlyOneOf(
						path.MatchRoot("app_store_urls"),
						path.MatchRoot("term"),
						path.MatchRoot("ids"),
						path.MatchRoot("amg_artist_ids"),
						path.MatchRoot("amg_album_ids"),
						path.MatchRoot("amg_video_ids"),
						path.MatchRoot("upcs"),
						path.MatchRoot("isbns"),
					),
				},
			},
			"sort": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Sort order for lookup results when supported by the API (amg_artist_ids lookups). Allowed values: popular, recent.",
				Validators: []validator.String{
					stringvalidator.OneOf("popular", "recent"),
					stringvalidator.AlsoRequires(
						path.MatchRoot("amg_artist_ids"),
					),
				},
			},
			"country": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "ISO 2-letter country code (lowercase). See http://en.wikipedia.org/wiki/ISO_3166-1_alpha-2 for a list of ISO Country Codes.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("app_store_urls"),
					),
					stringvalidator.LengthBetween(2, 2),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z]{2}$`), "must be a valid ISO 3166-1 alpha-2 country code"),
				},
			},
			"media": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Media type, defaults to 'all'. Supported values: 'movie', 'podcast', 'music', 'musicVideo', 'audiobook', 'shortFilm', 'tvShow', 'software', 'ebook', 'all'. See the iTunes Search API documentation for more details.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"movie",
						"podcast",
						"music",
						"musicVideo",
						"audiobook",
						"shortFilm",
						"tvShow",
						"software",
						"ebook",
						"all",
					),
				},
			},
			"entity": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The type of results you want returned, relative to the specified media type. Supported values: 'movieArtist', 'movie', 'podcastAuthor', 'podcast', 'podcastEpisode', 'musicArtist', 'musicTrack', 'album', 'musicVideo', 'mix', 'song', 'audiobookAuthor', 'audiobook', 'shortFilmArtist', 'shortFilm', 'tvEpisode', 'tvSeason', 'software', 'iPadSoftware', 'desktopSoftware', 'ebook', 'allArtist', 'allTrack'. See the iTunes Search API documentation for more details.",
				Validators: []validator.String{
					stringvalidator.AlsoRequires(
						path.MatchRoot("media"),
					),
					stringvalidator.OneOf(
						"movieArtist",
						"movie",
						"podcastAuthor",
						"podcast",
						"podcastEpisode",
						"musicArtist",
						"musicTrack",
						"album",
						"musicVideo",
						"mix",
						"song",
						"audiobookAuthor",
						"audiobook",
						"shortFilmArtist",
						"shortFilm",
						"tvEpisode",
						"tvSeason",
						"software",
						"iPadSoftware",
						"desktopSoftware",
						"ebook",
						"allArtist",
						"allTrack",
					),
				},
			},
			"attribute": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Search attribute that constrains which field Apple matches against your term (for example, songTerm, albumTerm, titleTerm).",
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("term")),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9]+(Term|Index)$`), "attribute names must end with Term or Index, such as songTerm or ratingIndex"),
				},
			},
			"lang": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Language for the returned results (en_us or ja_jp).",
				Validators: []validator.String{
					stringvalidator.OneOf("en_us", "ja_jp"),
				},
			},
			"version": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Search result key version to request from Apple (1 or 2).",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AtMost(2),
				},
			},
			"explicit": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to include explicit content in search results. Defaults to true when unset.",
			},
			"offset": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Result offset for paginating term-based searches.",
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRoot("term")),
					int64validator.AtLeast(0),
				},
			},
			"callback": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional JavaScript callback name for JSONP search responses. Terraform automatically unwraps the callback when decoding.",
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("term")),
				},
			},
			"limit": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of results to return. For lookups, this overrides the provider-managed defaults when you need to limit nested collections (for example, top 5 albums per artist). Valid range is 1-200.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AtMost(200),
				},
			},
			"results": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of content search results.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"track_name": schema.StringAttribute{
							MarkdownDescription: "Name of the track.",
							Computed:            true,
						},
						"bundle_id": schema.StringAttribute{
							MarkdownDescription: "Bundle ID for apps.",
							Computed:            true,
						},
						"track_id": schema.Int64Attribute{
							MarkdownDescription: "iTunes track ID.",
							Computed:            true,
						},
						"seller_name": schema.StringAttribute{
							MarkdownDescription: "Name of the seller.",
							Computed:            true,
						},
						"kind": schema.StringAttribute{
							MarkdownDescription: "Kind of content (e.g., software, ebook).",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Description of the content.",
							Computed:            true,
						},
						"release_date": schema.StringAttribute{
							MarkdownDescription: "Release date.",
							Computed:            true,
						},
						"price": schema.Float64Attribute{
							MarkdownDescription: "Price.",
							Computed:            true,
						},
						"formatted_price": schema.StringAttribute{
							MarkdownDescription: "Formatted price string.",
							Computed:            true,
						},
						"currency": schema.StringAttribute{
							MarkdownDescription: "Currency code.",
							Computed:            true,
						},
						"version": schema.StringAttribute{
							MarkdownDescription: "Current version.",
							Computed:            true,
						},
						"primary_genre": schema.StringAttribute{
							MarkdownDescription: "Primary genre.",
							Computed:            true,
						},
						"minimum_os_version": schema.StringAttribute{
							MarkdownDescription: "Minimum OS version required.",
							Computed:            true,
						},
						"file_size_bytes": schema.StringAttribute{
							MarkdownDescription: "File size in bytes.",
							Computed:            true,
						},
						"artist_view_url": schema.StringAttribute{
							MarkdownDescription: "URL to artist view.",
							Computed:            true,
						},
						"artwork_url": schema.StringAttribute{
							MarkdownDescription: "Artwork URL.",
							Computed:            true,
						},
						"artwork_base64": schema.StringAttribute{
							MarkdownDescription: "Base64-encoded artwork image.",
							Computed:            true,
						},
						"track_view_url": schema.StringAttribute{
							MarkdownDescription: "URL to track view.",
							Computed:            true,
						},
						"supported_devices": schema.ListAttribute{
							MarkdownDescription: "List of supported devices.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"genres": schema.ListAttribute{
							MarkdownDescription: "List of genres.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"languages": schema.ListAttribute{
							MarkdownDescription: "List of supported languages.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"average_rating": schema.Float64Attribute{
							MarkdownDescription: "Average user rating.",
							Computed:            true,
						},
						"rating_count": schema.Int64Attribute{
							MarkdownDescription: "Number of ratings.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the data source with the provider-configured client.
func (d *ContentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Read retrieves content from the iTunes Search API based on the configured
// selector (search term or lookup identifiers) and maps the results to state.
func (d *ContentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout := common.DefaultReadTimeout
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		configuredTimeout, timeoutDiags := data.Timeouts.Read(ctx, common.DefaultReadTimeout)
		resp.Diagnostics.Append(timeoutDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		readTimeout = configuredTimeout
	}

	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	var results []client.ContentResult

	if !data.Term.IsNull() {
		searchResults, diags := executeSearch(readCtx, data, d.client)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		results = searchResults
	} else {
		lookupResults, diags := executeLookup(readCtx, &data, d.client)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		results = lookupResults
	}

	data.Results = mapResultsToModel(readCtx, results)

	tflog.Debug(ctx, "Content data source read", map[string]interface{}{
		"result_count": len(data.Results),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
