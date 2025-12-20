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
	Country      types.String         `tfsdk:"country"`
	Media        types.String         `tfsdk:"media"`
	Entity       types.String         `tfsdk:"entity"`
	Limit        types.Int64          `tfsdk:"limit"`
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
		Description: "Search for, or lookup content in the iTunes Store.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx),
			"app_store_urls": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of App Store URLs. Mutually exclusive with term and ids.",
				Validators: []validator.List{
					listvalidator.ExactlyOneOf(
						path.MatchRelative().AtParent().AtName("term"),
						path.MatchRelative().AtParent().AtName("ids"),
					),
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(regexp.MustCompile(`^https://apps\.apple\.com/([a-z]{2})/.*?/id(\d+)`),
							"Each URL must be a valid App Store URL in the format: https://apps.apple.com/{country}/app/{app-name}/id{app-id}"),
					),
				},
			},
			"term": schema.StringAttribute{
				Optional:    true,
				Description: "Search term (e.g. app name). Mutually exclusive with app_store_urls and ids.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("app_store_urls"),
						path.MatchRoot("ids"),
					),
				},
			},
			"ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
				Description: "List of iTunes IDs to look up specific content. Mutually exclusive with app_store_urls and term.",
				Validators: []validator.List{
					listvalidator.ExactlyOneOf(
						path.MatchRoot("app_store_urls"),
						path.MatchRoot("term"),
					),
				},
			},
			"country": schema.StringAttribute{
				Optional:    true,
				Description: "ISO 2-letter country code (lowercase). See http://en.wikipedia.org/wiki/ISO_3166-1_alpha-2 for a list of ISO Country Codes.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("app_store_urls"),
					),
					stringvalidator.LengthBetween(2, 2),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z]{2}$`), "must be a valid ISO 3166-1 alpha-2 country code"),
				},
			},
			"media": schema.StringAttribute{
				Optional:    true,
				Description: "Media type, defaults to 'all'. Supported values: 'movie', 'podcast', 'music', 'musicVideo', 'audiobook', 'shortFilm', 'tvShow', 'software', 'ebook', 'all'",
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
				Optional:    true,
				Description: "The type of results you want returned, relative to the specified media type.",
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
			"limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of results.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AtMost(200),
				},
			},
			"results": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of content search results.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"track_name": schema.StringAttribute{
							Description: "Name of the track.",
							Computed:    true,
						},
						"bundle_id": schema.StringAttribute{
							Description: "Bundle ID for apps.",
							Computed:    true,
						},
						"track_id": schema.Int64Attribute{
							Description: "iTunes track ID.",
							Computed:    true,
						},
						"seller_name": schema.StringAttribute{
							Description: "Name of the seller.",
							Computed:    true,
						},
						"kind": schema.StringAttribute{
							Description: "Kind of content (e.g., software, ebook).",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the content.",
							Computed:    true,
						},
						"release_date": schema.StringAttribute{
							Description: "Release date.",
							Computed:    true,
						},
						"price": schema.Float64Attribute{
							Description: "Price.",
							Computed:    true,
						},
						"formatted_price": schema.StringAttribute{
							Description: "Formatted price string.",
							Computed:    true,
						},
						"currency": schema.StringAttribute{
							Description: "Currency code.",
							Computed:    true,
						},
						"version": schema.StringAttribute{
							Description: "Current version.",
							Computed:    true,
						},
						"primary_genre": schema.StringAttribute{
							Description: "Primary genre.",
							Computed:    true,
						},
						"minimum_os_version": schema.StringAttribute{
							Description: "Minimum OS version required.",
							Computed:    true,
						},
						"file_size_bytes": schema.StringAttribute{
							Description: "File size in bytes.",
							Computed:    true,
						},
						"artist_view_url": schema.StringAttribute{
							Description: "URL to artist view.",
							Computed:    true,
						},
						"artwork_url": schema.StringAttribute{
							Description: "Artwork URL.",
							Computed:    true,
						},
						"artwork_base64": schema.StringAttribute{
							Description: "Base64-encoded artwork image.",
							Computed:    true,
						},
						"track_view_url": schema.StringAttribute{
							Description: "URL to track view.",
							Computed:    true,
						},
						"supported_devices": schema.ListAttribute{
							Description: "List of supported devices.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"genres": schema.ListAttribute{
							Description: "List of genres.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"languages": schema.ListAttribute{
							Description: "List of supported languages.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"average_rating": schema.Float64Attribute{
							Description: "Average user rating.",
							Computed:    true,
						},
						"rating_count": schema.Int64Attribute{
							Description: "Number of ratings.",
							Computed:    true,
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

	if !data.AppStoreURLs.IsNull() {
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

		batches := chunkIDs(trackIDs)
		for _, batch := range batches {
			result, err := d.client.Lookup(readCtx, batch,
				data.Entity.ValueString(),
				data.Country.ValueString(),
				data.Limit.ValueInt64())

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

	} else if !data.IDs.IsNull() {
		var ids []int64
		resp.Diagnostics.Append(data.IDs.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		var allMissingIDs []int64
		batches := chunkIDs(ids)
		for _, batch := range batches {
			result, err := d.client.Lookup(readCtx, batch,
				data.Entity.ValueString(),
				data.Country.ValueString(),
				data.Limit.ValueInt64())

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

	} else {
		result, err := d.client.Search(readCtx,
			data.Term.ValueString(),
			data.Media.ValueString(),
			data.Entity.ValueString(),
			data.Country.ValueString(),
			data.Limit.ValueInt64())
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

// chunkIDs splits a slice of IDs into batches of maxLookupBatchSize.
func chunkIDs(ids []int64) [][]int64 {
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
