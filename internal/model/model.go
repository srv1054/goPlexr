package model

// Output (top-level JSON)
type Output struct {
	Server        string          `json:"server"`
	Sections      []SectionResult `json:"sections"`
	TotalItems    int             `json:"total_duplicate_items"`
	TotalVersions int             `json:"total_versions"`
	TotalGhosts   int             `json:"total_ghost_parts"`
	Summary       Summary         `json:"summary"`
}

type SectionResult struct {
	SectionID    string `json:"section_id"`
	SectionTitle string `json:"section_title"`
	Type         string `json:"type"`
	Items        []Item `json:"items"` // duplicate items only
}

type Item struct {
	RatingKey string    `json:"rating_key"`
	Title     string    `json:"title"`
	Year      int       `json:"year,omitempty"`
	Guid      string    `json:"guid,omitempty"`
	Versions  []Version `json:"versions"`
}

type Version struct {
	ID              string    `json:"id,omitempty"`
	Container       string    `json:"container,omitempty"`
	VideoCodec      string    `json:"video_codec,omitempty"`
	AudioCodec      string    `json:"audio_codec,omitempty"`
	VideoResolution string    `json:"video_resolution,omitempty"`
	Bitrate         int       `json:"bitrate,omitempty"`
	Width           int       `json:"width,omitempty"`
	Height          int       `json:"height,omitempty"`
	Parts           []PartOut `json:"parts,omitempty"`
}

type PartOut struct {
	ID             string `json:"id,omitempty"`
	File           string `json:"file,omitempty"`
	Size           int64  `json:"size,omitempty"`
	Duration       int    `json:"duration,omitempty"`
	VerifiedOnDisk bool   `json:"verified_on_disk"`
	Exists         bool   `json:"exists"`
	Accessible     bool   `json:"accessible"`
}

// Summary aggregates
type Summary struct {
	VerificationPerformed bool             `json:"verification_performed"`
	TotalLibraries        int              `json:"total_libraries"`
	TotalDuplicateItems   int              `json:"total_duplicate_items"`
	TotalGhostParts       int              `json:"total_ghost_parts"`
	Libraries             []LibrarySummary `json:"libraries"`
}

type LibrarySummary struct {
	SectionID       string `json:"section_id"`
	SectionTitle    string `json:"section_title"`
	Type            string `json:"type"`
	DuplicateItems  int    `json:"duplicate_items"`
	TotalVersions   int    `json:"total_versions"`
	GhostParts      int    `json:"ghost_parts"`
	ItemsWithGhosts int    `json:"items_with_ghosts"`
}
