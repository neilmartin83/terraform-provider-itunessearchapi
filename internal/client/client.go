package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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
	rateLimit := int64(20) // default value
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

// DownloadAndEncodeImage downloads an image from a URL and returns it as a base64 encoded string
func (c *Client) DownloadAndEncodeImage(ctx context.Context, imageURL string) (string, error) {
	imageData, err := c.downloadImage(ctx, imageURL)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(imageData), nil
}
