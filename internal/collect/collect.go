package collect

import (
	"context"
	"fmt"
	"strings"

	"goduper/internal/model"
	"goduper/internal/opts"
	"goduper/internal/plex"
)

func Run(ctx context.Context, pc *plex.Client, o opts.Options) (model.Output, error) {
	var sections []plex.Directory
	var err error

	if o.SectionsCSV != "" {
		for _, s := range strings.Split(o.SectionsCSV, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			sections = append(sections, plex.Directory{Key: s, Type: "movie", Title: "(manual " + s + ")"})
		}
	} else {
		sections, err = pc.DiscoverSections(ctx, o.IncludeShows)
		if err != nil {
			return model.Output{}, fmt.Errorf("discover sections: %w", err)
		}
		if len(sections) == 0 {
			return model.Output{}, fmt.Errorf("no movie/show sections found")
		}
	}

	out := model.Output{
		Server: pc.BaseURL(),
	}

	totalItems := 0
	totalVersions := 0
	totalGhosts := 0
	var libSummaries []model.LibrarySummary

	for _, sec := range sections {
		vids, err := pc.FetchDuplicatesForSection(ctx, sec.Key)
		if err != nil {
			// Don't fail the whole run for one library
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

		for _, v := range vids {
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
				secTotalVersions++
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
						secGhostParts++
						itemGhosts++
					}
				}
				item.Versions = append(item.Versions, ver)
			}
			if itemGhosts > 0 {
				secItemsWithGhosts++
			}
			sectionRes.Items = append(sectionRes.Items, item)
		}

		libSummaries = append(libSummaries, model.LibrarySummary{
			SectionID:       sec.Key,
			SectionTitle:    sec.Title,
			Type:            sec.Type,
			DuplicateItems:  len(sectionRes.Items),
			TotalVersions:   secTotalVersions,
			GhostParts:      secGhostParts,
			ItemsWithGhosts: secItemsWithGhosts,
		})

		totalItems += len(sectionRes.Items)
		totalVersions += secTotalVersions
		totalGhosts += secGhostParts
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
		Libraries:             libSummaries,
	}
	return out, nil
}

func fallback[T comparable](a, b T) T {
	var zero T
	if a != zero {
		return a
	}
	return b
}
