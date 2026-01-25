package content

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

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
)

var _ datasource.DataSource = &ContentDataSource{}

const defaultReadTimeout = 90 * time.Second

func NewContentDataSource() datasource.DataSource {
	return &ContentDataSource{}
}

// ContentDataSource defines the data source implementation.
type ContentDataSource struct {
	client *client.Client
}

// ContentDataSourceModel describes the data source data model.
type ContentDataSourceModel struct {
	Timeouts     timeouts.Value       `tfsdk:"timeouts"`
	AppStoreURLs types.List           `tfsdk:"app_store_urls"`
	Term         types.String         `tfsdk:"term"`
	IDs          types.List           `tfsdk:"ids"`
	AMGArtistIDs types.List           `tfsdk:"amg_artist_ids"`
	AMGAlbumIDs  types.List           `tfsdk:"amg_album_ids"`
	AMGVideoIDs  types.List           `tfsdk:"amg_video_ids"`
	UPCs         types.List           `tfsdk:"upcs"`
	ISBNs        types.List           `tfsdk:"isbns"`
	BundleIDs    types.List           `tfsdk:"bundle_ids"`
	Country      types.String         `tfsdk:"country"`
	Media        types.String         `tfsdk:"media"`
	Entity       types.String         `tfsdk:"entity"`
	Limit        types.Int64          `tfsdk:"limit"`
	Sort         types.String         `tfsdk:"sort"`
	Attribute    types.String         `tfsdk:"attribute"`
	Lang         types.String         `tfsdk:"lang"`
	Version      types.Int64          `tfsdk:"version"`
	Explicit     types.Bool           `tfsdk:"explicit"`
	Offset       types.Int64          `tfsdk:"offset"`
	Callback     types.String         `tfsdk:"callback"`
	Results      []ContentResultModel `tfsdk:"results"`
}

// ContentResultModel describes a single content search result.
type ContentResultModel struct {
	TrackName        types.String   `tfsdk:"track_name"`
	BundleID         types.String   `tfsdk:"bundle_id"`
	TrackID          types.Int64    `tfsdk:"track_id"`
	SellerName       types.String   `tfsdk:"seller_name"`
	Kind             types.String   `tfsdk:"kind"`
	Description      types.String   `tfsdk:"description"`
	ReleaseDate      types.String   `tfsdk:"release_date"`
	Price            types.Float64  `tfsdk:"price"`
	FormattedPrice   types.String   `tfsdk:"formatted_price"`
	Currency         types.String   `tfsdk:"currency"`
	Version          types.String   `tfsdk:"version"`
	PrimaryGenre     types.String   `tfsdk:"primary_genre"`
	MinimumOSVersion types.String   `tfsdk:"minimum_os_version"`
	FileSizeBytes    types.String   `tfsdk:"file_size_bytes"`
	ArtistViewURL    types.String   `tfsdk:"artist_view_url"`
	ArtworkURL       types.String   `tfsdk:"artwork_url"`
	ArtworkBase64    types.String   `tfsdk:"artwork_base64"`
	TrackViewURL     types.String   `tfsdk:"track_view_url"`
	SupportedDevices []types.String `tfsdk:"supported_devices"`
	Genres           []types.String `tfsdk:"genres"`
	Languages        []types.String `tfsdk:"languages"`
	AverageRating    types.Float64  `tfsdk:"average_rating"`
	RatingCount      types.Int64    `tfsdk:"rating_count"`
}

func (d *ContentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content"
}

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

func (d *ContentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ContentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContentDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout := defaultReadTimeout
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		configuredTimeout, timeoutDiags := data.Timeouts.Read(ctx, defaultReadTimeout)
		resp.Diagnostics.Append(timeoutDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		readTimeout = configuredTimeout
	}

	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	var results []client.ContentResult

	switch {
	case !data.AppStoreURLs.IsNull():
		var urls []string
		resp.Diagnostics.Append(data.AppStoreURLs.ElementsAs(ctx, &urls, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		var trackIDs []int64
		countryCode := ""
		var allMissingURLs []string

		for _, urlStr := range urls {
			trackID, country := parseAppStoreURL(urlStr)

			if countryCode == "" {
				countryCode = country
			} else if countryCode != country {
				resp.Diagnostics.AddError(
					"Inconsistent Countries",
					"All App Store URLs must be from the same country store.",
				)
				return
			}

			trackIDs = append(trackIDs, trackID)
		}

		data.Country = types.StringValue(countryCode)
		baseRequest := buildLookupRequest(data)
		batches := chunkInt64(trackIDs)
		for _, batch := range batches {
			req := baseRequest
			req.IDs = batch
			req.Limit = lookupLimitForBatch(data.Limit, len(batch), true)

			result, err := d.client.Lookup(readCtx, req)
			if err != nil {
				if notFoundErr, ok := err.(*client.NotFoundError); ok {
					for _, id := range notFoundErr.MissingIDs {
						for _, url := range urls {
							if strings.Contains(url, fmt.Sprintf("id%d", id)) {
								allMissingURLs = append(allMissingURLs, url)
							}
						}
					}
					continue
				}
				resp.Diagnostics.AddError("API Request Failed", err.Error())
				return
			}
			results = append(results, result.Results...)
		}

		if len(allMissingURLs) > 0 {
			resp.Diagnostics.AddError(
				"Some URLs not found",
				fmt.Sprintf("The following URLs were not found: %v", allMissingURLs),
			)
			return
		}

	case !data.IDs.IsNull():
		var ids []int64
		resp.Diagnostics.Append(data.IDs.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		var allMissingIDs []int64
		baseRequest := buildLookupRequest(data)
		batches := chunkInt64(ids)
		for _, batch := range batches {
			req := baseRequest
			req.IDs = batch
			req.Limit = lookupLimitForBatch(data.Limit, len(batch), true)

			result, err := d.client.Lookup(readCtx, req)
			if err != nil {
				if notFoundErr, ok := err.(*client.NotFoundError); ok {
					allMissingIDs = append(allMissingIDs, notFoundErr.MissingIDs...)
					continue
				}
				resp.Diagnostics.AddError("API Request Failed", err.Error())
				return
			}
			results = append(results, result.Results...)
		}

		if len(allMissingIDs) > 0 {
			resp.Diagnostics.AddError(
				"Resources Not Found",
				fmt.Sprintf("The following IDs were not found: %v", allMissingIDs),
			)
			return
		}

	case !data.AMGArtistIDs.IsNull():
		var amgIDs []int64
		resp.Diagnostics.Append(data.AMGArtistIDs.ElementsAs(ctx, &amgIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		baseRequest := buildLookupRequest(data)
		batches := chunkInt64(amgIDs)
		for _, batch := range batches {
			req := baseRequest
			req.AMGArtistIDs = batch
			req.Limit = lookupLimitForBatch(data.Limit, len(batch), false)

			result, err := d.client.Lookup(readCtx, req)
			if err != nil {
				resp.Diagnostics.AddError("API Request Failed", err.Error())
				return
			}
			results = append(results, result.Results...)
		}

	case !data.AMGAlbumIDs.IsNull():
		var amgIDs []int64
		resp.Diagnostics.Append(data.AMGAlbumIDs.ElementsAs(ctx, &amgIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		baseRequest := buildLookupRequest(data)
		batches := chunkInt64(amgIDs)
		for _, batch := range batches {
			req := baseRequest
			req.AMGAlbumIDs = batch
			req.Limit = lookupLimitForBatch(data.Limit, len(batch), false)

			result, err := d.client.Lookup(readCtx, req)
			if err != nil {
				resp.Diagnostics.AddError("API Request Failed", err.Error())
				return
			}
			results = append(results, result.Results...)
		}

	case !data.AMGVideoIDs.IsNull():
		var amgIDs []int64
		resp.Diagnostics.Append(data.AMGVideoIDs.ElementsAs(ctx, &amgIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		baseRequest := buildLookupRequest(data)
		batches := chunkInt64(amgIDs)
		for _, batch := range batches {
			req := baseRequest
			req.AMGVideoIDs = batch
			req.Limit = lookupLimitForBatch(data.Limit, len(batch), false)

			result, err := d.client.Lookup(readCtx, req)
			if err != nil {
				resp.Diagnostics.AddError("API Request Failed", err.Error())
				return
			}
			results = append(results, result.Results...)
		}

	case !data.UPCs.IsNull():
		var upcs []string
		resp.Diagnostics.Append(data.UPCs.ElementsAs(ctx, &upcs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		baseRequest := buildLookupRequest(data)
		batches := chunkStrings(upcs)
		for _, batch := range batches {
			req := baseRequest
			req.UPCs = batch
			req.Limit = lookupLimitForBatch(data.Limit, len(batch), false)

			result, err := d.client.Lookup(readCtx, req)
			if err != nil {
				resp.Diagnostics.AddError("API Request Failed", err.Error())
				return
			}
			results = append(results, result.Results...)
		}

	case !data.ISBNs.IsNull():
		var isbns []string
		resp.Diagnostics.Append(data.ISBNs.ElementsAs(ctx, &isbns, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		baseRequest := buildLookupRequest(data)
		batches := chunkStrings(isbns)
		for _, batch := range batches {
			req := baseRequest
			req.ISBNs = batch
			req.Limit = lookupLimitForBatch(data.Limit, len(batch), false)

			result, err := d.client.Lookup(readCtx, req)
			if err != nil {
				resp.Diagnostics.AddError("API Request Failed", err.Error())
				return
			}
			results = append(results, result.Results...)
		}

	case !data.BundleIDs.IsNull():
		var bundleIDs []string
		resp.Diagnostics.Append(data.BundleIDs.ElementsAs(ctx, &bundleIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		baseRequest := buildLookupRequest(data)
		batches := chunkStrings(bundleIDs)
		for _, batch := range batches {
			req := baseRequest
			req.BundleIDs = batch
			req.Limit = lookupLimitForBatch(data.Limit, len(batch), false)

			result, err := d.client.Lookup(readCtx, req)
			if err != nil {
				resp.Diagnostics.AddError("API Request Failed", err.Error())
				return
			}
			results = append(results, result.Results...)
		}

	default:
		searchReq := client.SearchRequest{
			Term:      data.Term.ValueString(),
			Media:     stringValue(data.Media),
			Entity:    stringValue(data.Entity),
			Country:   stringValue(data.Country),
			Attribute: stringValue(data.Attribute),
		}

		if !data.Limit.IsNull() && !data.Limit.IsUnknown() {
			searchReq.Limit = data.Limit.ValueInt64()
		}
		if !data.Lang.IsNull() && !data.Lang.IsUnknown() {
			searchReq.Lang = data.Lang.ValueString()
		}
		if !data.Version.IsNull() && !data.Version.IsUnknown() {
			searchReq.Version = data.Version.ValueInt64()
		}
		if !data.Explicit.IsNull() && !data.Explicit.IsUnknown() {
			explicit := data.Explicit.ValueBool()
			searchReq.Explicit = &explicit
		}
		if !data.Offset.IsNull() && !data.Offset.IsUnknown() {
			offset := data.Offset.ValueInt64()
			searchReq.Offset = &offset
		}
		if !data.Callback.IsNull() && !data.Callback.IsUnknown() {
			searchReq.Callback = data.Callback.ValueString()
		}

		result, err := d.client.Search(readCtx, searchReq)
		if err != nil {
			resp.Diagnostics.AddError("API Request Failed", err.Error())
			return
		}
		results = result.Results
	}

	var resultItems []ContentResultModel
	for _, result := range results {
		artworkURL := result.ArtworkURL
		if artworkURL != "" && strings.HasSuffix(artworkURL, ".jpg") {
			artworkURL = strings.TrimSuffix(artworkURL, ".jpg") + ".png"
		}

		var artworkBase64 string
		if artworkURL != "" {
			encoded, err := downloadAndEncodeImage(readCtx, artworkURL)
			if err != nil {
				tflog.Warn(ctx, "Failed to download artwork", map[string]interface{}{
					"track_name": result.TrackName,
					"error":      err.Error(),
				})
			} else {
				artworkBase64 = encoded
			}
		}

		resultItem := ContentResultModel{
			TrackName:        types.StringValue(result.TrackName),
			BundleID:         types.StringValue(result.BundleID),
			TrackID:          types.Int64Value(result.TrackID),
			SellerName:       types.StringValue(result.SellerName),
			Kind:             types.StringValue(result.Kind),
			Description:      types.StringValue(result.Description),
			ReleaseDate:      types.StringValue(result.ReleaseDate),
			Price:            types.Float64Value(result.Price),
			FormattedPrice:   types.StringValue(result.FormattedPrice),
			Currency:         types.StringValue(result.Currency),
			Version:          types.StringValue(result.Version),
			PrimaryGenre:     types.StringValue(result.PrimaryGenre),
			MinimumOSVersion: types.StringValue(result.MinimumOSVersion),
			FileSizeBytes:    types.StringValue(result.FileSizeBytes),
			ArtistViewURL:    types.StringValue(result.ArtistViewURL),
			ArtworkURL:       types.StringValue(artworkURL),
			ArtworkBase64:    types.StringValue(artworkBase64),
			TrackViewURL:     types.StringValue(result.TrackViewURL),
			AverageRating:    types.Float64Value(result.AverageRating),
			RatingCount:      types.Int64Value(result.RatingCount),
		}

		resultItem.SupportedDevices = make([]types.String, len(result.SupportedDevices))
		for i, device := range result.SupportedDevices {
			resultItem.SupportedDevices[i] = types.StringValue(device)
		}

		resultItem.Genres = make([]types.String, len(result.Genres))
		for i, genre := range result.Genres {
			resultItem.Genres[i] = types.StringValue(genre)
		}

		resultItem.Languages = make([]types.String, len(result.Languages))
		for i, language := range result.Languages {
			resultItem.Languages[i] = types.StringValue(language)
		}

		resultItems = append(resultItems, resultItem)
	}

	data.Results = resultItems

	tflog.Debug(ctx, "Content data source read", map[string]interface{}{
		"result_count": len(data.Results),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// downloadAndEncodeImage downloads an image from a URL and returns it as a base64 encoded string.
func downloadAndEncodeImage(ctx context.Context, imageURL string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("User-Agent", "Terraform-Provider-iTunesSearchAPI")
	req.Header.Set("Accept", "image/*")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error downloading image: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image download failed with status code: %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(imageData), nil
}

// parseAppStoreURL extracts the track ID and country code from an App Store URL.
func parseAppStoreURL(urlStr string) (trackID int64, countryCode string) {
	re := regexp.MustCompile(`^https://apps\.apple\.com/([a-z]{2})/.*?/id(\d+)`)
	matches := re.FindStringSubmatch(urlStr)

	countryCode = matches[1]
	trackID, _ = strconv.ParseInt(matches[2], 10, 64)

	return trackID, countryCode
}

// chunkInt64 splits a slice of int64 values into batches of maxLookupBatchSize.
func chunkInt64(ids []int64) [][]int64 {
	const maxLookupBatchSize = 200
	var batches [][]int64
	for i := 0; i < len(ids); i += maxLookupBatchSize {
		end := i + maxLookupBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batches = append(batches, ids[i:end])
	}
	return batches
}

// chunkStrings splits a slice of strings into batches of maxLookupBatchSize.
func chunkStrings(values []string) [][]string {
	const maxLookupBatchSize = 200
	var batches [][]string
	for i := 0; i < len(values); i += maxLookupBatchSize {
		end := i + maxLookupBatchSize
		if end > len(values) {
			end = len(values)
		}
		batches = append(batches, values[i:end])
	}
	return batches
}

// lookupLimitForBatch returns the effective limit for a lookup request batch.
func lookupLimitForBatch(limit types.Int64, batchSize int, autoAlign bool) int64 {
	if !limit.IsNull() && !limit.IsUnknown() {
		return limit.ValueInt64()
	}
	if autoAlign {
		return int64(batchSize)
	}
	return 0
}

// buildLookupRequest creates a baseline lookup request populated with shared fields.
func buildLookupRequest(data ContentDataSourceModel) client.LookupRequest {
	return client.LookupRequest{
		Entity:  stringValue(data.Entity),
		Country: stringValue(data.Country),
		Sort:    stringValue(data.Sort),
	}
}

// stringValue safely returns the string value when set.
func stringValue(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}
