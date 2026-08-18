package push

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Mogvl/dmbt-web/server/internal/anipar"
	"github.com/Mogvl/dmbt-web/server/internal/zh"
)

// keepshareID mirrors the original KEEPSHARE_ID default.
const keepshareID = "gv78k1oi"

// pushSite is used for the detail link (APP_HOST).
const pushSite = "animes.garden"

// buildResourceCardMessage ports buildResourceCardMessage from
// apps/server/src/push/message.ts (verified against the original test
// fixture output).
func buildResourceCardMessage(
	title string,
	createdAt time.Time,
	magnet string,
	size int64,
	provider, providerID string,
	fansubName string,
	parsed *anipar.ParseResult,
	subject *subjectCard,
) (photo string, caption string) {
	if subject != nil && subject.Poster != "" {
		photo = subject.Poster
	} else {
		photo = "https://animes.garden/favicon.svg"
	}

	subjectName := subjectNameOf(subject)

	var lines []string

	// <b>{SubjectDisplayName}{ · 第 x 集}</b>
	lines = append(lines, formatTitleLine(subjectName, parsed))

	// #{FansubHashtags} · #{yyyy年M月新番}
	if line := formatFansubs(parsed, subject, createdAt); line != "" {
		lines = append(lines, line)
	}

	// <b>字幕:</b> ...
	if line := formatSubtitleLine(parsed); line != "" {
		lines = append(lines, line)
	}

	// <b>格式:</b> ...
	if line := formatVideoLine(parsed); line != "" {
		lines = append(lines, line)
	}

	// <b>大小:</b> ...
	lines = append(lines, formatResourceSize(size))

	// <b>发布:</b> ...
	lines = append(lines, formatPublishTime(createdAt))

	// <b>追踪:</b> ...
	if subjectName != "" {
		fansub := normalizeFansubName(fansubName)
		if line := formatLabels(fansub, subjectName); line != "" {
			lines = append(lines, line)
		}
	}

	// links
	detail := fmt.Sprintf("https://%s/detail/%s/%s", pushSite, provider, providerID)
	play := fmt.Sprintf("https://keepshare.org/%s/%s", keepshareID, url.QueryEscape(magnet))
	lines = append(lines, fmt.Sprintf(`<a href="%s">查看详情</a> · <a href="%s">在线播放</a>`, detail, play))

	return photo, strings.Join(lines, "\n")
}

// subjectCard carries the bgm subject metadata used by the caption.
type subjectCard struct {
	Name      string // alias.zh[0] ?? title
	Poster    string
	OnairDate string // yyyy-MM-dd
}

func subjectNameOf(s *subjectCard) string {
	if s == nil || s.Name == "" {
		return ""
	}
	return s.Name
}

// formatTitleLine: <b>{name}{ · 第 x 集}</b>
func formatTitleLine(subjectName string, parsed *anipar.ParseResult) string {
	episode := formatEpisode(parsed)
	if episode == "" {
		return "<b>" + htmlEscape(subjectName) + "</b>"
	}
	return "<b>" + htmlEscape(subjectName) + " · " + htmlEscape(episode) + "</b>"
}

// formatEpisode: 第 {n} 集 / 第 {a}-{b} 集 / 第 {a},{b} 集
func formatEpisode(parsed *anipar.ParseResult) string {
	if parsed == nil {
		return ""
	}
	if parsed.EpisodesRange != nil {
		return "第 " + formatEpisodeNumber(parsed.EpisodesRange.From, parsed.EpisodesRange.FromSub) +
			"-" + formatEpisodeNumber(parsed.EpisodesRange.To, parsed.EpisodesRange.ToSub) + " 集"
	}
	if len(parsed.Episodes) > 0 {
		parts := make([]string, 0, len(parsed.Episodes))
		for _, e := range parsed.Episodes {
			parts = append(parts, formatEpisodeNumber(e.Number, e.NumberSub))
		}
		return "第 " + strings.Join(parts, ",") + " 集"
	}
	if parsed.Episode != nil {
		return "第 " + formatEpisodeNumber(parsed.Episode.Number, parsed.Episode.NumberSub) + " 集"
	}
	return ""
}

func formatEpisodeNumber(number, numberSub int) string {
	if numberSub > 0 {
		return strconv.Itoa(number) + "." + strconv.Itoa(numberSub)
	}
	return strconv.Itoa(number)
}

// formatFansubs: #{fansub hashtags} · #{yyyy年M月新番}
func formatFansubs(parsed *anipar.ParseResult, subject *subjectCard, createdAt time.Time) string {
	var names []string
	if parsed != nil && parsed.Fansub != nil {
		if parsed.Fansub.Name != "" {
			names = append(names, parsed.Fansub.Name)
		}
		names = append(names, parsed.Fansub.Collab...)
	}
	seen := map[string]bool{}
	var labels []string
	for _, n := range names {
		n = normalizeFansubName(n)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		if tag := formatHashTag(n); tag != "" {
			labels = append(labels, htmlEscape(tag))
		}
	}
	if len(labels) == 0 {
		return ""
	}
	// labels joined with ' ', then ' · ' before the quarter (original order)
	if quarter := formatQuarter(subject, createdAt); quarter != "" {
		return strings.Join(labels, " ") + " · " + htmlEscape(formatHashTag(quarter))
	}
	return strings.Join(labels, " ")
}

// normalizeFansubName ports normalizeFansubName: tradToSimple, strip
// [\s_\-()（）], known alias rewrites.
func normalizeFansubName(name string) string {
	if name == "" {
		return ""
	}
	simplified := zh.TradToSimple(name)
	compacted := strings.ToLower(strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '_', '-', '(', ')', '（', '）':
			return -1
		}
		return r
	}, simplified))
	switch compacted {
	case "flsnow", "雪飘工作室flsnow":
		return "雪飘工作室"
	case "nekomoekissaten", "喵萌奶茶屋":
		return "喵萌奶茶屋"
	}
	return simplified
}

// formatQuarter: alignQuarterDate then yyyy年M月新番.
func formatQuarter(subject *subjectCard, createdAt time.Time) string {
	date := createdAt
	if subject != nil && subject.OnairDate != "" {
		if t, err := time.Parse("2006-01-02", subject.OnairDate); err == nil {
			date = t
		}
	}
	date = alignQuarterDate(date)
	quarterMonth := (int(date.Month())-1)/3*3 + 1
	return fmt.Sprintf("%d年%d月新番", date.Year(), quarterMonth)
}

// alignQuarterDate ports alignQuarterDate: within 7 days before a quarter
// start (incl. next year's Jan 1) -> that start; else the date itself.
func alignQuarterDate(date time.Time) time.Time {
	year := date.Year()
	candidates := []time.Time{
		time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(year, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(year, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(year, 10, 1, 0, 0, 0, 0, time.UTC),
		time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, start := range candidates {
		threshold := start.Add(-7 * 24 * time.Hour)
		if !date.Before(threshold) && date.Before(start) {
			return start
		}
	}
	return date
}

// formatSubtitleLine: <b>字幕:</b> {languages / } · {format}
func formatSubtitleLine(parsed *anipar.ParseResult) string {
	if parsed == nil || parsed.Subtitle == nil {
		return ""
	}
	var parts []string
	if languages := formatSubtitleLanguages(parsed.Subtitle); languages != "" {
		parts = append(parts, languages)
	}
	if format := formatSubtitle(parsed.Subtitle); format != "" {
		parts = append(parts, format)
	}
	if len(parts) == 0 {
		return ""
	}
	escaped := make([]string, 0, len(parts))
	for _, p := range parts {
		escaped = append(escaped, htmlEscape(p))
	}
	return "<b>字幕:</b> " + strings.Join(escaped, " · ")
}

// formatSubtitleLanguages: 简->简中, 繁->繁中, 日->日语, 英->英语, joined ' / '.
func formatSubtitleLanguages(subtitle *anipar.SubtitleInfo) string {
	if subtitle == nil || len(subtitle.Languages) == 0 {
		return ""
	}
	mapped := make([]string, 0, len(subtitle.Languages))
	for _, lang := range subtitle.Languages {
		switch lang {
		case "简":
			mapped = append(mapped, "简中")
		case "繁":
			mapped = append(mapped, "繁中")
		case "日":
			mapped = append(mapped, "日语")
		case "英":
			mapped = append(mapped, "英语")
		default:
			mapped = append(mapped, lang)
		}
	}
	return strings.Join(mapped, " / ")
}

// formatSubtitle: 内嵌->内嵌字幕, 内封->内封字幕, 外挂->外挂字幕.
func formatSubtitle(subtitle *anipar.SubtitleInfo) string {
	if subtitle == nil || subtitle.Format == "" {
		return ""
	}
	switch subtitle.Format {
	case "内嵌":
		return "内嵌字幕"
	case "内封":
		return "内封字幕"
	case "外挂":
		return "外挂字幕"
	}
	return subtitle.Format
}

// formatVideoLine: <b>格式:</b> {res} · {codec-bit} · {fps} · {ext} · {audio}
func formatVideoLine(parsed *anipar.ParseResult) string {
	if parsed == nil || parsed.File == nil {
		return ""
	}
	file := parsed.File
	var parts []string
	if file.Video != nil && file.Video.Resolution != "" {
		parts = append(parts, formatResolution(file.Video.Resolution))
	}
	if codec := formatVideoCodec(file); codec != "" {
		parts = append(parts, codec)
	}
	if file.Video != nil && file.Video.Fps != "" {
		parts = append(parts, file.Video.Fps)
	}
	if container := formatContainer(file); container != "" {
		parts = append(parts, container)
	}
	if file.Audio != nil && file.Audio.Codec != "" {
		parts = append(parts, file.Audio.Codec)
	}
	if len(parts) == 0 {
		return ""
	}
	escaped := make([]string, 0, len(parts))
	for _, p := range parts {
		escaped = append(escaped, htmlEscape(p))
	}
	return "<b>格式:</b> " + strings.Join(escaped, " · ")
}

// formatVideoCodec: codec-bitDepth, falling back to either.
func formatVideoCodec(file *anipar.FileInfo) string {
	if file == nil || file.Video == nil {
		return ""
	}
	codec := file.Video.Codec
	bitDepth := file.Video.BitDepth
	if codec != "" && bitDepth != "" {
		return codec + "-" + bitDepth
	}
	if codec != "" {
		return codec
	}
	return bitDepth
}

// formatContainer: (extension ?? video.format).toLowerCase().
func formatContainer(file *anipar.FileInfo) string {
	if file == nil {
		return ""
	}
	ext := file.Extension
	if ext == "" && file.Video != nil {
		ext = file.Video.Format
	}
	return strings.ToLower(ext)
}

// formatResolution: "1920x1080" -> "1080p".
func formatResolution(resolution string) string {
	var w, h int
	if _, err := fmt.Sscanf(resolution, "%dx%d", &w, &h); err == nil {
		return strconv.Itoa(h) + "p"
	}
	return resolution
}

// formatResourceSize: size<=0 -> 未知; <1024 -> 'n KB'; <1MiB -> 'n.nn MB';
// else 'n.nn GB' (the resource size is in KB units, matching the original).
func formatResourceSize(size int64) string {
	if size <= 0 {
		return "<b>大小:</b> 未知"
	}
	if size < 1024 {
		return "<b>大小:</b> " + strconv.FormatInt(size, 10) + " KB"
	}
	if size < 1024*1024 {
		return fmt.Sprintf("<b>大小:</b> %.2f MB", float64(size)/1024)
	}
	return fmt.Sprintf("<b>大小:</b> %.2f GB", float64(size)/1024/1024)
}

// formatPublishTime: <b>发布:</b> Shanghai yyyy 年 M 月 d 日 HH:mm.
func formatPublishTime(value time.Time) string {
	shanghai := value.In(time.FixedZone("Asia/Shanghai", 8*60*60))
	return "<b>发布:</b> " + htmlEscape(shanghai.Format("2006 年 1 月 2 日 15:04"))
}

// formatLabels: <b>追踪:</b> #{subject} #{fansub_subject}.
func formatLabels(fansub, subjectName string) string {
	if fansub == "" {
		return ""
	}
	return "<b>追踪:</b> " + formatHashTag(subjectName) + " " + formatHashTag(fansub+"_"+subjectName)
}

// formatHashTag ports formatHashTag: keep letters/numbers/underscore
// (unicode-aware, CJK included), whitespace -> _, collapse, trim, '#...'.
func formatHashTag(text string) string {
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return '_'
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' {
			return r
		}
		return -1
	}, strings.TrimSpace(text))
	normalized = collapseUnderscores(normalized)
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return ""
	}
	return "#" + normalized
}

func collapseUnderscores(s string) string {
	var b strings.Builder
	last := false
	for _, r := range s {
		if r == '_' {
			if !last {
				b.WriteRune(r)
				last = true
			}
			continue
		}
		b.WriteRune(r)
		last = false
	}
	return b.String()
}
