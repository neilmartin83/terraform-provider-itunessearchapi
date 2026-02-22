package content

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
