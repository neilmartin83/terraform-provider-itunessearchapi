package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// Search performs a search against the iTunes Search API with the provided parameters.
func (c *Client) Search(ctx context.Context, req SearchRequest) (*ContentResponse, error) {
	query := url.Values{}
	query.Set("term", req.Term)
	if req.Media != "" {
		query.Set("media", req.Media)
	} else {
		query.Set("media", "all")
	}

	c.addCommonParameters(query, req.Entity, req.Country, req.Limit)

	if req.Attribute != "" {
		query.Set("attribute", req.Attribute)
	}
	if req.Lang != "" {
		query.Set("lang", req.Lang)
	}
	if req.Version > 0 {
		query.Set("version", fmt.Sprintf("%d", req.Version))
	}
	if req.Explicit != nil {
		if *req.Explicit {
			query.Set("explicit", "Yes")
		} else {
			query.Set("explicit", "No")
		}
	}
	if req.Offset != nil {
		query.Set("offset", fmt.Sprintf("%d", *req.Offset))
	}
	if req.Callback != "" {
		query.Set("callback", req.Callback)
	}

	apiURL := fmt.Sprintf("%s/search?%s", c.baseURL, query.Encode())

	resp, err := c.doRequest(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	var decoder io.Reader = resp.Body
	if req.Callback != "" {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading callback response: %w", err)
		}
		payload, err := unwrapJSONPBody(bodyBytes, req.Callback)
		if err != nil {
			return nil, err
		}
		decoder = bytes.NewReader(payload)
	}

	var result ContentResponse
	if err := json.NewDecoder(decoder).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding API response: %w", err)
	}

	return &result, nil
}
