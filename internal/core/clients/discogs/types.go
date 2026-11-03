package discogs

import "strings"

type (
	Format struct {
		Name         string   `json:"name"`
		Text         string   `json:"text"`
		Descriptions []string `json:"descriptions"`
	}

	Release struct {
		ID      int      `json:"id"`
		Title   string   `json:"title"`
		Country string   `json:"country"`
		Year    string   `json:"year"`
		Labels  []string `json:"label"`
		Catno   string   `json:"catno"`
		Formats []Format `json:"formats"`
		Thumb   string   `json:"thumb"`
		Path    string   `json:"uri"`
	}

	Artist struct {
		Name string `json:"name"`
	}

	ReleaseDetail struct {
		ID      int      `json:"id"`
		Title   string   `json:"title"`
		Artists []Artist `json:"artists"`
	}

	OwnedRelease struct {
		ID      int      `json:"id"`
		Title   string   `json:"title"`
		Year    int      `json:"year"`
		Thumb   string   `json:"thumb"`
		Artists []Artist `json:"artists"`
		Formats []Format `json:"formats"`
		Labels  []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}

	OwnedMatches struct {
		Exact   []OwnedRelease
		Similar []OwnedRelease
	}
)

func (r Release) Label() string {
	if len(r.Labels) == 0 {
		return ""
	}

	return r.Labels[0]
}

func (r Release) FormatSummary() string {
	return formatSummary(r.Formats)
}

func (r Release) Variant() string {
	return variant(r.Formats)
}

func (r Release) URL() string {
	return "https://www.discogs.com" + r.Path
}

func (d ReleaseDetail) PrimaryArtist() string {
	if len(d.Artists) == 0 {
		return ""
	}

	return d.Artists[0].Name
}

func (o OwnedRelease) PrimaryArtist() string {
	if len(o.Artists) == 0 {
		return ""
	}

	return o.Artists[0].Name
}

func (o OwnedRelease) Label() string {
	if len(o.Labels) == 0 {
		return ""
	}

	return o.Labels[0].Name
}

func (o OwnedRelease) FormatSummary() string {
	return formatSummary(o.Formats)
}

func (o OwnedRelease) Variant() string {
	return variant(o.Formats)
}

func (m OwnedMatches) Any() bool {
	return len(m.Exact) > 0 || len(m.Similar) > 0
}

func MatchOwned(release ReleaseDetail, owned []OwnedRelease) OwnedMatches {
	var matches OwnedMatches

	for _, o := range owned {
		switch {
		case o.ID == release.ID:
			matches.Exact = append(matches.Exact, o)
		case strings.EqualFold(o.PrimaryArtist(), release.PrimaryArtist()) &&
			strings.EqualFold(o.Title, release.Title):
			matches.Similar = append(matches.Similar, o)
		}
	}

	return matches
}

func formatSummary(formats []Format) string {
	var parts []string

	for _, f := range formats {
		parts = append(parts, f.Name)
		parts = append(parts, f.Descriptions...)
	}

	return strings.Join(parts, ", ")
}

func variant(formats []Format) string {
	var texts []string

	for _, f := range formats {
		if f.Text != "" {
			texts = append(texts, f.Text)
		}
	}

	return strings.Join(texts, ", ")
}
