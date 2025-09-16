package collect

import (
	"context"
	"strings"

	"goduper/internal/model"
	"goduper/internal/opts"
	"goduper/internal/plex"
)

func Run(ctx context.Context, pc *plex.Client, o opts.Options) (model.Output, error) {

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

	// Prepare output
	out := model.Output{
		Server: pc.BaseURL(),
	}
	ignored := make([]model.IgnoredItem, 0)

	totalItems := 0
	totalVersions := 0
	totalGhosts := 0
	totalVariantsExcluded := 0

	var libSummaries []model.LibrarySummary

	for _, sec := range sections {
		vids, err := pc.FetchDuplicatesForSection(ctx, sec.Key)
		if err != nil {

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
			// Fetch deeply if requested (to get full media/part details)
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

			// Ignore Exact 4K+1080 pair (and only that case)
			if shouldExcludeAs4k1080Pair(item, o.DupPolicy) {
				secVariantsExcluded++
				ignored = append(ignored, model.IgnoredItem{
					SectionID:    sec.Key,
					SectionTitle: sec.Title,
					Reason:       "4k+1080_pair",
					Item:         item,
				})
				continue
			}

			secTotalVersions += len(item.Versions)
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
	out.Ignored = ignored

	return out, nil
}

// ErrNoSections is returned when no movie/show sections are found.
var ErrNoSections = &noSectionsErr{}

type noSectionsErr struct{}

func (*noSectionsErr) Error() string { return "no movie/show sections found" }

// Only exclude when exactly one 4K and one 1080p version exist (no others).
func shouldExcludeAs4k1080Pair(it model.Item, policy string) bool {
	if strings.ToLower(policy) != "ignore-4k-1080" {
		return false
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

func normalizeResKey(v model.Version) string {
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

func fallback[T comparable](a, b T) T {
	var zero T
	if a != zero {
		return a
	}
	return b
}
