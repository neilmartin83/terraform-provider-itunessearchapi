//go:build acceptance

package client

import (
	"context"
	"testing"
)

func TestAccLookup_ByBundleID(t *testing.T) {
	c := NewClient()
	result, err := c.Lookup(context.Background(), LookupRequest{
		BundleIDs: []string{"com.apple.Pages"},
		Country:   "us",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected at least 1 result for com.apple.Pages")
	}
	if result.Results[0].BundleID != "com.apple.Pages" {
		t.Errorf("expected bundle ID %q, got %q", "com.apple.Pages", result.Results[0].BundleID)
	}
}

func TestAccLookup_ByMultipleBundleIDs(t *testing.T) {
	c := NewClient()
	result, err := c.Lookup(context.Background(), LookupRequest{
		BundleIDs: []string{"com.apple.Pages", "com.apple.Keynote"},
		Country:   "us",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(result.Results))
	}

	foundPages := false
	foundKeynote := false
	for _, r := range result.Results {
		if r.BundleID == "com.apple.Pages" {
			foundPages = true
		}
		if r.BundleID == "com.apple.Keynote" {
			foundKeynote = true
		}
	}
	if !foundPages {
		t.Error("Pages not found in results")
	}
	if !foundKeynote {
		t.Error("Keynote not found in results")
	}
}

func TestAccLookup_ByID(t *testing.T) {
	c := NewClient()

	pagesResult, err := c.Lookup(context.Background(), LookupRequest{
		BundleIDs: []string{"com.apple.Pages"},
		Country:   "us",
	})
	if err != nil {
		t.Fatalf("setup: failed to resolve Pages track ID: %v", err)
	}
	if len(pagesResult.Results) == 0 {
		t.Fatal("setup: no results for Pages bundle ID")
	}
	trackID := pagesResult.Results[0].TrackID

	result, err := c.Lookup(context.Background(), LookupRequest{
		IDs:     []int64{trackID},
		Country: "us",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].TrackID != trackID {
		t.Errorf("expected track ID %d, got %d", trackID, result.Results[0].TrackID)
	}
}

func TestAccLookup_NotFoundID(t *testing.T) {
	c := NewClient()
	_, err := c.Lookup(context.Background(), LookupRequest{
		IDs:     []int64{9999999999},
		Country: "us",
	})
	if err == nil {
		t.Fatal("expected NotFoundError for bogus ID")
	}
	if _, ok := err.(*NotFoundError); !ok {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
}

func TestAccLookup_WithCountry(t *testing.T) {
	c := NewClient()
	result, err := c.Lookup(context.Background(), LookupRequest{
		BundleIDs: []string{"com.apple.Pages"},
		Country:   "gb",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected at least 1 result for Pages in GB store")
	}
}

func TestAccLookup_WithEntityFilter(t *testing.T) {
	c := NewClient()
	result, err := c.Lookup(context.Background(), LookupRequest{
		BundleIDs: []string{"com.apple.Pages"},
		Country:   "us",
		Entity:    "software",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected at least 1 result with entity filter")
	}
}

func TestAccSearch_Basic(t *testing.T) {
	c := NewClient()
	result, err := c.Search(context.Background(), SearchRequest{
		Term:    "Pages",
		Media:   "software",
		Country: "us",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected at least 1 result for 'Pages' search")
	}
	if len(result.Results) > 5 {
		t.Errorf("expected at most 5 results, got %d", len(result.Results))
	}
}

func TestAccSearch_WithLimit(t *testing.T) {
	c := NewClient()
	result, err := c.Search(context.Background(), SearchRequest{
		Term:    "Apple",
		Media:   "software",
		Country: "us",
		Limit:   3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) > 3 {
		t.Errorf("expected at most 3 results, got %d", len(result.Results))
	}
}

func TestAccSearch_DifferentMedia(t *testing.T) {
	c := NewClient()
	result, err := c.Search(context.Background(), SearchRequest{
		Term:    "Beatles",
		Media:   "music",
		Country: "us",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected at least 1 result for 'Beatles' music search")
	}
}

func TestAccSearch_ResultFields(t *testing.T) {
	c := NewClient()
	result, err := c.Search(context.Background(), SearchRequest{
		Term:    "Pages",
		Media:   "software",
		Country: "us",
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected at least 1 result")
	}

	r := result.Results[0]
	if r.TrackID == 0 {
		t.Error("expected non-zero TrackID")
	}
	if r.TrackName == "" {
		t.Error("expected non-empty TrackName")
	}
	if r.TrackViewURL == "" {
		t.Error("expected non-empty TrackViewURL")
	}
}
