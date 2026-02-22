// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/common"
)

// Lookup issues a lookup request using the provided selectors and returns the results.
func (c *Client) Lookup(ctx context.Context, req LookupRequest) (*ContentResponse, error) {
	query := url.Values{}
	selectorSet := false

	if len(req.IDs) > 0 {
		query.Set("id", strings.Join(c.stringifyIDs(req.IDs), ","))
		selectorSet = true
	}
	if len(req.AMGArtistIDs) > 0 {
		query.Set("amgArtistId", strings.Join(c.stringifyIDs(req.AMGArtistIDs), ","))
		selectorSet = true
	}
	if len(req.AMGAlbumIDs) > 0 {
		query.Set("amgAlbumId", strings.Join(c.stringifyIDs(req.AMGAlbumIDs), ","))
		selectorSet = true
	}
	if len(req.AMGVideoIDs) > 0 {
		query.Set("amgVideoId", strings.Join(c.stringifyIDs(req.AMGVideoIDs), ","))
		selectorSet = true
	}
	if len(req.UPCs) > 0 {
		query.Set("upc", strings.Join(req.UPCs, ","))
		selectorSet = true
	}
	if len(req.ISBNs) > 0 {
		query.Set("isbn", strings.Join(req.ISBNs, ","))
		selectorSet = true
	}
	if len(req.BundleIDs) > 0 {
		query.Set("bundleId", strings.Join(req.BundleIDs, ","))
		selectorSet = true
	}

	if !selectorSet {
		return nil, fmt.Errorf("lookup requires at least one selector parameter")
	}

	limitToUse := req.Limit
	if limitToUse > common.MaxLookupBatchSize {
		limitToUse = common.MaxLookupBatchSize
	}

	c.addCommonParameters(query, req.Entity, req.Country, limitToUse)

	if req.Sort != "" {
		query.Set("sort", req.Sort)
	}

	apiURL := fmt.Sprintf("%s/lookup?%s", c.baseURL, query.Encode())

	resp, err := c.doRequest(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	var result ContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding API response: %w", err)
	}

	if len(req.IDs) == 0 {
		return &result, nil
	}

	foundIDs := make(map[int64]bool)
	for _, item := range result.Results {
		foundIDs[item.TrackID] = true
	}

	var missingIDs []int64
	for _, id := range req.IDs {
		if !foundIDs[id] {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) > 0 {
		return &result, &NotFoundError{MissingIDs: missingIDs}
	}

	return &result, nil
}
