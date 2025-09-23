package main

import "testing"

// Test various basenames to see if they are correctly identified as extras or not.
func TestIsExtraBasename(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// Should match (suffix tokens, any case, with/without ext, optional index)
		{"My Movies (1993)-featurette.mkv", true},
		{"My Movies-deleted.mkv", true},
		{"My Movies-featurette", true},
		{"My Movies-interview.mov", true},
		{"My Movies-scene.mkv", true},
		{"My Movies-short.mkv", true},
		{"My Movies-trailer.mkv", true},
		{"My Movies-other.mkv", true},
		{"My Movies (1993)-featurette", true},
		{"My Movies-TRAILER", true},
		{"My Movies-trailer-2.mkv", true},
		{"My Movies-trailer_02", true},
		{"My Movies-trailer.3", true},
		{"My Movies-TRaiLeR.mkv", true},

		// Should NOT match
		{"Some -other freakin movie.mkv", false}, // extra words after "-other"
		{"My Movies trailer.mkv", false},         // no hyphen before token
		{"My Movies-trailerized.mkv", false},     // token embedded in a larger word
		{"My Movies-othe.mkv", false},            // partial token
		{"My Movies-trailer cut.mkv", false},     // extra word after token
		{"My Movies are awesome-behindthescenes but maybe its not-trailer.mkv", true},
	}

	for _, tc := range cases {
		if got := isExtraBasename(tc.in); got != tc.want {
			t.Errorf("isExtraBasename(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// Test various paths to see if they are correctly identified as extras or not.
func TestIsExtraPath(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// ✅ extras by folder name
		{"/media/Movies/Extras/My.mkv", true},
		{"/mnt/Movies/Featurettes/Foo.mp4", true},
		{"/mnt/Movies/Behind-the-Scenes/Foo.mkv", true},
		{"/mnt/Movies/Deleted Scenes/Foo.mkv", true},
		{"/mnt/Movies/trailers/foo.mkv", true},
		{"/mnt/Movies/other/foo.mkv", true},

		// ✅ extras by basename suffix in a normal folder
		{"/media/Movies/My Movies (2019)-featurette.mkv", true},
		{"/media/Movies/My Movies-trailer", true},

		// ❌ regular movies
		{"/media/Movies/My Movie (2020).mkv", false},
		{"/media/Movies/Some -other freakin movie.mkv", false},
	}

	for _, tc := range cases {
		if got := isExtraPath(tc.in); got != tc.want {
			t.Errorf("isExtraPath(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
