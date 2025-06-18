package itunessearchapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	"artwork_base64":     types.StringType,
	"track_view_url":     types.StringType,
	"supported_devices":  types.ListType{ElemType: types.StringType},
	"genres":             types.ListType{ElemType: types.StringType},
	"languages":          types.ListType{ElemType: types.StringType},
	"average_rating":     types.Float64Type,
	"rating_count":       types.Int64Type,
}

type contentDataSource struct {
	provider *iTunesProvider
}

// NewContentDataSource creates a new instance of the content data source.
func NewContentDataSource(p *iTunesProvider) datasource.DataSource {
	return &contentDataSource{
		provider: p,
	}
}

type contentDataSourceModel struct {
	AppStoreURL types.String `tfsdk:"app_store_url"`
	Term        types.String `tfsdk:"term"`
	ID          types.Int64  `tfsdk:"id"`
	Country     types.String `tfsdk:"country"`
	Media       types.String `tfsdk:"media"`
	Entity      types.String `tfsdk:"entity"`
	Limit       types.Int64  `tfsdk:"limit"`
	Results     types.List   `tfsdk:"results"`
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
			"app_store_url": schema.StringAttribute{
				Optional:    true,
				Description: "App Store URL (e.g., https://apps.apple.com/gb/app/facebook/id284882215). Mutually exclusive with term and id.",
			},
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

// getString retrieves a string value from a map by key, converting it to a string if necessary.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// getFloat64 retrieves a float64 value from a map by key, handling both float64 and int types.
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

// getInt64 retrieves an int64 value from a map by key, handling both float64 and int types.
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

// getStringList retrieves a list of strings from a map by key.
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

// addCommonParameters adds common query parameters to the URL query.
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

// parseAppStoreURL parses an App Store URL and extracts the track ID and country code.
func parseAppStoreURL(urlStr string) (trackID int64, countryCode string, err error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid URL format: %v", err)
	}

	if parsedURL.Host != "apps.apple.com" {
		return 0, "", fmt.Errorf("URL must be from apps.apple.com")
	}

	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(parts) < 4 {
		return 0, "", fmt.Errorf("invalid App Store URL format")
	}

	countryCode = parts[0]
	if len(countryCode) != 2 {
		return 0, "", fmt.Errorf("invalid country code in URL")
	}

	idPart := parts[len(parts)-1]
	idStr := idPart

	if strings.Contains(idPart, "?") {
		idStr = strings.Split(idPart, "?")[0]
	}

	if !strings.HasPrefix(idStr, "id") {
		return 0, "", fmt.Errorf("invalid track ID format in URL")
	}

	trackID, err = strconv.ParseInt(strings.TrimPrefix(idStr, "id"), 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid track ID number: %v", err)
	}

	return trackID, countryCode, nil
}

// downloadImage downloads an image from the specified URL and returns its byte content.
func (p *iTunesProvider) downloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("User-Agent", "Terraform-Provider-iTunesSearchAPI")
	req.Header.Add("Accept", "image/*")

	resp, err := p.imgClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error downloading image: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image download failed with status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// downloadAndEncodeImage downloads an image from a URL and returns it as a base64 encoded string
func (p *iTunesProvider) downloadAndEncodeImage(ctx context.Context, imageURL string) (string, error) {
	imageData, err := p.downloadImage(ctx, imageURL)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(imageData), nil
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
	if !data.ID.IsNull() {
		paramCount++
	}
	if !data.AppStoreURL.IsNull() {
		paramCount++
	}

	if paramCount != 1 {
		resp.Diagnostics.AddError(
			"Invalid Input",
			"You must provide exactly one of: 'term', 'id', or 'app_store_url'.",
		)
		return
	}

	var apiURL string
	if !data.AppStoreURL.IsNull() {
		trackID, countryCode, err := parseAppStoreURL(data.AppStoreURL.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid App Store URL", err.Error())
			return
		}

		data.ID = types.Int64Value(trackID)
		data.Country = types.StringValue(countryCode)

		query := url.Values{}
		query.Set("id", fmt.Sprintf("%d", trackID))
		addCommonParameters(query, &data)
		apiURL = fmt.Sprintf("https://itunes.apple.com/lookup?%s", query.Encode())
	} else if !data.ID.IsNull() {
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

	httpResp, err := d.provider.doRateLimitedRequest(ctx, apiURL)
	if err != nil {
		resp.Diagnostics.AddError("API request failed", err.Error())
		return
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	var apiResp struct {
		Results []map[string]interface{} `json:"results"`
	}

	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		resp.Diagnostics.AddError("Error decoding API response", err.Error())
		return
	}

	var resultItems []attr.Value
	for _, item := range apiResp.Results {
		artworkURL := getString(item, "artworkUrl512")
		if artworkURL != "" && strings.HasSuffix(artworkURL, ".jpg") {
			artworkURL = strings.TrimSuffix(artworkURL, ".jpg") + ".png"
		}

		var artworkBase64 string
		if artworkURL != "" {
			encoded, err := d.provider.downloadAndEncodeImage(ctx, artworkURL)
			if err != nil {
				fmt.Printf("Warning: Failed to download artwork for %s: %v\n", getString(item, "trackName"), err)
			} else {
				artworkBase64 = encoded
			}
		}

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
			"artwork_base64":     types.StringValue(artworkBase64),
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
