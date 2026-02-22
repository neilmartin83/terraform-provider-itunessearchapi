// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

package content

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestContentDataSource_Metadata(t *testing.T) {
	ds := &ContentDataSource{}
	req := datasource.MetadataRequest{
		ProviderTypeName: "itunessearchapi",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	expected := "itunessearchapi_content"
	if resp.TypeName != expected {
		t.Errorf("expected type name %q, got %q", expected, resp.TypeName)
	}
}

func TestContentDataSource_Schema(t *testing.T) {
	ds := &ContentDataSource{}
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("expected non-nil schema attributes")
	}

	requiredAttrs := []string{
		"timeouts", "app_store_urls", "term", "ids",
		"amg_artist_ids", "amg_album_ids", "amg_video_ids",
		"upcs", "isbns", "bundle_ids", "country", "media",
		"entity", "sort", "attribute", "lang", "version",
		"explicit", "offset", "callback", "limit", "results",
	}

	for _, attr := range requiredAttrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("expected schema to contain attribute %q", attr)
		}
	}
}
