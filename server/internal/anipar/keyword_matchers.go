package anipar

import (
	"strings"
)

// MARK: 匹配逻辑

var (
	aiResolutionRE    = jsRegex(`^(AI)(\d{3,4}[Pp])$`, true)
	resAtFpsRE        = jsRegex(`^(\d{3,5}(?:[Pp]|[Xx×]\d{3,5}))@(\d+(?:\.\d+)?FPS)$`, true)
	resFrameModeRE    = jsRegex(`^(\d{3,4}P)(高帧率)$`, true)
	resPOrDimsRE      = jsRegex(`^\d{3,4}P$`, true)
	resDimsRE         = jsRegex(`^\d{3,5}[Xx×]\d{3,5}$`, false)
	tmdbidRE          = jsRegex(`^tmdbid=(.+)$`, false)
	monogatariRE      = jsRegex(`^.+(物语|物語)$`, false)
	yearMonthRE       = jsRegex(`^(\d\d\d\d)年(\d\d?)月新?番$`, false)
	starMonthRE       = jsRegex(`^★?(\d{1,2})月新?番★?$`, false)
	dateRE            = jsRegex(`^(\d\d\d\d)\.(\d?\d)\.(\d?\d)$`, false)
	yearSPRE          = jsRegex(`^(\d\d\d\d)(SP)$`, false)
	versionNumberRE   = jsRegex(`^[vV](\d+)$`, false)
	volNumberRE       = jsRegex(`^(?:Vol|vol|Volume|volume)\.?\s*(\d+)$`, false)
	prefixStarMonthRE = jsRegex(`^★?(\d\d?)月新?番★?`, false)
	prefixMovieRE     = jsRegex(`^★?(剧场版|劇場版)★?`, false)
	prefixOldRE       = jsRegex(`^★?(老番)★?`, false)
	singleNumberRE    = jsRegex(`^\d+$`, false)
)

// matchSingleTag mirrors matchSingleTag in keyword.ts.
func matchSingleTag(ctx *Context, text string) bool {
	text = jsTrim(text)

	upper := strings.ToUpper(text)

	// Match keywords
	{
		info, ok := audioCompoundTerms[upper]
		if ok {
			if info.codec != "" {
				ctx.update3("file", "audio", "codec", info.codec)
			}
			if info.channels != "" {
				ctx.update3("file", "audio", "channels", info.channels)
			}
			if info.trackCount != 0 {
				ctx.update3("file", "audio", "trackCount", info.trackCount)
			}
			return true
		}
	}
	if audioChannels[upper] {
		ctx.update3("file", "audio", "channels", text)
		return true
	}
	if audioCodecs[upper] {
		ctx.update3("file", "audio", "codec", text)
		return true
	}
	if audioLanguages[upper] {
		ctx.update3("file", "audio", "language", text)
		return true
	}
	{
		if info, ok := videoCompoundTerms[upper]; ok {
			if info.codec != "" {
				ctx.update3("file", "video", "codec", info.codec)
			}
			if info.bitDepth != "" {
				ctx.update3("file", "video", "bitDepth", info.bitDepth)
			}
			if info.resolution != "" {
				ctx.update3("file", "video", "resolution", info.resolution)
			}
			if info.audioCodec != "" {
				ctx.update3("file", "audio", "codec", info.audioCodec)
			}
			return true
		}
	}
	if videoCodecs[upper] {
		ctx.update3("file", "video", "codec", text)
		return true
	}
	if videoBitDepths[upper] {
		ctx.update3("file", "video", "bitDepth", text)
		return true
	}
	if videoFormats[upper] {
		ctx.update3("file", "video", "format", text)
		return true
	}
	if videoQualities[upper] {
		ctx.update3("file", "video", "quality", text)
		return true
	}
	if videoResolutionTerms[upper] {
		ctx.update3("file", "video", "resolution", text)
		return true
	}
	if videoFrameRates[upper] {
		ctx.update3("file", "video", "fps", text)
		return true
	}
	{
		res := aiResolutionRE.FindStringSubmatch(text)
		if res != nil {
			ctx.update3("file", "video", "enhancement", res[1])
			ctx.update3("file", "video", "resolution", res[2])
			return true
		}
	}
	{
		res := resAtFpsRE.FindStringSubmatch(text)
		if res != nil {
			ctx.update3("file", "video", "resolution", res[1])
			ctx.update3("file", "video", "fps", res[2])
			return true
		}
	}
	{
		res := resFrameModeRE.FindStringSubmatch(text)
		if res != nil {
			ctx.update3("file", "video", "resolution", res[1])
			ctx.update3("file", "video", "frameRateMode", res[2])
			return true
		}
	}
	if resPOrDimsRE.MatchString(text) || resDimsRE.MatchString(text) {
		ctx.update3("file", "video", "resolution", text)
		return true
	}
	if videoResolutions[upper] {
		ctx.update3("file", "video", "resolution", text)
		return true
	}
	if sourceSet[upper] {
		ctx.update("source", text)
		return true
	}
	if platforms[text] {
		ctx.update("platform", text)
		return true
	}
	if typesSet[upper] {
		ctx.update("type", text)
		return true
	}
	if variantsSet[text] {
		ctx.update("variants", append([]string{}, append(ctx.result.Variants, text)...))
		return true
	}
	if FileExtensions[upper] {
		ctx.update2("file", "extension", text)
		return true
	}
	if otherTags[text] {
		ctx.tags = append(ctx.tags, jsTrim(text))
		return true
	}
	if strings.HasSuffix(text, ".ver") {
		ctx.tags = append(ctx.tags, jsTrim(text))
		return true
	}
	if strings.HasPrefix(text, "Bloomy_Cafe") {
		ctx.tags = append(ctx.tags, jsTrim(text))
		return true
	}
	if ignores[text] {
		return true
	}

	// tmdbid=1406607
	{
		res := tmdbidRE.FindStringSubmatch(text)
		if res != nil {
			ctx.update("tmdbId", res[1])
			return true
		}
	}

	// 抚物语 ...
	{
		res := monogatariRE.FindStringSubmatch(text)
		if res != nil {
			ctx.tags = append(ctx.tags, text)
			return true
		}
	}

	// Match language and subtitles
	{
		appendSubtitleLanguage := func(language string) {
			languages := append([]string{}, ctx.result.SubtitleLanguages()...)
			languages = append(languages, language)
			ctx.update2("subtitle", "languages", languages)
		}

		updateSubtitleFormat := func(format string, overwrite bool) {
			if format != "" && (overwrite || ctx.result.SubtitleFormat() == "") {
				ctx.update2("subtitle", "format", format)
			}
		}

		updateSubtitleEncoding := func(encoding string) {
			if encoding != "" && ctx.result.SubtitleEncoding() == "" && ctx.result.SubtitleEncodings() == nil {
				ctx.update2("subtitle", "encoding", encoding)
			}
		}

		updateSubtitleEncodings := func(encodings []string) {
			if encodings == nil {
				return
			}
			values := append([]string{}, ctx.result.SubtitleEncodings()...)
			values = append(values, encodings...)
			ctx.update2("subtitle", "encodings", values)
			if ctx.result.SubtitleEncoding() != "" {
				ctx.clearSubtitleEncoding()
			}
		}

		language, ok := subtitleLanguageTerms[upper]
		if !ok {
			language, ok = subtitleLanguageTerms[text]
		}
		if ok {
			appendSubtitleLanguage(language)
			return true
		}

		if format, ok := subtitleFormatTerms[upper]; ok {
			updateSubtitleFormat(format, false)
			return true
		}

		encodingInfo, ok := subtitleEncodingTerms[upper]
		if !ok {
			encodingInfo, ok = subtitleEncodingTerms[text]
		}
		if ok {
			updateSubtitleFormat(encodingInfo.format, true)
			updateSubtitleEncoding(encodingInfo.encoding)
			updateSubtitleEncodings(encodingInfo.encodings)
			return true
		}

		if languageWithFormat, ok := languageSubtitleFormatTerms[text]; ok {
			appendSubtitleLanguage(languageWithFormat.language)
			updateSubtitleFormat(languageWithFormat.format, false)
			return true
		}

		if platformLanguage, ok := platformLanguageTerms[text]; ok {
			ctx.update("platform", platformLanguage[0])
			appendSubtitleLanguage(platformLanguage[1])
			return true
		}

		for _, prefix := range subtitleLanguagePrefixes {
			if strings.HasPrefix(text, prefix) {
				suffix := text[len(prefix):]
				suffixFormat, known := subtitleFormatSuffixTerms[suffix]
				if suffix == "" || known {
					appendSubtitleLanguage(prefix)
					if suffix != "" {
						updateSubtitleFormat(suffixFormat, false)
					}
					return true
				}
			}
		}
	}

	// Match regex
	{
		{
			// 2024年10月番
			match := yearMonthRE.FindStringSubmatch(text)
			if match != nil {
				if year, ok := strToInt(match[1]); ok && 1949 <= year && year <= 2099 {
					ctx.update("year", year)
				}
				if month, ok := strToInt(match[2]); ok && 1 <= month && month <= 12 {
					ctx.update("month", month)
				}
				return true
			}
		}
		{
			// ★10月新番 ★04月新番★
			match := starMonthRE.FindStringSubmatch(text)
			if match != nil {
				if month, ok := strToInt(match[1]); ok && 1 <= month && month <= 12 {
					ctx.update("month", month)
				}
				return true
			}
		}
		{
			// [2024.12.15]
			match := dateRE.FindStringSubmatch(text)
			if match != nil {
				if year, ok := strToInt(match[1]); ok && 1949 <= year && year <= 2099 {
					ctx.update("year", year)
				}
				if month, ok := strToInt(match[2]); ok && 1 <= month && month <= 12 {
					ctx.update("month", month)
				}
				return true
			}
		}
		{
			// [2024SP]
			match := yearSPRE.FindStringSubmatch(text)
			if match != nil {
				if year, ok := strToInt(match[1]); ok && 1949 <= year && year <= 2099 {
					ctx.update("year", year)
				}
				ctx.update("type", match[2])
				return true
			}
		}
		{
			// v2
			match := versionNumberRE.FindStringSubmatch(text)
			if match != nil {
				if version, ok := strToInt(match[1]); ok {
					ctx.update("version", version)
				}
				return true
			}
		}
	}

	// Match prefix
	{
		for _, prefix := range searchPrefix {
			if strings.HasPrefix(text, prefix) {
				title := jsTrim(text[len(prefix):])
				parts := strings.Split(title, "/")
				var search []string
				for _, t := range parts {
					t = jsTrim(t)
					if t != "" {
						search = append(search, t)
					}
				}
				ctx.update("search", search)
				return true
			}
		}
		for _, prefix := range hiringPrefix {
			if strings.HasPrefix(text, prefix) {
				return true
			}
		}
		for _, prefix := range otherPrefix {
			if strings.HasPrefix(text, prefix) {
				ctx.tags = append(ctx.tags, jsTrim(text))
				return true
			}
		}
	}

	// Match vol.1
	{
		res := volNumberRE.FindStringSubmatch(text)
		if res != nil {
			if vol, ok := strToInt(res[1]); ok {
				ctx.update2("volume", "number", vol)
				return true
			}
		}
	}

	return false
}

// matchMultipleTags mirrors matchMultipleTags in keyword.ts.
func matchMultipleTags(ctx *Context, text string, separators ...string) bool {
	if len(separators) == 0 {
		separators = []string{" ", "_", "&", "+"}
	}
	for _, sep := range separators {
		parts := strings.Split(text, sep)
		if len(parts) <= 1 {
			continue
		}
		matched := true
		for _, part := range parts {
			if !matchSingleTag(ctx, part) {
				matched = false
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// parseWrappedTag mirrors parseWrappedTag in keyword.ts.
func parseWrappedTag(ctx *Context, token *Token) bool {
	if token.IsWrapped() {
		text := token.text
		if matchSingleTag(ctx, text) {
			return true
		}
		if matchEpiodes(ctx, text) {
			return true
		}
		if matchMultipleTags(ctx, text) {
			return true
		}
	}
	return false
}

// parsePrefixWrappedTags mirrors parsePrefixWrappedTags.
func parsePrefixWrappedTags(ctx *Context) {
	for ctx.left < ctx.right {
		if parseWrappedTag(ctx, ctx.tokens[ctx.left]) {
			ctx.left++
		} else if jsTrim(ctx.tokens[ctx.left].text) == "" {
			ctx.left++
		} else {
			break
		}
	}
}

// parseSuffixWrappedTags mirrors parseSuffixWrappedTags.
func parseSuffixWrappedTags(ctx *Context) {
	for ctx.left < ctx.right {
		if parseWrappedTag(ctx, ctx.tokens[ctx.right]) {
			ctx.right--
		} else if jsTrim(ctx.tokens[ctx.right].text) == "" {
			ctx.right--
		} else {
			// Unknown tags
			isIgnoreLast := func() bool {
				token := ctx.tokens[len(ctx.tokens)-1]
				for _, prefix := range searchPrefix {
					if strings.HasPrefix(token.text, prefix) {
						return true
					}
				}
				for _, prefix := range hiringPrefix {
					if strings.HasPrefix(token.text, prefix) {
						return true
					}
				}
				if ignores[token.text] {
					return true
				}
				return false
			}

			if (ctx.left+2 < ctx.right && ctx.right == len(ctx.tokens)-1) ||
				(ctx.right == len(ctx.tokens)-2 && isIgnoreLast()) {
				ctx.tags = append(ctx.tags, jsTrim(ctx.tokens[ctx.right].text))
				ctx.right--
			} else {
				break
			}
		}
	}
}

// parsePrefixTextTags mirrors parsePrefixTextTags.
func parsePrefixTextTags(ctx *Context) bool {
	if ctx.left > ctx.right {
		return false
	}

	token := ctx.tokens[ctx.left]
	trimmed := parsePrefixTextInlineTags(ctx, token.text)

	if trimmed != token.text {
		if trimmed != "" {
			ctx.tokens[ctx.left] = NewToken(trimmed, token.left, token.right)
		} else {
			ctx.left++
		}
		return true
	}

	return false
}

// parsePrefixTextInlineTags mirrors parsePrefixTextInlineTags.
func parsePrefixTextInlineTags(ctx *Context, text string) string {
	text = jsTrim(text)
	{
		// ★10月新番
		match := prefixStarMonthRE.FindStringSubmatch(text)
		if match != nil {
			matched := match[0]
			if month, ok := strToInt(match[1]); ok {
				ctx.update("month", month)
			}
			text = text[len(matched):]
		}
	}
	{
		// ★剧场版★
		match := prefixMovieRE.FindStringSubmatch(text)
		if match != nil {
			matched := match[0]
			ctx.update("type", match[1])
			text = text[len(matched):]
		}
	}
	{
		// ★老番★
		match := prefixOldRE.FindStringSubmatch(text)
		if match != nil {
			matched := match[0]
			text = text[len(matched):]
		}
	}
	return text
}

// parseSuffixTextInlineTags mirrors parseSuffixTextInlineTags.
func parseSuffixTextInlineTags(ctx *Context, text string) string {
	changed := 0

	tokens := tokenize(text)

	for len(tokens) > 1 {
		token := tokens[len(tokens)-1].Trim()
		if parseWrappedTag(ctx, token) {
			changed++
			tokens = tokens[:len(tokens)-1]
		} else {
			break
		}
	}

	if changed > 0 {
		var b strings.Builder
		for _, t := range tokens {
			b.WriteString(t.String())
		}
		return b.String()
	}

	return text
}

// parseSuffixTextInlineMultipleTags mirrors parseSuffixTextInlineMultipleTags.
func parseSuffixTextInlineMultipleTags(ctx *Context, text string, separators ...string) string {
	if len(separators) == 0 {
		separators = []string{" ", "★"}
	}
	text = jsTrim(text)
	for _, sep := range separators {
		parts := strings.Split(text, sep)
		if len(parts) > 1 {
			changed := 0
			for len(parts) > 1 {
				part := parts[len(parts)-1]
				// Skip single number
				if singleNumberRE.MatchString(part) {
					break
				}
				if matchSingleTag(ctx, part) || matchEpiodes(ctx, part) || matchMultipleTags(ctx, part) {
					changed++
					parts = parts[:len(parts)-1]
				} else {
					break
				}
			}
			if changed > 1 {
				text = strings.Join(parts, sep)
			}
			if changed > 0 {
				break
			}
		}
	}
	return text
}
