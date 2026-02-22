package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"results":[{"trackId":1,"trackName":"Test"}]}`)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	result, err := c.Search(context.Background(), SearchRequest{
		Term: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
}

func TestSearch_DefaultMedia(t *testing.T) {
	var receivedMedia string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMedia = r.URL.Query().Get("media")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"results":[]}`)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.Search(context.Background(), SearchRequest{
		Term: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedMedia != "all" {
		t.Errorf("expected default media %q, got %q", "all", receivedMedia)
	}
}

func TestSearch_WithParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("term") != "xcode" {
			t.Errorf("expected term %q, got %q", "xcode", q.Get("term"))
		}
		if q.Get("media") != "software" {
			t.Errorf("expected media %q, got %q", "software", q.Get("media"))
		}
		if q.Get("country") != "us" {
			t.Errorf("expected country %q, got %q", "us", q.Get("country"))
		}
		if q.Get("limit") != "5" {
			t.Errorf("expected limit %q, got %q", "5", q.Get("limit"))
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"results":[]}`)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.Search(context.Background(), SearchRequest{
		Term:    "xcode",
		Media:   "software",
		Country: "us",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_JSONPCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `myCallback({"results":[{"trackId":1}]});`)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	result, err := c.Search(context.Background(), SearchRequest{
		Term:     "test",
		Callback: "myCallback",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].TrackID != 1 {
		t.Errorf("expected track ID 1, got %d", result.Results[0].TrackID)
	}
}

func TestSearch_ExplicitParameter(t *testing.T) {
	var receivedExplicit string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedExplicit = r.URL.Query().Get("explicit")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"results":[]}`)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	explicit := false
	_, err := c.Search(context.Background(), SearchRequest{
		Term:     "test",
		Explicit: &explicit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedExplicit != "No" {
		t.Errorf("expected explicit %q, got %q", "No", receivedExplicit)
	}
}
