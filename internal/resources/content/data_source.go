package content

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/client"
)

var contentAttributeTypes = map[string]attr.Type{
	"track_name":         types.StringType,
	"bundle_id":          types.StringType,
	"track_id":           types.Int64Type,
	"seller_name":        types.StringType,
	"kind":               types.StringType,
	"description":        types.StringType,
	"release_date":       types.StringType,
	"price":              types.Float64Type,
	"formatted_price":    types.StringType,
	"currency":           types.StringType,
	"version":            types.StringType,
	"primary_genre":      types.StringType,
	"minimum_os_version": types.StringType,
	"file_size_bytes":    types.StringType,
	"artist_view_url":    types.StringType,
	"artwork_url":        types.StringType,
	"artwork_base64":     types.StringType,
	"track_view_url":     types.StringType,
	"supported_devices":  types.ListType{ElemType: types.StringType},
	"genres":             types.ListType{ElemType: types.StringType},
	"languages":          types.ListType{ElemType: types.StringType},
	"average_rating":     types.Float64Type,
	"rating_count":       types.Int64Type,
}

type contentDataSource struct {
	client *client.Client
}

type contentDataSourceModel struct {
	AppStoreURLs types.List   `tfsdk:"app_store_urls"`
	Term         types.String `tfsdk:"term"`
	IDs          types.List   `tfsdk:"ids"`
	Country      types.String `tfsdk:"country"`
	Media        types.String `tfsdk:"media"`
	Entity       types.String `tfsdk:"entity"`
	Limit        types.Int64  `tfsdk:"limit"`
	Results      types.List   `tfsdk:"results"`
}

// NewContentDataSource creates a new instance of the content data source.
func NewContentDataSource(client *client.Client) datasource.DataSource {
	return &contentDataSource{
		client: client,
	}
}

// Metadata implements the datasource.DataSource interface for the content data source.
func (d *contentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content"
}

// Schema implements the datasource.DataSource interface for the content data source.
func (d *contentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Search for, or lookup content in the iTunes Store.",
		Attributes: map[string]schema.Attribute{
			"app_store_urls": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of App Store URLs. Mutually exclusive with term and ids.",
			},
			"term": schema.StringAttribute{
				Optional:    true,
				Description: "Search term (e.g. app name). Mutually exclusive with id.",
			},
			"ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
				Description: "List of iTunes IDs to look up specific content. Mutually exclusive with term.",
			},
			"country": schema.StringAttribute{
				Optional:    true,
				Description: "ISO 2-letter country code. See http://en.wikipedia.org/wiki/ISO_3166-1_alpha-2 for a list of ISO Country Codes.",
			},
			"media": schema.StringAttribute{
				Optional:    true,
				Description: "Media type, defaults to 'all'. Supported values: 'movie', 'podcast', 'music', 'musicVideo', 'audiobook', 'shortFilm', 'tvShow', 'software', 'ebook', 'all'",
			},
			"entity": schema.StringAttribute{
				Optional:    true,
				Description: "The type of results you want returned, relative to the specified media type.",
			},
			"limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of results.",
			},
			"results": schema.ListAttribute{
				Computed:    true,
				Description: "List of content search results.",
				ElementType: types.ObjectType{
					AttrTypes: contentAttributeTypes,
				},
			},
		},
	}
}

// Read implements the datasource.Read method for the content data source.
func (d *contentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data contentDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	paramCount := 0
	if !data.Term.IsNull() {
		paramCount++
	}
	if !data.IDs.IsNull() {
		paramCount++
	}
	if !data.AppStoreURLs.IsNull() {
		paramCount++
	}

	if paramCount != 1 {
		resp.Diagnostics.AddError(
			"Invalid Input",
			"You must provide exactly one of: 'term', 'ids', or 'app_store_urls'.",
		)
		return
	}

	var results []client.ContentResult

	if !data.AppStoreURLs.IsNull() {
		var urls []string
		diags = data.AppStoreURLs.ElementsAs(ctx, &urls, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var trackIDs []int64
		countryCode := ""

		for _, urlStr := range urls {
			trackID, country, err := d.client.ParseAppStoreURL(urlStr)
			if err != nil {
				resp.Diagnostics.AddError("Invalid App Store URL", err.Error())
				return
			}

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

		batches := d.client.ChunkIDs(trackIDs)
		for _, batch := range batches {
			result, err := d.client.ProcessBatch(ctx, batch,
				data.Entity.ValueString(),
				data.Country.ValueString(),
				data.Limit.ValueInt64())
			if err != nil {
				resp.Diagnostics.AddError("API Request Failed", err.Error())
				return
			}
			results = append(results, result.Results...)
		}

	} else if !data.IDs.IsNull() {
		var ids []int64
		diags = data.IDs.ElementsAs(ctx, &ids, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		batches := d.client.ChunkIDs(ids)
		for _, batch := range batches {
			result, err := d.client.ProcessBatch(ctx, batch,
				data.Entity.ValueString(),
				data.Country.ValueString(),
				data.Limit.ValueInt64())
			if err != nil {
				resp.Diagnostics.AddError("API Request Failed", err.Error())
				return
			}
			results = append(results, result.Results...)
		}

	} else {
		result, err := d.client.Search(ctx,
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

	var resultItems []attr.Value
	for _, result := range results {
		artworkURL := result.ArtworkURL
		if artworkURL != "" && strings.HasSuffix(artworkURL, ".jpg") {
			artworkURL = strings.TrimSuffix(artworkURL, ".jpg") + ".png"
		}

		var artworkBase64 string
		if artworkURL != "" {
			encoded, err := d.client.DownloadAndEncodeImage(ctx, artworkURL)
			if err != nil {
				fmt.Printf("Warning: Failed to download artwork for %s: %v\n",
					result.TrackName, err)
			} else {
				artworkBase64 = encoded
			}
		}

		attrs := map[string]attr.Value{
			"track_name":         types.StringValue(result.TrackName),
			"bundle_id":          types.StringValue(result.BundleID),
			"track_id":           types.Int64Value(result.TrackID),
			"seller_name":        types.StringValue(result.SellerName),
			"kind":               types.StringValue(result.Kind),
			"description":        types.StringValue(result.Description),
			"release_date":       types.StringValue(result.ReleaseDate),
			"price":              types.Float64Value(result.Price),
			"formatted_price":    types.StringValue(result.FormattedPrice),
			"currency":           types.StringValue(result.Currency),
			"version":            types.StringValue(result.Version),
			"primary_genre":      types.StringValue(result.PrimaryGenre),
			"minimum_os_version": types.StringValue(result.MinimumOSVersion),
			"file_size_bytes":    types.StringValue(result.FileSizeBytes),
			"artist_view_url":    types.StringValue(result.ArtistViewURL),
			"artwork_url":        types.StringValue(artworkURL),
			"artwork_base64":     types.StringValue(artworkBase64),
			"track_view_url":     types.StringValue(result.TrackViewURL),
			"supported_devices":  types.ListValueMust(types.StringType, convertToAttrValues(result.SupportedDevices)),
			"genres":             types.ListValueMust(types.StringType, convertToAttrValues(result.Genres)),
			"languages":          types.ListValueMust(types.StringType, convertToAttrValues(result.Languages)),
			"average_rating":     types.Float64Value(result.AverageRating),
			"rating_count":       types.Int64Value(result.RatingCount),
		}

		obj, diags := types.ObjectValue(contentAttributeTypes, attrs)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			continue
		}
		resultItems = append(resultItems, obj)
	}

	data.Results, diags = types.ListValue(
		types.ObjectType{
			AttrTypes: contentAttributeTypes,
		},
		resultItems,
	)
	resp.Diagnostics.Append(diags...)

	resp.State.Set(ctx, &data)
}

// Helper function to convert string slice to attr.Value slice
func convertToAttrValues(strings []string) []attr.Value {
	values := make([]attr.Value, len(strings))
	for i, s := range strings {
		values[i] = types.StringValue(s)
	}
	return values
}
