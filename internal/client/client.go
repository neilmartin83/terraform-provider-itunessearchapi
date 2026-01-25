package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Logger is an interface for logging HTTP requests and responses
type Logger interface {
	LogRequest(ctx context.Context, method, url string, body []byte)
	LogResponse(ctx context.Context, statusCode int, headers http.Header, body []byte)
	LogAuth(ctx context.Context, message string, fields map[string]interface{})
}

// tokenBucket implements a simple token bucket rate limiter
type tokenBucket struct {
	tokens         float64
	maxTokens      float64
	refillRate     float64
	lastRefillTime time.Time
	mu             sync.Mutex
}

// newTokenBucket creates a new token bucket rate limiter
// maxRequests: maximum number of requests allowed
// perDuration: time window for the max requests (e.g., 1 minute)
func newTokenBucket(maxRequests int, perDuration time.Duration) *tokenBucket {
	refillRate := float64(maxRequests) / perDuration.Seconds()
	return &tokenBucket{
		tokens:         float64(maxRequests),
		maxTokens:      float64(maxRequests),
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

// take attempts to take a token from the bucket, blocking until one is available
func (tb *tokenBucket) take(ctx context.Context) error {
	for {
		tb.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(tb.lastRefillTime).Seconds()

		tb.tokens += elapsed * tb.refillRate
		if tb.tokens > tb.maxTokens {
			tb.tokens = tb.maxTokens
		}
		tb.lastRefillTime = now

		if tb.tokens >= 1.0 {
			tb.tokens -= 1.0
			tb.mu.Unlock()
			return nil
		}

		tokensNeeded := 1.0 - tb.tokens
		waitDuration := time.Duration(tokensNeeded / tb.refillRate * float64(time.Second))
		tb.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}
	}
}

// Client represents the iTunes Search API client.
type Client struct {
	apiClient   *http.Client
	logger      Logger
	rateLimiter *tokenBucket
}

// APIError represents an error response from the iTunes Search API.
type APIError struct {
	StatusCode int
	Message    string
}

// NotFoundError represents an error when requested IDs or URLs are not found.
type NotFoundError struct {
	MissingIDs  []int64
	MissingURLs []string
}

// ContentResponse represents the response from the iTunes Search API for content lookups.
type ContentResponse struct {
	Results []ContentResult `json:"results"`
}

// ContentResult represents a single content item returned by the iTunes Search API.
type ContentResult struct {
	TrackName        string   `json:"trackName"`
	BundleID         string   `json:"bundleId"`
	TrackID          int64    `json:"trackId"`
	SellerName       string   `json:"sellerName"`
	Kind             string   `json:"kind"`
	Description      string   `json:"description"`
	ReleaseDate      string   `json:"releaseDate"`
	Price            float64  `json:"price"`
	FormattedPrice   string   `json:"formattedPrice"`
	Currency         string   `json:"currency"`
	Version          string   `json:"version"`
	PrimaryGenre     string   `json:"primaryGenreName"`
	MinimumOSVersion string   `json:"minimumOsVersion"`
	FileSizeBytes    string   `json:"fileSizeBytes"`
	ArtistViewURL    string   `json:"artistViewUrl"`
	ArtworkURL       string   `json:"artworkUrl512"`
	TrackViewURL     string   `json:"trackViewUrl"`
	SupportedDevices []string `json:"supportedDevices"`
	Genres           []string `json:"genres"`
	Languages        []string `json:"languageCodesISO2A"`
	AverageRating    float64  `json:"averageUserRating"`
	RatingCount      int64    `json:"userRatingCount"`
}

const (
	maxLookupBatchSize = 200
	rateLimitRequests  = 20
	rateLimitDuration  = 1 * time.Minute
)

// NewClient creates a new iTunes Search API client instance.
func NewClient() *Client {
	return &Client{
		apiClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: newTokenBucket(rateLimitRequests, rateLimitDuration),
	}
}

// SetLogger sets the logger for the client
func (c *Client) SetLogger(logger Logger) {
	c.logger = logger
}

// Error implements the error interface for APIError.
func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// Error implements the error interface for NotFoundError.
func (e *NotFoundError) Error() string {
	if len(e.MissingIDs) > 0 {
		return fmt.Sprintf("The following IDs were not found: %v", e.MissingIDs)
	}
	return fmt.Sprintf("The following URLs were not found: %v", e.MissingURLs)
}

// Lookup looks up a batch of IDs and returns the results.
func (c *Client) Lookup(ctx context.Context, ids []int64, entity, country string, limit int64) (*ContentResponse, error) {
	query := url.Values{}
	query.Set("id", strings.Join(c.stringifyIDs(ids), ","))

	limitToUse := limit
	if limitToUse > maxLookupBatchSize {
		limitToUse = maxLookupBatchSize
	}

	c.addCommonParameters(query, entity, country, limitToUse)

	apiURL := fmt.Sprintf("https://itunes.apple.com/lookup?%s", query.Encode())

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

	foundIDs := make(map[int64]bool)
	for _, item := range result.Results {
		foundIDs[item.TrackID] = true
	}

	var missingIDs []int64
	for _, id := range ids {
		if !foundIDs[id] {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) > 0 {
		return &result, &NotFoundError{MissingIDs: missingIDs}
	}

	return &result, nil
}

// Search performs a search against the iTunes Search API with the provided parameters.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - term: Search term to query (required)
//   - media: Media type to search (e.g., "software", "music"). Defaults to "all" if empty
//   - entity: The type of results to return, relative to the specified media type
//   - country: Two-letter country code for store-specific results
//   - limit: Maximum number of results to return
//
// Returns:
//   - *ContentResponse: Pointer to the parsed API response containing search results
//   - error: Any error encountered during the request or response processing
//
// The function performs a rate-limited HTTP GET request to the iTunes Search API,
// automatically handling response parsing and cleanup. It will return an error if:
//   - The rate limiter encounters an error
//   - The HTTP request fails
//   - The response cannot be decoded
//
// Example:
//
//	result, err := client.Search(ctx, "Xcode", "software", "", "US", 1)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) Search(ctx context.Context, term, media, entity, country string, limit int64) (*ContentResponse, error) {
	query := url.Values{}
	query.Set("term", term)
	if media != "" {
		query.Set("media", media)
	} else {
		query.Set("media", "all")
	}

	c.addCommonParameters(query, entity, country, limit)

	apiURL := fmt.Sprintf("https://itunes.apple.com/search?%s", query.Encode())

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

	return &result, nil
}

// doRequest performs an HTTP GET request to the specified URL.
func (c *Client) doRequest(ctx context.Context, url string) (*http.Response, error) {
	// Apply client-side rate limiting
	if err := c.rateLimiter.take(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Accept", "application/json")

	if c.logger != nil {
		c.logger.LogRequest(ctx, req.Method, req.URL.String(), nil)
	}

	for {
		resp, err := c.apiClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request: %v", err)
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			if c.logger != nil && resp.Body != nil {
				responseBody, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				if err := resp.Body.Close(); err != nil && c.logger != nil {
					c.logger.LogAuth(ctx, "Failed to close response body", map[string]interface{}{
						"error": err.Error(),
					})
				}
				c.logger.LogResponse(ctx, resp.StatusCode, resp.Header, responseBody)
				resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
			}

			if resp.StatusCode != http.StatusOK {
				if cerr := resp.Body.Close(); cerr != nil {
					return nil, fmt.Errorf("API request failed with status code: %d and error closing response body: %v",
						resp.StatusCode, cerr)
				}
				return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
			}
			return resp, nil
		}

		retryAfter := resp.Header.Get("Retry-After")

		if c.logger != nil {
			c.logger.LogResponse(ctx, resp.StatusCode, resp.Header, nil)
		}

		if err := resp.Body.Close(); err != nil && c.logger != nil {
			c.logger.LogAuth(ctx, "Failed to close response body", map[string]interface{}{
				"error": err.Error(),
			})
		}

		if retryAfter != "" {
			seconds, err := time.ParseDuration(retryAfter + "s")
			if err == nil {
				waitDuration := seconds + (1 * time.Second)
				if c.logger != nil {
					c.logger.LogAuth(ctx, "Rate limited, waiting before retry", map[string]interface{}{
						"retry_after_seconds": retryAfter,
						"wait_duration":       waitDuration.String(),
					})
				}
				time.Sleep(waitDuration)
				continue
			} else if c.logger != nil {
				c.logger.LogAuth(ctx, "Failed to parse Retry-After header", map[string]interface{}{
					"retry_after": retryAfter,
					"error":       err.Error(),
				})
			}
		}

		return nil, fmt.Errorf("received 429 Too Many Requests with no valid Retry-After header")
	}
}

// addCommonParameters adds common query parameters to the URL query.
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
