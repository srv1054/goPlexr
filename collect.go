package main

import (
	"context"
	"strings"
	"unicode"
)

// Run performs the main collection logic and returns the results.
func RunCollection(ctx context.Context, pc *Client, o Options) (Output, error) {
	// --- discover sections ---
	var (
		sections []Directory
		err      error
	)

	if o.SectionsCSV != "" {
		for _, s := range strings.Split(o.SectionsCSV, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			sections = append(sections, Directory{
				Key:   s,
				Type:  "movie",
				Title: "(manual " + s + ")",
			})
		}
	} else {
		sections, err = pc.DiscoverSections(ctx, o.IncludeShows)
		if err != nil {
			return Output{}, err
		}
		if len(sections) == 0 {
			return Output{}, ErrNoSections
		}
	}

	// --- collect and summarize ---
	out := Output{
		Server: pc.BaseURL(),
	}
	ignored := make([]IgnoredItem, 0)

	totalItems := 0
	totalVersions := 0
	totalGhosts := 0
	totalVariantsExcluded := 0

	var libSummaries []LibrarySummary

	for _, sec := range sections {
		vids, err := pc.FetchDuplicatesForSection(ctx, sec.Key)
		if err != nil {
			// Skip this library on error; continue with others
			continue
		}

		sectionRes := SectionResult{
			SectionID:    sec.Key,
			SectionTitle: sec.Title,
			Type:         sec.Type,
		}

		secGhostParts := 0
		secItemsWithGhosts := 0
		secTotalVersions := 0
		secVariantsExcluded := 0

		for _, v := range vids {
			// deep fetch for parts and verification flags (if enabled)
			var vv *Video
			if o.Deep {
				vv, err = pc.DeepFetchItem(ctx, v.RatingKey, o.Verify)
				if err != nil {
					vv = &v
				}
			} else {
				vv = &v
			}

			item := Item{
				RatingKey: vv.RatingKey,
				Title:     fallback(vv.Title, v.Title),
				Year:      vv.Year,
				Guid:      vv.Guid,
			}

			itemGhosts := 0

			for _, m := range vv.Media {
				ver := Version{
					ID:              m.ID,
					Container:       m.Container,
					VideoCodec:      m.VideoCodec,
					AudioCodec:      m.AudioCodec,
					VideoResolution: m.VideoResolution,
					Bitrate:         m.Bitrate,
					Width:           m.Width,
					Height:          m.Height,
				}

				// build parts first, and detect if this entire version is in an Extras folder
				versionGhosts := 0
				versionIsExtra := false
				for _, p := range m.Part {
					if o.IgnoreExtras && isExtraPath(p.File) {
						versionIsExtra = true
					}

					exists := p.ExistsInt == 1
					accessible := p.AccessibleInt == 1
					verified := exists && accessible

					ver.Parts = append(ver.Parts, PartOut{
						ID:             p.ID,
						File:           p.File,
						Size:           p.Size,
						Duration:       p.Duration,
						VerifiedOnDisk: verified,
						Exists:         exists,
						Accessible:     accessible,
					})

					if o.Verify && !verified {
						versionGhosts++
					}
				}

				// If ignoring extras and this version lives under Extras/Featurettes/... — drop it entirely.
				if o.IgnoreExtras && versionIsExtra {
					continue
				}

				itemGhosts += versionGhosts
				item.Versions = append(item.Versions, ver)
			}

			// If extras filtering left fewer than 2 versions, it's no longer a duplicate.
			if len(item.Versions) < 2 {
				secVariantsExcluded++
				continue
			}

			// Ignore EXACT 4K+1080 pair (only that case)
			if shouldExcludeAs4k1080Pair(item, o.DupPolicy) {
				secVariantsExcluded++
				ignored = append(ignored, IgnoredItem{
					SectionID:    sec.Key,
					SectionTitle: sec.Title,
					Reason:       "4k+1080_pair",
					Item:         item,
				})
				continue
			}

			// Count only kept items
			secTotalVersions += len(item.Versions)
			if itemGhosts > 0 {
				secItemsWithGhosts++
			}
			secGhostParts += itemGhosts
			sectionRes.Items = append(sectionRes.Items, item)
		}

		libSummaries = append(libSummaries, LibrarySummary{
			SectionID:        sec.Key,
			SectionTitle:     sec.Title,
			Type:             sec.Type,
			DuplicateItems:   len(sectionRes.Items),
			TotalVersions:    secTotalVersions,
			GhostParts:       secGhostParts,
			ItemsWithGhosts:  secItemsWithGhosts,
			VariantsExcluded: secVariantsExcluded,
		})

		totalItems += len(sectionRes.Items)
		totalVersions += secTotalVersions
		totalGhosts += secGhostParts
		totalVariantsExcluded += secVariantsExcluded

		out.Sections = append(out.Sections, sectionRes)
	}

	out.TotalItems = totalItems
	out.TotalVersions = totalVersions
	out.TotalGhosts = totalGhosts
	out.Summary = Summary{
		VerificationPerformed: o.Verify,
		TotalLibraries:        len(out.Sections),
		TotalDuplicateItems:   totalItems,
		TotalGhostParts:       totalGhosts,
		DuplicatePolicy:       o.DupPolicy,
		VariantItemsExcluded:  totalVariantsExcluded,
		Libraries:             libSummaries,
	}
	out.Ignored = ignored

	return out, nil
}

// ErrNoSections is returned when no movie/show sections are found.
var ErrNoSections = &noSectionsErr{}

type noSectionsErr struct{}

func (*noSectionsErr) Error() string { return "no movie/show sections found" }

// Only exclude when exactly one 4K and one 1080p version exist (no others).
func shouldExcludeAs4k1080Pair(it Item, policy string) bool {
	if strings.ToLower(policy) != "ignore-4k-1080" {
		return false // "plex" behavior: keep all multi-version items
	}

	counts := map[string]int{}
	total := 0
	for _, v := range it.Versions {
		key := normalizeResKey(v)
		counts[key]++
		total++
	}

	return total == 2 && counts["2160"] == 1 && counts["1080"] == 1
}

// Normalize resolution label to a key we can compare.
// (Make sure this matches 4K short-side tweak of 1580 to account for cinemascope aspect ratios etc)
func normalizeResKey(v Version) string {
	r := strings.ToLower(strings.TrimSpace(v.VideoResolution))
	switch {
	case r == "4k" || strings.Contains(r, "2160") || strings.Contains(r, "uhd"):
		return "2160"
	case strings.Contains(r, "1080"):
		return "1080"
	case strings.Contains(r, "720"):
		return "720"
	case r == "sd" || strings.Contains(r, "480"):
		return "480"
	}

	// Fallback by dimensions (long/short side)
	w, h := v.Width, v.Height
	if w < h {
		w, h = h, w
	}
	const (
		th4kLong    = 3200
		th4kShort   = 1580
		th1080Long  = 1700
		th1080Short = 900
		th720Long   = 1200
		th720Short  = 650
	)
	switch {
	case w >= th4kLong || h >= th4kShort:
		return "2160"
	case w >= th1080Long || h >= th1080Short:
		return "1080"
	case w >= th720Long || h >= th720Short:
		return "720"
	case h > 0 || w > 0:
		return "480"
	default:
		return "unknown"
	}
}

// Known Extra folder names (case-insensitive, per path segment)
// Plex standard:  https://support.plex.tv/articles/local-files-for-trailers-and-extras/
var extraDirNames = map[string]struct{}{
	"extras": {}, "featurettes": {}, "interviews": {}, "shorts": {}, "deleted": {},
	"deleted scenes": {}, "trailers": {}, "behind the scenes": {}, "other": {}, "scenes": {},
}

// Allowed Plex standardized tokens for filename suffix (case-insensitive)
// Plex standard:  https://support.plex.tv/articles/local-files-for-trailers-and-extras/
var extraTokens = map[string]struct{}{
	"behindthescenes": {}, "deleted": {}, "featurette": {}, "deletedscene": {}, "interviews": {},
	"interview": {}, "scene": {}, "short": {}, "trailer": {}, "other": {}, "deletedscenes": {},
}

// isExtraPath returns true if the path is inside an Extras-like folder OR
// the basename ends with a standardized Plex suffix (with/without extension).
func isExtraPath(p string) bool {
	if p == "" {
		return false
	}
	// Normalize separators to forward slashes
	s := strings.ReplaceAll(p, "\\", "/")
	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return false
	}

	// Folder-based detection on parent dirs
	for _, seg := range parts[:len(parts)-1] {
		seg = strings.TrimSpace(strings.ToLower(seg))
		if seg == "" {
			continue
		}
		if _, ok := extraDirNames[seg]; ok {
			return true
		}
	}

	// Filename-based detection (hyphen-suffix token), with or without extension
	base := parts[len(parts)-1] // keep extension; we handle both cases below
	return isExtraBasename(base)
}

// isExtraBasename checks only the basename (may include extension).
// Matches “…-token” or “…-token-<digits>” (also _, ., or space before digits), case-insensitive.
func isExtraBasename(name string) bool {
	if name == "" {
		return false
	}

	// Strip extension if present (optional; we accept with or without)
	if dot := strings.LastIndexByte(name, '.'); dot > 0 {
		name = name[:dot]
	}

	name = strings.TrimSpace(name)
	// Must have a hyphen introducing the token
	idx := strings.LastIndexByte(name, '-')
	if idx == -1 || idx == len(name)-1 {
		return false
	}

	suffix := strings.ToLower(strings.TrimSpace(name[idx+1:]))
	if suffix == "" {
		return false
	}

	// Exact token match
	if _, ok := extraTokens[suffix]; ok {
		return true
	}

	// Token followed by one separator and digits
	for tok := range extraTokens {
		if strings.HasPrefix(suffix, tok) {
			rest := strings.TrimSpace(suffix[len(tok):])
			if rest == "" {
				return true
			}
			// one leading separator then digits (e.g., -trailer-2, -trailer_01, -trailer 3, -trailer.4)
			if len(rest) >= 2 && (rest[0] == '-' || rest[0] == '_' || rest[0] == '.' || rest[0] == ' ') {
				digits := strings.TrimSpace(rest[1:])
				if allDigits(digits) {
					return true
				}
			}
		}
	}
	return false
}

// allDigits returns true if s is non-empty and consists only of digit runes.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// fallback returns a if it's not the zero value, otherwise b.
func fallback[T comparable](a, b T) T {
	var zero T
	if a != zero {
		return a
	}
	return b
}
