// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/common"
)

// DefaultBaseURL is the base URL for the iTunes Search API.
const DefaultBaseURL = "https://itunes.apple.com"

// Client represents the iTunes Search API client.
type Client struct {
	apiClient   *http.Client
	logger      Logger
	rateLimiter *tokenBucket
	baseURL     string
}

// NewClient creates a new iTunes Search API client instance.
func NewClient() *Client {
	return &Client{
		apiClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: newTokenBucket(),
		baseURL:     DefaultBaseURL,
	}
}

// SetLogger sets the logger for the client.
func (c *Client) SetLogger(logger Logger) {
	c.logger = logger
}

// setBaseURL overrides the base URL for testing purposes.
func (c *Client) setBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// doRequest performs a rate-limited HTTP GET request to the specified URL,
// retrying on HTTP 429 responses up to MaxRetries times.
func (c *Client) doRequest(ctx context.Context, url string) (*http.Response, error) {
	if err := c.rateLimiter.take(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Accept", "application/json")

	if c.logger != nil {
		c.logger.LogRequest(ctx, req.Method, req.URL.String(), nil)
	}

	retryCount := 0
	for {
		resp, err := c.apiClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request: %w", err)
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			if c.logger != nil && resp.Body != nil {
				responseBody, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				_ = resp.Body.Close()
				c.logger.LogResponse(ctx, resp.StatusCode, resp.Header, responseBody)
				resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
			}

			if resp.StatusCode != http.StatusOK {
				_ = resp.Body.Close()
				return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
			}
			return resp, nil
		}

		if c.logger != nil {
			c.logger.LogResponse(ctx, resp.StatusCode, resp.Header, nil)
		}

		retryAfter := resp.Header.Get("Retry-After")
		_ = resp.Body.Close()

		retryCount++
		if retryCount >= common.MaxRetries {
			return nil, fmt.Errorf("exceeded maximum retries (%d) for rate-limited requests", common.MaxRetries)
		}

		if retryAfter != "" {
			seconds, err := time.ParseDuration(retryAfter + "s")
			if err == nil {
				waitDuration := seconds + (1 * time.Second)
				if waitDuration > common.MaxRetryWait {
					waitDuration = common.MaxRetryWait
				}
				if c.logger != nil {
					c.logger.LogAuth(ctx, "Rate limited, waiting before retry", map[string]interface{}{
						"retry_after_seconds": retryAfter,
						"wait_duration":       waitDuration.String(),
						"retry_count":         retryCount,
					})
				}
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(waitDuration):
				}
				continue
			}
			if c.logger != nil {
				c.logger.LogAuth(ctx, "Failed to parse Retry-After header", map[string]interface{}{
					"retry_after": retryAfter,
					"error":       err.Error(),
				})
			}
		}

		return nil, fmt.Errorf("received 429 Too Many Requests with no valid Retry-After header")
	}
}

// addCommonParameters adds shared query parameters to the URL query values.
func (c *Client) addCommonParameters(query url.Values, entity, country string, limit int64) {
	if entity != "" {
		query.Set("entity", entity)
	}
	if country != "" {
		query.Set("country", country)
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
}

// stringifyIDs converts a slice of int64 to a slice of strings.
func (c *Client) stringifyIDs(ids []int64) []string {
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = strconv.FormatInt(id, 10)
	}
	return strIDs
}

// unwrapJSONPBody strips the callback wrapper from a JSONP response.
func unwrapJSONPBody(body []byte, callback string) ([]byte, error) {
	trimmed := bytes.TrimSpace(body)
	trimmed = bytes.TrimSuffix(trimmed, []byte(";"))
	prefix := []byte(callback + "(")
	suffix := []byte(")")
	if !bytes.HasPrefix(trimmed, prefix) || !bytes.HasSuffix(trimmed, suffix) {
		return nil, fmt.Errorf("unexpected JSONP response format")
	}
	return trimmed[len(prefix) : len(trimmed)-len(suffix)], nil
}
