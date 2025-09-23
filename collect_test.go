package main

import (
	"testing"
)

func TestNormalizeResKey_ByResolution(t *testing.T) {
	cases := []struct {
		v    Version
		want string
	}{
		{Version{VideoResolution: "4K"}, "2160"},
		{Version{VideoResolution: "2160"}, "2160"},
		{Version{VideoResolution: "1080"}, "1080"},
		{Version{VideoResolution: "720"}, "720"},
		{Version{Width: 3840, Height: 1600}, "2160"},
		{Version{Width: 1920, Height: 1080}, "1080"},
		{Version{Width: 1280, Height: 720}, "720"},
	}
	for _, c := range cases {
		got := normalizeResKey(c.v)
		if got != c.want {
			t.Fatalf("normalizeResKey(%+v) = %q, want %q", c.v, got, c.want)
		}
	}
}

func TestShouldExcludeAs4k1080Pair(t *testing.T) {
	// item with 4K + 1080 only should be excluded for policy ignore-4k-1080
	it := Item{
		Versions: []Version{
			{VideoResolution: "2160"},
			{VideoResolution: "1080"},
		},
	}
	if !shouldExcludeAs4kHdPair(it, "ignore-4k-1080") {
		t.Fatalf("expected exclusion for exact 4k+1080 pair")
	}

	// if an extra version exists, do not exclude
	it.Versions = append(it.Versions, Version{VideoResolution: "720"})
	if shouldExcludeAs4kHdPair(it, "ignore-4k-1080") {
		t.Fatalf("did not expect exclusion when extra versions present")
	}

	// non-matching policy should not exclude
	it.Versions = []Version{{VideoResolution: "2160"}, {VideoResolution: "1080"}}
	if shouldExcludeAs4kHdPair(it, "plex") {
		t.Fatalf("did not expect exclusion under 'plex' policy")
	}
}

func TestShouldExcludeAs4kHdPair_Mislabeled720(t *testing.T) {
	item := Item{
		Versions: []Version{
			{Width: 3840, Height: 1600, VideoResolution: "4k"},
			{Width: 720, Height: 388, VideoResolution: "sd"}, // mislabel
		},
	}
	if !shouldExcludeAs4kHdPair(item, "ignore-4k-1080") {
		t.Fatalf("expected true for 4k + sd(720x388) pair")
	}

	// But 4k + 720x480 should NOT be treated as HD-ish
	itemBad := Item{
		Versions: []Version{
			{Width: 3840, Height: 1600, VideoResolution: "4k"},
			{Width: 720, Height: 480, VideoResolution: "sd"},
		},
	}
	if shouldExcludeAs4kHdPair(itemBad, "ignore-4k-1080") {
		t.Fatalf("expected false for 4k + sd(720x480)")
	}
}
