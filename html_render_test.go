package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderHTML_IgnoredSectionsAppear(t *testing.T) {
	out := Output{
		Server: "http://localhost:32400",
		Sections: []SectionResult{
			{SectionID: "1", SectionTitle: "Movies", Type: "movie"},
		},
		Summary: Summary{
			VerificationPerformed: true,
			TotalLibraries:        1,
			TotalDuplicateItems:   0,
			TotalGhostParts:       0,
			DuplicatePolicy:       "ignore-4k-1080",
			Libraries: []LibrarySummary{
				{SectionID: "1", SectionTitle: "Movies", Type: "movie"},
			},
		},
		Ignored: []IgnoredItem{
			{
				SectionID:    "1",
				SectionTitle: "Movies",
				Reason:       "extra_version",
				Item: Item{
					Title: "Foo",
					Year:  2020,
					Versions: []Version{
						{
							Container:       "mkv",
							VideoCodec:      "hevc",
							AudioCodec:      "aac",
							VideoResolution: "1080",
							Width:           1920, Height: 1080,
							Parts: []PartOut{{File: "/movies/Foo-trailer.mkv", Size: 1000, VerifiedOnDisk: true}},
						},
					},
				},
			},
			{
				SectionID:    "1",
				SectionTitle: "Movies",
				Reason:       "4k+1080_pair",
				Item: Item{
					Title: "Bar",
					Year:  2019,
					Versions: []Version{
						{
							Container:       "mkv",
							VideoCodec:      "hevc",
							AudioCodec:      "ac3",
							VideoResolution: "2160",
							Width:           3840, Height: 1600,
							Parts: []PartOut{{File: "/movies/Bar-4k.mkv", Size: 2000, VerifiedOnDisk: true}},
						},
						{
							Container:       "mkv",
							VideoCodec:      "h264",
							AudioCodec:      "aac",
							VideoResolution: "1080",
							Width:           1920, Height: 1080,
							Parts: []PartOut{{File: "/movies/Bar-1080.mkv", Size: 1500, VerifiedOnDisk: true}},
						},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	html1 := filepath.Join(dir, "with_extras.html")
	if err := RenderHTML(out, true, true, html1); err != nil {
		t.Fatalf("RenderHTML with extras=true failed: %v", err)
	}
	b1, _ := os.ReadFile(html1)
	s1 := string(b1)
	if !strings.Contains(s1, "Ignored Extras") {
		t.Errorf("expected 'Ignored Extras' section when ignoreExtras is true")
	}
	if !strings.Contains(s1, "Ignored (4K+1080 Pairs)") {
		t.Errorf("expected 'Ignored (4K+1080 Pairs)' section to be present")
	}

	// When ignoreExtras=false, the extras section should be hidden
	html2 := filepath.Join(dir, "no_extras.html")
	if err := RenderHTML(out, true, false, html2); err != nil {
		t.Fatalf("RenderHTML with extras=false failed: %v", err)
	}
	b2, _ := os.ReadFile(html2)
	s2 := string(b2)
	if strings.Contains(s2, "Ignored Extras") {
		t.Errorf("did not expect 'Ignored Extras' section when ignoreExtras is false")
	}
}
