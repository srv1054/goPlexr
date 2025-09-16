package collect

import (
	"context"
	"strings"

	"goduper/internal/model"
	"goduper/internal/opts"
	"goduper/internal/plex"
)

func Run(ctx context.Context, pc *plex.Client, o opts.Options) (model.Output, error) {
	// --- discover sections ---
	var (
		sections []plex.Directory
		err      error
	)

	if o.SectionsCSV != "" {
		for _, s := range strings.Split(o.SectionsCSV, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			sections = append(sections, plex.Directory{
				Key:   s,
				Type:  "movie",
				Title: "(manual " + s + ")",
			})
		}
	} else {
		sections, err = pc.DiscoverSections(ctx, o.IncludeShows)
		if err != nil {
			return model.Output{}, err
		}
		if len(sections) == 0 {
			return model.Output{}, ErrNoSections
		}
	}

	// --- collect + summarize ---
	out := model.Output{
		Server: pc.BaseURL(),
	}

	totalItems := 0
	totalVersions := 0
	totalGhosts := 0
	totalVariantsExcluded := 0

	var libSummaries []model.LibrarySummary

	for _, sec := range sections {
		vids, err := pc.FetchDuplicatesForSection(ctx, sec.Key)
		if err != nil {
			// Skip this library if it errors; continue with others
			continue
		}

		sectionRes := model.SectionResult{
			SectionID:    sec.Key,
			SectionTitle: sec.Title,
			Type:         sec.Type,
		}

		secGhostParts := 0
		secItemsWithGhosts := 0
		secTotalVersions := 0
		secVariantsExcluded := 0

		for _, v := range vids {
			// deep fetch for parts + verification flags (if enabled)
			var vv *plex.Video
			if o.Deep {
				vv, err = pc.DeepFetchItem(ctx, v.RatingKey, o.Verify)
				if err != nil {
					vv = &v
				}
			} else {
				vv = &v
			}

			item := model.Item{
				RatingKey: vv.RatingKey,
				Title:     fallback(vv.Title, v.Title),
				Year:      vv.Year,
				Guid:      vv.Guid,
			}

			itemGhosts := 0
			itemVersions := 0

			for _, m := range vv.Media {
				ver := model.Version{
					ID:              m.ID,
					Container:       m.Container,
					VideoCodec:      m.VideoCodec,
					AudioCodec:      m.AudioCodec,
					VideoResolution: m.VideoResolution,
					Bitrate:         m.Bitrate,
					Width:           m.Width,
					Height:          m.Height,
				}
				itemVersions++

				for _, p := range m.Part {
					exists := p.ExistsInt == 1
					accessible := p.AccessibleInt == 1
					verified := exists && accessible

					ver.Parts = append(ver.Parts, model.PartOut{
						ID:             p.ID,
						File:           p.File,
						Size:           p.Size,
						Duration:       p.Duration,
						VerifiedOnDisk: verified,
						Exists:         exists,
						Accessible:     accessible,
					})

					if o.Verify && !verified {
						itemGhosts++
					}
				}

				item.Versions = append(item.Versions, ver)
			}

			// Policy: ignore EXACT 4K+1080 pair (and only that case)
			if shouldExcludeAs4k1080Pair(item, o.DupPolicy) {
				secVariantsExcluded++
				continue
			}

			secTotalVersions += itemVersions
			if itemGhosts > 0 {
				secItemsWithGhosts++
			}
			secGhostParts += itemGhosts
			sectionRes.Items = append(sectionRes.Items, item)
		}

		libSummaries = append(libSummaries, model.LibrarySummary{
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
	out.Summary = model.Summary{
		VerificationPerformed: o.Verify,
		TotalLibraries:        len(out.Sections),
		TotalDuplicateItems:   totalItems,
		TotalGhostParts:       totalGhosts,
		DuplicatePolicy:       o.DupPolicy,
		VariantItemsExcluded:  totalVariantsExcluded,
		Libraries:             libSummaries,
	}

	return out, nil
}

// ErrNoSections is returned when no movie/show sections are found.
var ErrNoSections = &noSectionsErr{}

type noSectionsErr struct{}

func (*noSectionsErr) Error() string { return "no movie/show sections found" }

// Only exclude when exactly one 4K and one 1080p version exist (no others).
func shouldExcludeAs4k1080Pair(it model.Item, policy string) bool {
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

// Normalize resolution label to a key we can compare (2160, 1080, 720, 480, unknown).
func normalizeResKey(v model.Version) string {
	r := strings.ToLower(strings.TrimSpace(v.VideoResolution))
	switch {
	case r == "4k" || strings.Contains(r, "2160"):
		return "2160"
	case strings.Contains(r, "1080"):
		return "1080"
	case strings.Contains(r, "720"):
		return "720"
	case r == "sd" || strings.Contains(r, "480"):
		return "480"
	}
	// Fallback to height
	switch {
	case v.Height >= 1580:
		return "2160"
	case v.Height >= 1000:
		return "1080"
	case v.Height >= 700:
		return "720"
	case v.Height > 0:
		return "480"
	default:
		return "unknown"
	}
}

func fallback[T comparable](a, b T) T {
	var zero T
	if a != zero {
		return a
	}
	return b
}
