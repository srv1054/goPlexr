package collect

import (
    "testing"

    "goduper/internal/model"
)

func TestNormalizeResKey_ByResolution(t *testing.T) {
    cases := []struct{
        v model.Version
        want string
    }{
        {model.Version{VideoResolution: "4K"}, "2160"},
        {model.Version{VideoResolution: "2160"}, "2160"},
        {model.Version{VideoResolution: "1080"}, "1080"},
        {model.Version{VideoResolution: "720"}, "720"},
        {model.Version{Width: 3840, Height: 1600}, "2160"},
        {model.Version{Width: 1920, Height: 1080}, "1080"},
        {model.Version{Width: 1280, Height: 720}, "720"},
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
    it := model.Item{
        Versions: []model.Version{
            {VideoResolution: "2160"},
            {VideoResolution: "1080"},
        },
    }
    if !shouldExcludeAs4k1080Pair(it, "ignore-4k-1080") {
        t.Fatalf("expected exclusion for exact 4k+1080 pair")
    }

    // if an extra version exists, do not exclude
    it.Versions = append(it.Versions, model.Version{VideoResolution: "720"})
    if shouldExcludeAs4k1080Pair(it, "ignore-4k-1080") {
        t.Fatalf("did not expect exclusion when extra versions present")
    }

    // non-matching policy should not exclude
    it.Versions = []model.Version{{VideoResolution: "2160"}, {VideoResolution: "1080"}}
    if shouldExcludeAs4k1080Pair(it, "plex") {
        t.Fatalf("did not expect exclusion under 'plex' policy")
    }
}
