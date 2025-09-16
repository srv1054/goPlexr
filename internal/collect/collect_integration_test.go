package collect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goduper/internal/opts"
	"goduper/internal/plex"
)

// minimal XML fixtures for endpoints
const sectionsXML = `<?xml version="1.0"?>
<MediaContainer>
  <Directory key="1" type="movie" title="Movies" />
  <Directory key="2" type="show" title="Shows" />
</MediaContainer>`

// one duplicate item in section 1 (two media entries)
const duplicatesXML = `<?xml version="1.0"?>
<MediaContainer>
  <Video ratingKey="100" key="/library/metadata/100" title="Dup Movie" year="2020">
    <Media id="m1" videoResolution="1080" container="mkv" width="1920" height="1080">
      <Part id="p1" file="/path/dup1.mkv" size="1000" exists="1" accessible="1" />
    </Media>
    <Media id="m2" videoResolution="2160" container="mkv" width="3840" height="1600">
      <Part id="p2" file="/path/dup2.mkv" size="2000" exists="1" accessible="1" />
    </Media>
  </Video>
</MediaContainer>`

// detailed metadata for ratingKey 100 (same as duplicates, with checkFiles info)
const metadata100XML = `<?xml version="1.0"?>
<MediaContainer>
  <Video ratingKey="100" key="/library/metadata/100" title="Dup Movie" year="2020">
    <Media id="m1" videoResolution="1080" container="mkv" width="1920" height="1080">
      <Part id="p1" file="/path/dup1.mkv" size="1000" exists="1" accessible="1" />
    </Media>
    <Media id="m2" videoResolution="2160" container="mkv" width="3840" height="1600">
      <Part id="p2" file="/path/dup2.mkv" size="2000" exists="1" accessible="1" />
    </Media>
  </Video>
</MediaContainer>`

func TestCollectRun_WithMockPlex(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/library/sections", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(sectionsXML))
	})

	mux.HandleFunc("/library/sections/1/all", func(w http.ResponseWriter, r *http.Request) {
		// duplicate=1
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(duplicatesXML))
	})

	mux.HandleFunc("/library/metadata/100", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(metadata100XML))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// create options and client pointing at the test server
	o := opts.Options{
		BaseURL:      ts.URL,
		Token:        "fake",
		Deep:         true,
		Verify:       true,
		IncludeShows: false,
		DupPolicy:    "ignore-4k-1080",
		Timeout:      5 * time.Second,
	}

	pc, err := plex.NewClient(o)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	out, err := Run(context.Background(), pc, o)
	if err != nil {
		t.Fatalf("collect.Run error: %v", err)
	}

	// Expect one section scanned and (depending on policy) possibly 0 duplicates because 4k+1080 pair is ignored
	if len(out.Sections) == 0 {
		t.Fatalf("expected at least one section in output")
	}

	// Because default policy ignores exact 4k+1080 pair, the duplicate count should be 0 and the item should appear in Ignored
	if out.TotalItems != 0 {
		t.Fatalf("expected TotalItems 0 but got %d", out.TotalItems)
	}
	if len(out.Ignored) == 0 {
		t.Fatalf("expected ignored items to be present for 4k+1080 pair")
	}

	// verify the ignored item matches ratingKey
	if out.Ignored[0].Item.RatingKey != "100" {
		t.Fatalf("ignored item ratingKey mismatch: got %q", out.Ignored[0].Item.RatingKey)
	}

	// verify summary flags
	if out.Summary.DuplicatePolicy != "ignore-4k-1080" {
		t.Fatalf("expected duplicate policy to be preserved in summary")
	}
}
