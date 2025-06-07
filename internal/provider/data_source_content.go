package itunessearchapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	"file_size_bytes":    types.Int64Type,
	"artist_view_url":    types.StringType,
	"artwork_url":        types.StringType,
	"track_view_url":     types.StringType,
	"supported_devices":  types.ListType{ElemType: types.StringType},
	"genres":             types.ListType{ElemType: types.StringType},
	"languages":          types.ListType{ElemType: types.StringType},
	"average_rating":     types.Float64Type,
	"rating_count":       types.Int64Type,
}

type contentDataSource struct{}

func NewContentDataSource() datasource.DataSource {
	return &contentDataSource{}
}

type contentDataSourceModel struct {
	Term    types.String `tfsdk:"term"`
	ID      types.Int64  `tfsdk:"id"`
	Country types.String `tfsdk:"country"`
	Media   types.String `tfsdk:"media"`
	Entity  types.String `tfsdk:"entity"`
	Limit   types.Int64  `tfsdk:"limit"`
	Results types.List   `tfsdk:"results"`
}

func (d *contentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content"
}

func (d *contentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Search for, or lookup content in the iTunes Store.",
		Attributes: map[string]schema.Attribute{
			"term": schema.StringAttribute{
				Optional:    true,
				Description: "Search term (e.g. app name). Mutually exclusive with id.",
			},
			"id": schema.Int64Attribute{
				Optional:    true,
				Description: "iTunes ID to look up specific content. Mutually exclusive with term.",
			},
			"country": schema.StringAttribute{
				Optional:    true,
				Description: "ISO 2-letter country code. See http://en.wikipedia.org/wiki/ ISO_3166-1_alpha-2 for a list of ISO Country Codes.",
			},
			"media": schema.StringAttribute{
				Optional:    true,
				Description: "Media type, defaults to 'all'. Supported values: 'movie', 'podcast', 'music', 'musicVideo', 'audiobook', 'shortFilm', 'tvShow', 'software', 'ebook', 'all'",
			},
			"entity": schema.StringAttribute{
				Optional:    true,
				Description: "The type of results you want returned, relative to the specified media type. For example: movieArtist for a movie media type search. The default is the track entity associated with the specified media type.",
			},
			"limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of results when searching by term.",
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

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		}
	}
	return 0
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int64(val)
		case int:
			return int64(val)
		}
	}
	return 0
}

func getStringList(m map[string]interface{}, key string) []attr.Value {
	var list []attr.Value
	if v, ok := m[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			for _, elem := range arr {
				list = append(list, types.StringValue(fmt.Sprintf("%v", elem)))
			}
		}
	}
	return list
}

func addCommonParameters(query url.Values, data *contentDataSourceModel) {
	if !data.Entity.IsNull() {
		query.Set("entity", data.Entity.ValueString())
	}
	if !data.Country.IsNull() {
		query.Set("country", data.Country.ValueString())
	}
	if !data.Limit.IsNull() {
		query.Set("limit", fmt.Sprintf("%d", data.Limit.ValueInt64()))
	}
}

func (d *contentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data contentDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Term.IsNull() == data.ID.IsNull() {
		resp.Diagnostics.AddError("Invalid Input", "You must provide either 'term' or 'id', but not both.")
		return
	}

	var apiURL string
	if !data.ID.IsNull() {
		query := url.Values{}
		query.Set("id", fmt.Sprintf("%d", data.ID.ValueInt64()))
		addCommonParameters(query, &data)
		apiURL = fmt.Sprintf("https://itunes.apple.com/lookup?%s", query.Encode())
	} else {
		query := url.Values{}
		query.Set("term", data.Term.ValueString())
		if !data.Media.IsNull() {
			query.Set("media", data.Media.ValueString())
		} else {
			query.Set("media", "all")
		}
		addCommonParameters(query, &data)
		apiURL = fmt.Sprintf("https://itunes.apple.com/search?%s", query.Encode())
	}

	httpResp, err := http.Get(apiURL)
	if err != nil {
		resp.Diagnostics.AddError("API request failed", err.Error())
		return
	}
	defer httpResp.Body.Close()

	var apiResp struct {
		Results []map[string]interface{} `json:"results"`
	}

	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		resp.Diagnostics.AddError("Error decoding API response", err.Error())
		return
	}

	var resultItems []attr.Value
	for _, item := range apiResp.Results {
		attrs := map[string]attr.Value{
			"track_name":         types.StringValue(getString(item, "trackName")),
			"bundle_id":          types.StringValue(getString(item, "bundleId")),
			"track_id":           types.Int64Value(getInt64(item, "trackId")),
			"seller_name":        types.StringValue(getString(item, "sellerName")),
			"kind":               types.StringValue(getString(item, "kind")),
			"description":        types.StringValue(getString(item, "description")),
			"release_date":       types.StringValue(getString(item, "releaseDate")),
			"price":              types.Float64Value(getFloat64(item, "price")),
			"formatted_price":    types.StringValue(getString(item, "formattedPrice")),
			"currency":           types.StringValue(getString(item, "currency")),
			"version":            types.StringValue(getString(item, "version")),
			"primary_genre":      types.StringValue(getString(item, "primaryGenreName")),
			"minimum_os_version": types.StringValue(getString(item, "minimumOsVersion")),
			"file_size_bytes":    types.Int64Value(getInt64(item, "fileSizeBytes")),
			"artist_view_url":    types.StringValue(getString(item, "artistViewUrl")),
			"artwork_url":        types.StringValue(getString(item, "artworkUrl512")),
			"track_view_url":     types.StringValue(getString(item, "trackViewUrl")),
			"supported_devices":  types.ListValueMust(types.StringType, getStringList(item, "supportedDevices")),
			"genres":             types.ListValueMust(types.StringType, getStringList(item, "genres")),
			"languages":          types.ListValueMust(types.StringType, getStringList(item, "languageCodesISO2A")),
			"average_rating":     types.Float64Value(getFloat64(item, "averageUserRating")),
			"rating_count":       types.Int64Value(getInt64(item, "userRatingCount")),
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
