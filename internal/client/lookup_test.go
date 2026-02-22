// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLookup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"results":[{"trackId":123,"trackName":"Test App"}]}`)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	result, err := c.Lookup(context.Background(), LookupRequest{
		IDs: []int64{123},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].TrackName != "Test App" {
		t.Errorf("expected track name %q, got %q", "Test App", result.Results[0].TrackName)
	}
}

func TestLookup_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"results":[{"trackId":123,"trackName":"Test App"}]}`)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	result, err := c.Lookup(context.Background(), LookupRequest{
		IDs: []int64{123, 456},
	})
	if err == nil {
		t.Fatal("expected NotFoundError")
	}
	notFoundErr, ok := err.(*NotFoundError)
	if !ok {
		t.Fatalf("expected *NotFoundError, got %T", err)
	}
	if len(notFoundErr.MissingIDs) != 1 || notFoundErr.MissingIDs[0] != 456 {
		t.Errorf("expected missing ID 456, got %v", notFoundErr.MissingIDs)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result alongside error, got %d", len(result.Results))
	}
}

func TestLookup_NoSelector(t *testing.T) {
	c := NewClient()
	_, err := c.Lookup(context.Background(), LookupRequest{})
	if err == nil {
		t.Fatal("expected error for empty selector")
	}
}

func TestLookup_LimitCapped(t *testing.T) {
	var receivedLimit string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedLimit = r.URL.Query().Get("limit")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"results":[]}`)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, _ = c.Lookup(context.Background(), LookupRequest{
		BundleIDs: []string{"com.test"},
		Limit:     500,
	})
	if receivedLimit != "200" {
		t.Errorf("expected limit capped to 200, got %q", receivedLimit)
	}
}

func TestLookup_BundleIDs(t *testing.T) {
	var receivedBundleID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBundleID = r.URL.Query().Get("bundleId")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"results":[{"trackId":1,"bundleId":"com.example.app"}]}`)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	result, err := c.Lookup(context.Background(), LookupRequest{
		BundleIDs: []string{"com.example.app"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBundleID != "com.example.app" {
		t.Errorf("expected bundleId query param %q, got %q", "com.example.app", receivedBundleID)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}
