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

type softwareDataSource struct{}

func NewSoftwareDataSource() datasource.DataSource {
	return &softwareDataSource{}
}

type softwareDataSourceModel struct {
	Term    types.String `tfsdk:"term"`
	TrackID types.Int64  `tfsdk:"track_id"`
	Country types.String `tfsdk:"country"`
	Media   types.String `tfsdk:"media"`
	Limit   types.Int64  `tfsdk:"limit"`
	Results types.List   `tfsdk:"results"`
}

func (d *softwareDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "software"
}

func (d *softwareDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"term": schema.StringAttribute{
				Optional:    true,
				Description: "Search term (e.g. app name). Mutually exclusive with track_id.",
			},
			"track_id": schema.Int64Attribute{
				Optional:    true,
				Description: "iTunes track ID to look up a specific app. Mutually exclusive with term.",
			},
			"country": schema.StringAttribute{
				Optional:    true,
				Description: "ISO 2-letter country code.",
			},
			"media": schema.StringAttribute{
				Optional:    true,
				Description: "Media type, defaults to 'software'.",
			},
			"limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of results when searching by term.",
			},
			"results": schema.ListAttribute{
				Computed:    true,
				Description: "List of software search results.",
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"track_name":  types.StringType,
						"bundle_id":   types.StringType,
						"track_id":    types.Int64Type,
						"seller_name": types.StringType,
					},
				},
			},
		},
	}
}

func (d *softwareDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data softwareDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate input: only one of term or track_id
	if data.Term.IsNull() == data.TrackID.IsNull() {
		resp.Diagnostics.AddError("Invalid Input", "You must provide either 'term' or 'track_id', but not both.")
		return
	}

	var apiURL string
	if !data.TrackID.IsNull() {
		apiURL = fmt.Sprintf("https://itunes.apple.com/lookup?id=%d", data.TrackID.ValueInt64())
	} else {
		query := url.Values{}
		query.Set("term", data.Term.ValueString())
		query.Set("media", "software")
		if !data.Country.IsNull() {
			query.Set("country", data.Country.ValueString())
		}
		if !data.Limit.IsNull() {
			query.Set("limit", fmt.Sprintf("%d", data.Limit.ValueInt64()))
		}
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
		trackName := types.StringValue(fmt.Sprintf("%v", item["trackName"]))
		bundleId := types.StringValue(fmt.Sprintf("%v", item["bundleId"]))
		trackId := types.Int64Value(int64(item["trackId"].(float64)))
		seller := types.StringValue(fmt.Sprintf("%v", item["sellerName"]))

		obj, _ := types.ObjectValue(map[string]attr.Type{
			"track_name":  types.StringType,
			"bundle_id":   types.StringType,
			"track_id":    types.Int64Type,
			"seller_name": types.StringType,
		}, map[string]attr.Value{
			"track_name":  trackName,
			"bundle_id":   bundleId,
			"track_id":    trackId,
			"seller_name": seller,
		})

		resultItems = append(resultItems, obj)
	}

	data.Results, diags = types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"track_name":  types.StringType,
				"bundle_id":   types.StringType,
				"track_id":    types.Int64Type,
				"seller_name": types.StringType,
			},
		},
		resultItems,
	)
	resp.Diagnostics.Append(diags...)
}
