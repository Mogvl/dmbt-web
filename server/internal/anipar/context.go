package anipar

import (
	"strings"
)

// Context mirrors the Context class in context.ts: it tracks the consumed
// (left) and unconsumed (right) token cursors plus the partial parse result.
type Context struct {
	left   int
	right  int
	tokens []*Token
	fansub string // options.fansub
	result *ParseResult
	tags   []string
}

func NewContext(tokens []*Token, fansub string) *Context {
	result := &ParseResult{}
	if fansub != "" {
		result.Fansub = &FansubInfo{Name: fansub}
	}
	return &Context{
		left:   0,
		right:  len(tokens) - 1,
		tokens: tokens,
		fansub: fansub,
		result: result,
		tags:   []string{},
	}
}

func (c *Context) Left() int  { return c.left }
func (c *Context) Right() int { return c.right }

// HasEpisode mirrors the hasEpisode getter.
func (c *Context) HasEpisode() bool {
	return c.result.Episode != nil || c.result.Episodes != nil || c.result.EpisodesRange != nil
}

// SubtitleLanguages mirrors `ctx.result.subtitle?.languages ?? []`.
func (r *ParseResult) SubtitleLanguages() []string {
	if r.Subtitle == nil {
		return nil
	}
	return r.Subtitle.Languages
}

func (r *ParseResult) SubtitleFormat() string {
	if r.Subtitle == nil {
		return ""
	}
	return r.Subtitle.Format
}

func (r *ParseResult) SubtitleEncoding() string {
	if r.Subtitle == nil {
		return ""
	}
	return r.Subtitle.Encoding
}

func (r *ParseResult) SubtitleEncodings() []string {
	if r.Subtitle == nil {
		return nil
	}
	return r.Subtitle.Encodings
}

func (c *Context) clearSubtitleEncoding() {
	if c.result.Subtitle != nil {
		c.result.Subtitle.Encoding = ""
	}
}

// update mirrors Context.update.
func (c *Context) update(key string, value any) {
	switch key {
	case "title":
		c.result.Title = value.(string)
	case "titles":
		c.result.Titles = value.([]string)
	case "type":
		c.result.Type = value.(string)
	case "source":
		c.result.Source = value.(string)
	case "platform":
		c.result.Platform = value.(string)
	case "version":
		c.result.Version = value.(int)
	case "year":
		c.result.Year = value.(int)
	case "month":
		c.result.Month = value.(int)
	case "tmdbId":
		c.result.TmdbID = value.(string)
	case "search":
		c.result.Search = value.([]string)
	case "variants":
		c.result.Variants = value.([]string)
	case "episodes":
		c.result.Episodes = value.([]EpisodeInfo)
	case "seasons":
		c.result.Seasons = value.([]SeasonInfo)
	}
}

// update2 mirrors Context.update2.
func (c *Context) update2(key1, key2 string, value any) {
	switch key1 {
	case "fansub":
		if c.result.Fansub == nil {
			c.result.Fansub = &FansubInfo{}
		}
		switch key2 {
		case "name":
			c.result.Fansub.Name = value.(string)
		case "alias":
			c.result.Fansub.Alias = value.(string)
		case "collab":
			c.result.Fansub.Collab = value.([]string)
		case "tags":
			c.result.Fansub.Tags = value.([]string)
		}
	case "season":
		if c.result.Season == nil {
			c.result.Season = &SeasonInfo{}
		}
		switch key2 {
		case "number":
			c.result.Season.Number = value.(int)
		case "title":
			c.result.Season.Title = value.(string)
		}
	case "episode":
		if c.result.Episode == nil {
			c.result.Episode = &EpisodeInfo{}
		}
		switch key2 {
		case "number":
			c.result.Episode.Number = value.(int)
		case "numberSub":
			c.result.Episode.NumberSub = value.(int)
		case "type":
			c.result.Episode.Type = value.(string)
		case "title":
			c.result.Episode.Title = value.(string)
		}
	case "volume":
		if c.result.Volume == nil {
			c.result.Volume = &VolumeInfo{}
		}
		if key2 == "number" {
			c.result.Volume.Number = value.(int)
		}
	case "seasonsRange":
		if c.result.SeasonsRange == nil {
			c.result.SeasonsRange = &SeasonsRange{}
		}
		switch key2 {
		case "from":
			c.result.SeasonsRange.From = value.(int)
		case "to":
			c.result.SeasonsRange.To = value.(int)
		}
	case "episodesRange":
		if c.result.EpisodesRange == nil {
			c.result.EpisodesRange = &EpisodesRange{}
		}
		switch key2 {
		case "from":
			c.result.EpisodesRange.From = value.(int)
		case "fromSub":
			c.result.EpisodesRange.FromSub = value.(int)
		case "to":
			c.result.EpisodesRange.To = value.(int)
		case "toSub":
			c.result.EpisodesRange.ToSub = value.(int)
		case "type":
			v := value.(string)
			c.result.EpisodesRange.Type = &v
		}
	case "volumesRange":
		if c.result.VolumesRange == nil {
			c.result.VolumesRange = &VolumesRange{}
		}
		switch key2 {
		case "from":
			c.result.VolumesRange.From = value.(int)
		case "to":
			c.result.VolumesRange.To = value.(int)
		case "type":
			c.result.VolumesRange.Type = value.(string)
		}
	case "subtitle":
		if c.result.Subtitle == nil {
			c.result.Subtitle = &SubtitleInfo{}
		}
		switch key2 {
		case "format":
			c.result.Subtitle.Format = value.(string)
		case "encoding":
			c.result.Subtitle.Encoding = value.(string)
		case "encodings":
			c.result.Subtitle.Encodings = value.([]string)
		case "languages":
			c.result.Subtitle.Languages = value.([]string)
		}
	case "file":
		if c.result.File == nil {
			c.result.File = &FileInfo{}
		}
		if key2 == "extension" {
			c.result.File.Extension = value.(string)
		}
	case "part":
		if c.result.Part == nil {
			c.result.Part = &PartInfo{}
		}
		if key2 == "number" {
			c.result.Part.Number = value.(int)
		}
	}
}

// update3 mirrors Context.update3 for file.audio / file.video.
func (c *Context) update3(key1, key2, key3 string, value any) {
	if key1 != "file" {
		return
	}
	if c.result.File == nil {
		c.result.File = &FileInfo{}
	}
	switch key2 {
	case "audio":
		if c.result.File.Audio == nil {
			c.result.File.Audio = &AudioInfo{}
		}
		switch key3 {
		case "codec":
			c.result.File.Audio.Codec = value.(string)
		case "channels":
			c.result.File.Audio.Channels = value.(string)
		case "language":
			c.result.File.Audio.Language = value.(string)
		case "trackCount":
			c.result.File.Audio.TrackCount = value.(int)
		}
	case "video":
		if c.result.File.Video == nil {
			c.result.File.Video = &VideoInfo{}
		}
		switch key3 {
		case "codec":
			c.result.File.Video.Codec = value.(string)
		case "bitDepth":
			c.result.File.Video.BitDepth = value.(string)
		case "resolution":
			c.result.File.Video.Resolution = value.(string)
		case "enhancement":
			c.result.File.Video.Enhancement = value.(string)
		case "format":
			c.result.File.Video.Format = value.(string)
		case "frameRateMode":
			c.result.File.Video.FrameRateMode = value.(string)
		case "quality":
			c.result.File.Video.Quality = value.(string)
		case "fps":
			c.result.File.Video.Fps = value.(string)
		}
	}
}

// normalize mirrors Context.normalize: merges tags, normalizes subtitle /
// file / variants fields, and returns nil when no title remains.
func (c *Context) normalize() *ParseResult {
	if len(c.tags) > 0 {
		merged := append([]string{}, c.result.Tags...)
		merged = append(merged, c.tags...)
		c.result.Tags = dedupeStrings(merged)
	}

	if c.result.Subtitle != nil {
		if c.result.Subtitle.Languages != nil {
			c.result.Subtitle.Languages = normalizeLanguages(c.result.Subtitle.Languages)
		}
		if c.result.Subtitle.Format != "" {
			c.result.Subtitle.Format = normalizeSubtitleFormat(c.result.Subtitle.Format)
		}
		if c.result.Subtitle.Encoding != "" {
			c.result.Subtitle.Encoding = normalizeSubtitleEncoding(c.result.Subtitle.Encoding)
		}
		if c.result.Subtitle.Encodings != nil {
			c.result.Subtitle.Encodings = normalizeSubtitleEncodings(c.result.Subtitle.Encodings)
		}
	}

	if c.result.File != nil {
		normalizeFileInfo(c.result.File)
	}

	if c.result.Variants != nil {
		c.result.Variants = dedupeStrings(c.result.Variants)
	}

	if c.result.Title != "" {
		return c.result
	}
	return nil
}

// MARK: 字幕类型归一化

func normalizeSubtitleFormat(format string) string {
	trimmed := jsTrim(format)
	upper := normalizeUpperTag(trimmed)

	// ASS/SRT with optional ×N count.
	{
		res := assSrtFormatRE.FindStringSubmatch(trimmed)
		if res != nil {
			if res[2] != "" {
				return strings.ToUpper(res[1]) + "字幕×" + res[2]
			}
			return strings.ToUpper(res[1]) + "字幕"
		}
	}

	if hardSubRE.MatchString(upper) || neiqianRE1.MatchString(trimmed) {
		return "内嵌字幕"
	}
	if softSubRE.MatchString(upper) {
		return "软字幕"
	}
	if neifengRE.MatchString(trimmed) {
		return "内封字幕"
	}
	if waiguaRE.MatchString(trimmed) {
		return "外挂字幕"
	}
	if neiguaRE.MatchString(trimmed) {
		return "内挂字幕"
	}
	if subRE.MatchString(upper) || trimmed == "字幕" {
		return "字幕"
	}

	return strings.ReplaceAll(strings.ReplaceAll(trimmed, "內", "内"), "掛", "挂")
}

func normalizeSubtitleEncoding(encoding string) string {
	return normalizeUpperTag(encoding)
}

func normalizeSubtitleEncodings(encodings []string) []string {
	normalized := map[string]bool{}
	for _, e := range encodings {
		normalized[normalizeSubtitleEncoding(e)] = true
	}
	order := []string{"GB", "BIG5"}
	result := []string{}
	for _, encoding := range order {
		if normalized[encoding] {
			delete(normalized, encoding)
			result = append(result, encoding)
		}
	}
	for _, e := range encodings {
		enc := normalizeSubtitleEncoding(e)
		if normalized[enc] {
			delete(normalized, enc)
			result = append(result, enc)
		}
	}
	return result
}

// MARK: 媒体信息归一化

func normalizeFileInfo(file *FileInfo) {
	if file.Audio != nil {
		normalizeAudioInfo(file.Audio)
	}
	if file.Video != nil {
		normalizeVideoInfo(file.Video)
	}
}

func normalizeAudioInfo(audio *AudioInfo) {
	if audio.Channels != "" {
		audio.Channels = normalizeAudioChannels(audio.Channels)
	}
	if audio.Codec != "" {
		audio.Codec = normalizeAudioCodec(audio.Codec)
	}
	if audio.Language != "" {
		audio.Language = normalizeAudioLanguage(audio.Language)
	}
}

func normalizeVideoInfo(video *VideoInfo) {
	if video.BitDepth != "" {
		video.BitDepth = normalizeBitDepth(video.BitDepth)
	}
	if video.Codec != "" {
		video.Codec = normalizeVideoCodec(video.Codec)
	}
	if video.Enhancement != "" {
		video.Enhancement = normalizeUpperTag(video.Enhancement)
	}
	if video.Format != "" {
		video.Format = normalizeUpperTag(video.Format)
	}
	if video.Fps != "" {
		video.Fps = normalizeFrameRate(video.Fps)
	}
	if video.Quality != "" {
		video.Quality = normalizeUpperTag(video.Quality)
	}
	if video.Resolution != "" {
		video.Resolution = normalizeVideoResolution(video.Resolution)
	}
}

func normalizeUpperTag(value string) string {
	return strings.ToUpper(jsTrim(value))
}

func normalizeLowerTag(value string) string {
	return strings.ToLower(jsTrim(value))
}

func normalizeAudioCodec(codec string) string {
	upper := normalizeUpperTag(codec)
	if v, ok := audioCodecMap[upper]; ok {
		return v
	}
	return normalizeLowerTag(codec)
}

func normalizeVideoCodec(codec string) string {
	upper := normalizeUpperTag(codec)
	switch {
	case videoCodecAVCRE.MatchString(upper):
		return "AVC"
	case videoCodecHEVCRE.MatchString(upper):
		return "HEVC"
	case videoCodecDivxRE.MatchString(upper):
		return "DivX"
	case upper == "XVID":
		return "Xvid"
	case videoCodecHi10RE.MatchString(upper):
		return "Hi10P"
	case videoCodecHi444RE.MatchString(upper):
		return strings.Replace(upper, "HI", "Hi", 1)
	}
	return normalizeLowerTag(codec)
}

func normalizeAudioChannels(channels string) string {
	upper := normalizeUpperTag(channels)
	if upper == "2CH" {
		return "2.0"
	}
	if upper == "5.1CH" {
		return "5.1"
	}
	return chSuffixRE.ReplaceAllString(upper, "")
}

func normalizeAudioLanguage(language string) string {
	upper := normalizeUpperTag(language)
	if upper == "DUALAUDIO" || upper == "DUAL AUDIO" {
		return "dual audio"
	}
	return normalizeLowerTag(language)
}

func normalizeBitDepth(bitDepth string) string {
	upper := normalizeUpperTag(bitDepth)
	res := bitDepthRE.FindStringSubmatch(upper)
	if res != nil {
		return res[1] + "-bit"
	}
	return normalizeLowerTag(bitDepth)
}

func normalizeFrameRate(fps string) string {
	upper := normalizeUpperTag(fps)
	res := frameRateRE.FindStringSubmatch(upper)
	if res != nil {
		return res[1] + "fps"
	}
	return normalizeLowerTag(fps)
}

func normalizeVideoResolution(resolution string) string {
	trimmed := jsTrim(resolution)
	upper := strings.ToUpper(trimmed)
	{
		res := resPHeightRE.FindStringSubmatch(trimmed)
		if res != nil {
			return res[1] + "p"
		}
	}
	{
		res := resPixelRE.FindStringSubmatch(trimmed)
		if res != nil {
			return res[1] + "x" + res[2]
		}
	}
	if resKRE.MatchString(upper) {
		return upper
	}
	return trimmed
}

// MARK: 字幕语言归一化

var normalizedLanguages = []string{"简", "繁", "粤", "日", "英", "泰"}

func normalizeLanguage(language string) []string {
	trimmed := jsTrim(language)
	upper := strings.ToUpper(trimmed)

	if ignoreLanguageRE.MatchString(upper) {
		return nil
	}

	matches := map[string]bool{
		"简": langCHSRE.MatchString(trimmed),
		"繁": langCHTRE.MatchString(trimmed),
		"粤": langYUERE.MatchString(trimmed),
		"日": langJPRE.MatchString(trimmed),
		"英": langENRE.MatchString(trimmed),
		"泰": langTHRE.MatchString(trimmed),
	}

	if chineseCharRE.MatchString(trimmed) && !matches["简"] && !matches["繁"] && !matches["粤"] {
		return nil
	}

	languages := []string{}
	for _, language := range normalizedLanguages {
		if matches[language] {
			languages = append(languages, language)
		}
	}
	if len(languages) > 0 {
		return languages
	}
	return nil
}

func normalizeLanguages(languages []string) []string {
	normalized := map[string]bool{}
	var unknown []string
	for _, language := range languages {
		parts := normalizeLanguage(language)
		if parts != nil {
			for _, part := range parts {
				normalized[part] = true
			}
		} else if !containsString(unknown, language) {
			unknown = append(unknown, language)
		}
	}
	result := []string{}
	for _, language := range normalizedLanguages {
		if normalized[language] {
			result = append(result, language)
		}
	}
	return append(result, unknown...)
}

var (
	assSrtFormatRE = jsRegex(`^(ASS|SRT)(?:[Xx×](\d+))?$`, true)
	hardSubRE      = jsRegex(`^HARDSUBS?$`, false)
	softSubRE      = jsRegex(`^SOFTSUBS?$`, false)
	subRE          = jsRegex(`^(SUB|SUBBED|SUBTITLED)$`, false)
	neiqianRE1     = jsRegex(`^(内嵌|內嵌)(字幕)?$`, false)
	neifengRE      = jsRegex(`^(内封|內封)(字幕)?$`, false)
	waiguaRE       = jsRegex(`^(外挂|外掛)(字幕)?$`, false)
	neiguaRE       = jsRegex(`^(内挂|內掛)(字幕)?$`, false)

	audioCodecMap = map[string]string{
		"AAC":      "AAC",
		"AC3":      "AC-3",
		"AC-3":     "AC-3",
		"DTS":      "DTS",
		"DTS-ES":   "DTS-ES",
		"EAC3":     "E-AC-3",
		"E-AC-3":   "E-AC-3",
		"EAC3&AAC": "E-AC-3+AAC",
		"FLAC":     "FLAC",
		"FLAC/AC3": "FLAC+AC-3",
		"LOSSLESS": "lossless",
		"MP3":      "MP3",
		"OGG":      "Ogg",
		"OPUS":     "Opus",
		"QAAC":     "qaac",
		"TRUEHD":   "TrueHD",
		"VORBIS":   "Vorbis",
		"WAV":      "WAV",
	}

	videoCodecAVCRE   = jsRegex(`^(H\.?264|X\.?264|AVC)$`, false)
	videoCodecHEVCRE  = jsRegex(`^(H\.?265|X\.?265|HEVC2?|HVC1)$`, false)
	videoCodecDivxRE  = jsRegex(`^DIVX\d*$`, false)
	videoCodecHi10RE  = jsRegex(`^HI10P?$`, false)
	videoCodecHi444RE = jsRegex(`^HI444P*$`, false)

	chSuffixRE   = jsRegex(`CH$`, false)
	bitDepthRE   = jsRegex(`^(\d+)[-_ ]?BITS?$`, false)
	frameRateRE  = jsRegex(`^(\d+(?:\.\d+)?)\s*FPS$`, false)
	resPHeightRE = jsRegex(`^(\d{3,4})P$`, true)
	resPixelRE   = jsRegex(`^(\d{3,5})[Xx×](\d{3,5})$`, false)
	resKRE       = jsRegex(`^\d+K$`, false)

	ignoreLanguageRE = jsRegex(`^(CN|CHINESE|ZH|中|中文|中字|国语中字|國語中字)$`, false)
	langCHSRE        = jsRegex(`(^|[^A-Z])CHS($|[^A-Z])|ZH[-_]?HANS|简|簡|简体|簡體|简中|簡中`, true)
	langCHTRE        = jsRegex(`(^|[^A-Z])CHT($|[^A-Z])|ZH[-_]?HANT|繁|繁体|繁體|繁中|BIG5`, true)
	langYUERE        = jsRegex(`(^|[^A-Z])YUE($|[^A-Z])|粤|粵|广东话|廣東話|CANTONESE`, true)
	langJPRE         = jsRegex(`(^|[^A-Z])(JP|JPN|JA)($|[^A-Z])|日|日本|JAPANESE`, true)
	langENRE         = jsRegex(`(^|[^A-Z])(EN|ENG)($|[^A-Z])|英|ENGLISH`, true)
	langTHRE         = jsRegex(`(^|[^A-Z])(TH|THA)($|[^A-Z])|泰|THAI`, true)
	chineseCharRE    = jsRegex(`[中华華]`, false)
)
