// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/client"
	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/common"
)

// appStoreURLRegex matches App Store URLs and captures country code and track ID.
var appStoreURLRegex = regexp.MustCompile(`^https://apps\.apple\.com/([a-z]{2})/.*?/id(\d+)`)

// parseAppStoreURL extracts the track ID and country code from an App Store URL.
func parseAppStoreURL(urlStr string) (trackID int64, countryCode string) {
	matches := appStoreURLRegex.FindStringSubmatch(urlStr)

	countryCode = matches[1]
	trackID, _ = strconv.ParseInt(matches[2], 10, 64)

	return trackID, countryCode
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
		Entity:  common.StringValue(data.Entity),
		Country: common.StringValue(data.Country),
		Sort:    common.StringValue(data.Sort),
	}
}

// downloadAndEncodeImage downloads an image from a URL and returns it as a base64-encoded string.
func downloadAndEncodeImage(ctx context.Context, httpClient *http.Client, imageURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Terraform-Provider-iTunesSearchAPI")
	req.Header.Set("Accept", "image/*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error downloading image: %w", err)
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

// executeLookup dispatches the appropriate lookup request based on which selector
// is set in the data model, handling batching and error aggregation.
func executeLookup(ctx context.Context, data *ContentDataSourceModel, c *client.Client) ([]client.ContentResult, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch {
	case !data.AppStoreURLs.IsNull():
		return executeLookupAppStoreURLs(ctx, data, c)

	case !data.IDs.IsNull():
		return executeLookupIDs(ctx, data, c)

	case !data.AMGArtistIDs.IsNull():
		return executeLookupInt64Field(ctx, data.AMGArtistIDs, data, c, func(req *client.LookupRequest, batch []int64) {
			req.AMGArtistIDs = batch
		}, false)

	case !data.AMGAlbumIDs.IsNull():
		return executeLookupInt64Field(ctx, data.AMGAlbumIDs, data, c, func(req *client.LookupRequest, batch []int64) {
			req.AMGAlbumIDs = batch
		}, false)

	case !data.AMGVideoIDs.IsNull():
		return executeLookupInt64Field(ctx, data.AMGVideoIDs, data, c, func(req *client.LookupRequest, batch []int64) {
			req.AMGVideoIDs = batch
		}, false)

	case !data.UPCs.IsNull():
		return executeLookupStringField(ctx, data.UPCs, data, c, func(req *client.LookupRequest, batch []string) {
			req.UPCs = batch
		})

	case !data.ISBNs.IsNull():
		return executeLookupStringField(ctx, data.ISBNs, data, c, func(req *client.LookupRequest, batch []string) {
			req.ISBNs = batch
		})

	case !data.BundleIDs.IsNull():
		return executeLookupStringField(ctx, data.BundleIDs, data, c, func(req *client.LookupRequest, batch []string) {
			req.BundleIDs = batch
		})
	}

	diags.AddError("No Selector", "No lookup selector was provided.")
	return nil, diags
}

// executeLookupAppStoreURLs handles lookup requests using App Store URLs.
func executeLookupAppStoreURLs(ctx context.Context, data *ContentDataSourceModel, c *client.Client) ([]client.ContentResult, diag.Diagnostics) {
	var diags diag.Diagnostics
	var urls []string
	diags.Append(data.AppStoreURLs.ElementsAs(ctx, &urls, false)...)
	if diags.HasError() {
		return nil, diags
	}

	var trackIDs []int64
	countryCode := ""

	for _, urlStr := range urls {
		trackID, country := parseAppStoreURL(urlStr)
		if countryCode == "" {
			countryCode = country
		} else if countryCode != country {
			diags.AddError("Inconsistent Countries", "All App Store URLs must be from the same country store.")
			return nil, diags
		}
		trackIDs = append(trackIDs, trackID)
	}

	data.Country = types.StringValue(countryCode)
	baseRequest := buildLookupRequest(*data)

	var results []client.ContentResult
	var allMissingURLs []string

	batches := common.ChunkInt64(trackIDs, common.MaxLookupBatchSize)
	for _, batch := range batches {
		req := baseRequest
		req.IDs = batch
		req.Limit = lookupLimitForBatch(data.Limit, len(batch), true)

		result, err := c.Lookup(ctx, req)
		if err != nil {
			if notFoundErr, ok := err.(*client.NotFoundError); ok {
				for _, id := range notFoundErr.MissingIDs {
					for _, url := range urls {
						if strings.Contains(url, fmt.Sprintf("id%d", id)) {
							allMissingURLs = append(allMissingURLs, url)
						}
					}
				}
				if result != nil {
					results = append(results, result.Results...)
				}
				continue
			}
			diags.AddError("API Request Failed", err.Error())
			return nil, diags
		}
		results = append(results, result.Results...)
	}

	if len(allMissingURLs) > 0 {
		diags.AddError("Some URLs not found", fmt.Sprintf("The following URLs were not found: %v", allMissingURLs))
		return nil, diags
	}

	return results, diags
}

// executeLookupIDs handles lookup requests using iTunes track IDs.
func executeLookupIDs(ctx context.Context, data *ContentDataSourceModel, c *client.Client) ([]client.ContentResult, diag.Diagnostics) {
	var diags diag.Diagnostics
	var ids []int64
	diags.Append(data.IDs.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return nil, diags
	}

	var results []client.ContentResult
	var allMissingIDs []int64
	baseRequest := buildLookupRequest(*data)

	batches := common.ChunkInt64(ids, common.MaxLookupBatchSize)
	for _, batch := range batches {
		req := baseRequest
		req.IDs = batch
		req.Limit = lookupLimitForBatch(data.Limit, len(batch), true)

		result, err := c.Lookup(ctx, req)
		if err != nil {
			if notFoundErr, ok := err.(*client.NotFoundError); ok {
				allMissingIDs = append(allMissingIDs, notFoundErr.MissingIDs...)
				if result != nil {
					results = append(results, result.Results...)
				}
				continue
			}
			diags.AddError("API Request Failed", err.Error())
			return nil, diags
		}
		results = append(results, result.Results...)
	}

	if len(allMissingIDs) > 0 {
		diags.AddError("Resources Not Found", fmt.Sprintf("The following IDs were not found: %v", allMissingIDs))
		return nil, diags
	}

	return results, diags
}

// executeLookupInt64Field handles lookup requests for int64 selector fields
// (AMG artist/album/video IDs) using the provided setter to populate the request.
func executeLookupInt64Field(
	ctx context.Context,
	field types.List,
	data *ContentDataSourceModel,
	c *client.Client,
	setter func(req *client.LookupRequest, batch []int64),
	autoAlign bool,
) ([]client.ContentResult, diag.Diagnostics) {
	var diags diag.Diagnostics
	var ids []int64
	diags.Append(field.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return nil, diags
	}

	var results []client.ContentResult
	baseRequest := buildLookupRequest(*data)

	batches := common.ChunkInt64(ids, common.MaxLookupBatchSize)
	for _, batch := range batches {
		req := baseRequest
		setter(&req, batch)
		req.Limit = lookupLimitForBatch(data.Limit, len(batch), autoAlign)

		result, err := c.Lookup(ctx, req)
		if err != nil {
			diags.AddError("API Request Failed", err.Error())
			return nil, diags
		}
		results = append(results, result.Results...)
	}

	return results, diags
}

// executeLookupStringField handles lookup requests for string selector fields
// (UPCs, ISBNs, bundle IDs) using the provided setter to populate the request.
func executeLookupStringField(
	ctx context.Context,
	field types.List,
	data *ContentDataSourceModel,
	c *client.Client,
	setter func(req *client.LookupRequest, batch []string),
) ([]client.ContentResult, diag.Diagnostics) {
	var diags diag.Diagnostics
	var values []string
	diags.Append(field.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return nil, diags
	}

	var results []client.ContentResult
	baseRequest := buildLookupRequest(*data)

	batches := common.ChunkStrings(values, common.MaxLookupBatchSize)
	for _, batch := range batches {
		req := baseRequest
		setter(&req, batch)
		req.Limit = lookupLimitForBatch(data.Limit, len(batch), false)

		result, err := c.Lookup(ctx, req)
		if err != nil {
			diags.AddError("API Request Failed", err.Error())
			return nil, diags
		}
		results = append(results, result.Results...)
	}

	return results, diags
}

// executeSearch performs a search request using the term and optional parameters
// from the data model.
func executeSearch(ctx context.Context, data ContentDataSourceModel, c *client.Client) ([]client.ContentResult, diag.Diagnostics) {
	var diags diag.Diagnostics

	searchReq := client.SearchRequest{
		Term:      data.Term.ValueString(),
		Media:     common.StringValue(data.Media),
		Entity:    common.StringValue(data.Entity),
		Country:   common.StringValue(data.Country),
		Attribute: common.StringValue(data.Attribute),
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

	result, err := c.Search(ctx, searchReq)
	if err != nil {
		diags.AddError("API Request Failed", err.Error())
		return nil, diags
	}

	return result.Results, diags
}

// mapResultsToModel converts API content results to Terraform model objects,
// downloading and encoding artwork images.
func mapResultsToModel(ctx context.Context, results []client.ContentResult) []ContentResultModel {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	var resultItems []ContentResultModel

	for _, result := range results {
		artworkURL := result.ArtworkURL
		if artworkURL != "" && strings.HasSuffix(artworkURL, ".jpg") {
			artworkURL = strings.TrimSuffix(artworkURL, ".jpg") + ".png"
		}

		var artworkBase64 string
		if artworkURL != "" {
			encoded, err := downloadAndEncodeImage(ctx, httpClient, artworkURL)
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

	return resultItems
}
