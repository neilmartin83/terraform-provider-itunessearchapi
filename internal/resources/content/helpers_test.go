package content

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseAppStoreURL(t *testing.T) {
	tests := []struct {
		url             string
		expectedID      int64
		expectedCountry string
	}{
		{
			url:             "https://apps.apple.com/us/app/pages/id361309726",
			expectedID:      361309726,
			expectedCountry: "us",
		},
		{
			url:             "https://apps.apple.com/gb/app/microsoft-word/id462054704",
			expectedID:      462054704,
			expectedCountry: "gb",
		},
		{
			url:             "https://apps.apple.com/de/app/some-app/id999",
			expectedID:      999,
			expectedCountry: "de",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			trackID, country := parseAppStoreURL(tt.url)
			if trackID != tt.expectedID {
				t.Errorf("expected track ID %d, got %d", tt.expectedID, trackID)
			}
			if country != tt.expectedCountry {
				t.Errorf("expected country %q, got %q", tt.expectedCountry, country)
			}
		})
	}
}

func TestLookupLimitForBatch_NullLimit(t *testing.T) {
	limit := types.Int64Null()
	if got := lookupLimitForBatch(limit, 50, false); got != 0 {
		t.Errorf("expected 0 for null limit without autoAlign, got %d", got)
	}
}

func TestLookupLimitForBatch_NullLimitAutoAlign(t *testing.T) {
	limit := types.Int64Null()
	if got := lookupLimitForBatch(limit, 50, true); got != 50 {
		t.Errorf("expected 50 for null limit with autoAlign, got %d", got)
	}
}

func TestLookupLimitForBatch_SetLimit(t *testing.T) {
	limit := types.Int64Value(10)
	if got := lookupLimitForBatch(limit, 50, true); got != 10 {
		t.Errorf("expected 10 for set limit, got %d", got)
	}
}

func TestLookupLimitForBatch_UnknownLimit(t *testing.T) {
	limit := types.Int64Unknown()
	if got := lookupLimitForBatch(limit, 100, false); got != 0 {
		t.Errorf("expected 0 for unknown limit without autoAlign, got %d", got)
	}
}

func TestBuildLookupRequest(t *testing.T) {
	data := ContentDataSourceModel{
		Entity:  types.StringValue("software"),
		Country: types.StringValue("us"),
		Sort:    types.StringValue("recent"),
	}

	req := buildLookupRequest(data)
	if req.Entity != "software" {
		t.Errorf("expected entity %q, got %q", "software", req.Entity)
	}
	if req.Country != "us" {
		t.Errorf("expected country %q, got %q", "us", req.Country)
	}
	if req.Sort != "recent" {
		t.Errorf("expected sort %q, got %q", "recent", req.Sort)
	}
}

func TestBuildLookupRequest_NullFields(t *testing.T) {
	data := ContentDataSourceModel{
		Entity:  types.StringNull(),
		Country: types.StringNull(),
		Sort:    types.StringNull(),
	}

	req := buildLookupRequest(data)
	if req.Entity != "" {
		t.Errorf("expected empty entity, got %q", req.Entity)
	}
	if req.Country != "" {
		t.Errorf("expected empty country, got %q", req.Country)
	}
	if req.Sort != "" {
		t.Errorf("expected empty sort, got %q", req.Sort)
	}
}

func TestDownloadAndEncodeImage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "fake-image-data")
	}))
	defer server.Close()

	httpClient := &http.Client{}
	encoded, err := downloadAndEncodeImage(context.Background(), httpClient, server.URL+"/image.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if encoded == "" {
		t.Error("expected non-empty base64 string")
	}
}

func TestDownloadAndEncodeImage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	httpClient := &http.Client{}
	_, err := downloadAndEncodeImage(context.Background(), httpClient, server.URL+"/missing.png")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}
