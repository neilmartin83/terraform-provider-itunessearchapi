package itunessearchapi

import (
	"context"
	"fmt"
	"net/http"
)

// doRateLimitedRequest performs a rate limited HTTP GET request to the specified URL.
func (p *iTunesProvider) doRateLimitedRequest(ctx context.Context, url string) (*http.Response, error) {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait error: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("User-Agent", "Terraform-Provider-iTunesSearchAPI")
	req.Header.Add("Accept", "application/json")

	resp, err := p.apiClient.Do(req)
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
