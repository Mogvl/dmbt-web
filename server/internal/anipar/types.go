package anipar

// ParseResult mirrors the TypeScript ParseResult interface. All optional
// fields are omitted from JSON output when empty, matching the original
// serialized snapshots.
type ParseResult struct {
	Title         string         `json:"title"`
	Titles        []string       `json:"titles,omitempty"`
	Fansub        *FansubInfo    `json:"fansub,omitempty"`
	Season        *SeasonInfo    `json:"season,omitempty"`
	Seasons       []SeasonInfo   `json:"seasons,omitempty"`
	SeasonsRange  *SeasonsRange  `json:"seasonsRange,omitempty"`
	Part          *PartInfo      `json:"part,omitempty"`
	Type          string         `json:"type,omitempty"`
	Episode       *EpisodeInfo   `json:"episode,omitempty"`
	Volume        *VolumeInfo    `json:"volume,omitempty"`
	Volumes       *VolumeInfo    `json:"volumes,omitempty"`
	VolumesRange  *VolumesRange  `json:"volumesRange,omitempty"`
	Episodes      []EpisodeInfo  `json:"episodes,omitempty"`
	EpisodesRange *EpisodesRange `json:"episodesRange,omitempty"`
	Version       int            `json:"version,omitempty"`
	Subtitle      *SubtitleInfo  `json:"subtitle,omitempty"`
	Source        string         `json:"source,omitempty"`
	Platform      string         `json:"platform,omitempty"`
	Year          int            `json:"year,omitempty"`
	Month         int            `json:"month,omitempty"`
	File          *FileInfo      `json:"file,omitempty"`
	TmdbID        string         `json:"tmdbId,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
	Variants      []string       `json:"variants,omitempty"`
	Search        []string       `json:"search,omitempty"`
}

// FansubInfo mirrors ParseResult.fansub.
type FansubInfo struct {
	Name   string   `json:"name"`
	Alias  string   `json:"alias,omitempty"`
	Collab []string `json:"collab,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

// SubtitleInfo mirrors ParseResult.subtitle.
type SubtitleInfo struct {
	Format    string   `json:"format,omitempty"`
	Encoding  string   `json:"encoding,omitempty"`
	Encodings []string `json:"encodings,omitempty"`
	Languages []string `json:"languages,omitempty"`
}

// EpisodeInfo mirrors ParseResult.episode / episodes[] items.
type EpisodeInfo struct {
	Number    int    `json:"number"`
	NumberSub int    `json:"numberSub,omitempty"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
}

// VolumeInfo mirrors ParseResult.volume / volumes.
type VolumeInfo struct {
	Number int `json:"number"`
}

// VolumesRange mirrors ParseResult.volumesRange.
type VolumesRange struct {
	From int    `json:"from"`
	To   int    `json:"to"`
	Type string `json:"type,omitempty"`
}

// EpisodesRange mirrors ParseResult.episodesRange. Type is a pointer because
// the original code can explicitly set it to "" (e.g. "01-12v2" strips the
// version suffix leaving an empty type), which must survive JSON output.
type EpisodesRange struct {
	From    int     `json:"from"`
	FromSub int     `json:"fromSub,omitempty"`
	To      int     `json:"to"`
	ToSub   int     `json:"toSub,omitempty"`
	Type    *string `json:"type,omitempty"`
}

// SeasonInfo mirrors ParseResult.season / seasons[] items.
type SeasonInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title,omitempty"`
}

// SeasonsRange mirrors ParseResult.seasonsRange.
type SeasonsRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// PartInfo mirrors ParseResult.part.
type PartInfo struct {
	Number int `json:"number"`
}

// FileInfo mirrors ParseResult.file.
type FileInfo struct {
	Extension string     `json:"extension,omitempty"`
	Audio     *AudioInfo `json:"audio,omitempty"`
	Video     *VideoInfo `json:"video,omitempty"`
}

// AudioInfo mirrors ParseResult.file.audio.
type AudioInfo struct {
	Channels   string `json:"channels,omitempty"`
	Codec      string `json:"codec,omitempty"`
	Language   string `json:"language,omitempty"`
	TrackCount int    `json:"trackCount,omitempty"`
}

// VideoInfo mirrors ParseResult.file.video.
type VideoInfo struct {
	Codec         string `json:"codec,omitempty"`
	Enhancement   string `json:"enhancement,omitempty"`
	Format        string `json:"format,omitempty"`
	FrameRateMode string `json:"frameRateMode,omitempty"`
	Quality       string `json:"quality,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	BitDepth      string `json:"bitDepth,omitempty"`
	Fps           string `json:"fps,omitempty"`
}
