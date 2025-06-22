package client

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
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	rateLimiter *rate.Limiter
	apiClient   *http.Client
	imgClient   *http.Client
}

type Config struct {
	RequestsPerMinute int64
}

type APIError struct {
	StatusCode int
	Message    string
}

type NotFoundError struct {
	MissingIDs  []int64
	MissingURLs []string
}

type ContentResponse struct {
	Results []ContentResult `json:"results"`
}

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
)

// NewClient creates a new iTunes Search API client instance.
func NewClient() *Client {
	return &Client{}
}

// Configure sets up the client with the provided configuration.
// It initializes:
//   - Rate limiting (default 20 requests per minute if not specified)
//   - HTTP transport settings for both API and image clients
//   - Separate HTTP clients for API calls (10s timeout) and image downloads (30s timeout)
//
// The context parameter is currently unused but maintained for future API compatibility.
//
// Parameters:
//   - ctx: Context for future cancellation/timeout support
//   - config: Client configuration including rate limiting settings
//
// Returns an error if configuration fails, though current implementation always returns nil.
func (c *Client) Configure(ctx context.Context, config Config) error {
	rateLimit := int64(20)
	if config.RequestsPerMinute > 0 {
		rateLimit = config.RequestsPerMinute
	}

	c.rateLimiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(rateLimit)), 1)

	transport := &http.Transport{
		MaxIdleConns:       100,
		MaxConnsPerHost:    100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: false,
		DisableKeepAlives:  false,
	}

	c.apiClient = &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	c.imgClient = &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	return nil
}

// Error implements the error interface for APIError.
// It returns a formatted string containing the HTTP status code and error message.
//
// Returns:
//   - string: A formatted error message in the format "API error (HTTP {status}): {message}"
func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// Error implements the error interface for NotFoundError.
// It returns a formatted string containing either missing IDs or URLs based on which field is populated.
//
// Returns:
//   - string: A formatted error message listing either missing IDs or URLs
//
// The function prioritizes MissingIDs over MissingURLs if both are present.
func (e *NotFoundError) Error() string {
	if len(e.MissingIDs) > 0 {
		return fmt.Sprintf("The following IDs were not found: %v", e.MissingIDs)
	}
	return fmt.Sprintf("The following URLs were not found: %v", e.MissingURLs)
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

	c.AddCommonParameters(query, entity, country, limit)

	apiURL := fmt.Sprintf("https://itunes.apple.com/search?%s", query.Encode())

	resp, err := c.DoRateLimitedRequest(ctx, apiURL)
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

// DoRateLimitedRequest performs a rate limited HTTP GET request to the specified URL.
func (c *Client) DoRateLimitedRequest(ctx context.Context, url string) (*http.Response, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait error: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("User-Agent", "Terraform-Provider-iTunesSearchAPI")
	req.Header.Add("Accept", "application/json")

	resp, err := c.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
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

// downloadImage downloads an image from the specified URL and returns its byte content.
func (c *Client) downloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("User-Agent", "Terraform-Provider-iTunesSearchAPI")
	req.Header.Add("Accept", "image/*")

	resp, err := c.imgClient.Do(req)
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

// DownloadAndEncodeImage downloads an image from a URL and returns it as a base64 encoded string.
func (c *Client) DownloadAndEncodeImage(ctx context.Context, imageURL string) (string, error) {
	imageData, err := c.downloadImage(ctx, imageURL)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(imageData), nil
}

// getString retrieves a string value from a map by key, converting it to a string if necessary.
func (c *Client) GetString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// getFloat64 retrieves a float64 value from a map by key, handling both float64 and int types.
func (c *Client) GetFloat64(m map[string]interface{}, key string) float64 {
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
func (c *Client) GetInt64(m map[string]interface{}, key string) int64 {
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
func (c *Client) GetStringList(m map[string]interface{}, key string) []string {
	var list []string
	if v, ok := m[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			for _, elem := range arr {
				list = append(list, fmt.Sprintf("%v", elem))
			}
		}
	}
	return list
}

// ParseAppStoreURL parses an App Store URL and extracts the track ID and country code.
func (c *Client) ParseAppStoreURL(urlStr string) (trackID int64, countryCode string, err error) {
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

// AddCommonParameters adds common query parameters to the URL query.
func (c *Client) AddCommonParameters(query url.Values, entity, country string, limit int64) {
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

// StringifyIDs converts a slice of int64 to a slice of strings.
func (c *Client) StringifyIDs(ids []int64) []string {
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = strconv.FormatInt(id, 10)
	}
	return strIDs
}

// ChunkIDs splits a slice of IDs into batches of maxLookupBatchSize.
func (c *Client) ChunkIDs(ids []int64) [][]int64 {
	var batches [][]int64
	for i := 0; i < len(ids); i += maxLookupBatchSize {
		end := i + maxLookupBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batches = append(batches, ids[i:end])
	}
	return batches
}

// ProcessBatch processes a batch of IDs and returns the results.
func (c *Client) ProcessBatch(ctx context.Context, ids []int64, entity, country string, limit int64) (*ContentResponse, error) {
	query := url.Values{}
	query.Set("id", strings.Join(c.StringifyIDs(ids), ","))
	query.Set("limit", strconv.Itoa(maxLookupBatchSize))
	c.AddCommonParameters(query, entity, country, limit)

	apiURL := fmt.Sprintf("https://itunes.apple.com/lookup?%s", query.Encode())

	resp, err := c.DoRateLimitedRequest(ctx, apiURL)
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
