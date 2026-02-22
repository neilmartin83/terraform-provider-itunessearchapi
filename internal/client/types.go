// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
)

// Logger defines the interface for logging HTTP requests and responses.
type Logger interface {
	LogRequest(ctx context.Context, method, url string, body []byte)
	LogResponse(ctx context.Context, statusCode int, headers http.Header, body []byte)
	LogAuth(ctx context.Context, message string, fields map[string]interface{})
}

// APIError represents an error response from the iTunes Search API.
type APIError struct {
	StatusCode int
	Message    string
}

// Error implements the error interface for APIError.
func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// NotFoundError represents an error when requested IDs or URLs are not found.
type NotFoundError struct {
	MissingIDs  []int64
	MissingURLs []string
}

// Error implements the error interface for NotFoundError.
func (e *NotFoundError) Error() string {
	if len(e.MissingIDs) > 0 {
		return fmt.Sprintf("The following IDs were not found: %v", e.MissingIDs)
	}
	return fmt.Sprintf("The following URLs were not found: %v", e.MissingURLs)
}

// ContentResponse represents the response envelope from the iTunes Search API.
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

// LookupRequest captures the supported query parameters for lookup operations.
type LookupRequest struct {
	IDs          []int64
	AMGArtistIDs []int64
	AMGAlbumIDs  []int64
	AMGVideoIDs  []int64
	UPCs         []string
	ISBNs        []string
	BundleIDs    []string
	Entity       string
	Country      string
	Limit        int64
	Sort         string
}

// SearchRequest captures the supported query parameters for search operations.
type SearchRequest struct {
	Term      string
	Media     string
	Entity    string
	Country   string
	Attribute string
	Limit     int64
	Lang      string
	Version   int64
	Explicit  *bool
	Offset    *int64
	Callback  string
}
