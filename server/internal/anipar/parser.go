package anipar

// parsePipeline is the fansub-specific pipeline function type.
type parsePipeline func(ctx *Context) *ParseResult

var parsers = map[string]parsePipeline{
	FansubKiraraFantasia: func(ctx *Context) *ParseResult {
		parseFansub(ctx)
		parseSuffixWrappedTags(ctx)
		parseSuffixEpisodes(ctx)
		parseMultipleTitles(ctx, defaultSplitOptions())
		return ctx.normalize()
	},
	FansubANi: func(ctx *Context) *ParseResult {
		parseFansub(ctx)
		parseSuffixWrappedTags(ctx)
		parseSuffixEpisodes(ctx)

		titles := parseMultipleTitles(ctx, defaultSplitOptions())
		if len(titles) == 2 {
			ctx.update("title", titles[1])
			ctx.update("titles", []string{titles[0]})
		}

		return ctx.normalize()
	},
	FansubLoliHouse: func(ctx *Context) *ParseResult {
		parseFansub(ctx)
		parseSuffixWrappedTags(ctx)
		parseSuffixEpisodes(ctx)
		parseMultipleTitles(ctx, defaultSplitOptions().withSpace(false).withSeparators([]string{"/"}))

		// Postprocess

		if ctx.result.Titles != nil && ctx.result.EpisodesRange != nil {
			title := ctx.result.Titles[len(ctx.result.Titles)-1]
			if stringsHasSuffix(title, " -") {
				ctx.result.Titles[len(ctx.result.Titles)-1] = title[:len(title)-2]
			}
		}

		// 在地下城寻求邂逅是否搞错了什么2 + season 2 -> 在地下城寻求邂逅是否搞错了什么
		if ctx.result.Title != "" && ctx.result.Season != nil && ctx.result.Season.Number != 0 {
			title := ctx.result.Title
			season := ctx.result.Season.Number
			runes := []rune(title)
			if len(runes) >= 2 {
				lastDigit := runes[len(runes)-1] >= '0' && runes[len(runes)-1] <= '9'
				if lastDigit && int(runes[len(runes)-1]-'0') == season && !isDigit(runes[len(runes)-2]) {
					seasonStr := itoa(season)
					ctx.result.Title = string(runes[:len(runes)-len([]rune(seasonStr))])
				}
			}
		}

		// v-tags on LoliHouse
		for _, tag := range ctx.tags {
			res := versionNumberRE.FindStringSubmatch(tag)
			if res != nil {
				if version, ok := strToInt(res[1]); ok {
					ctx.update("version", version)
				}
			}
		}

		return ctx.normalize()
	},
	Fansub绿茶字幕组: func(ctx *Context) *ParseResult {
		parseFansub(ctx)
		parseSuffixWrappedTags(ctx)
		parseSuffixEpisodes(ctx)
		parseMultipleTitles(ctx, defaultSplitOptions().withSpace(false).withSeparators([]string{"/"}))
		return ctx.normalize()
	},
	Fansub桜都字幕组: func(ctx *Context) *ParseResult {
		parseFansub(ctx)
		parseSuffixWrappedTags(ctx)
		parseSuffixEpisodes(ctx)
		parseMultipleTitles(ctx, defaultSplitOptions().withSpace(false))
		return ctx.normalize()
	},
	FansubPrejudiceStudio: func(ctx *Context) *ParseResult {
		parseFansub(ctx)
		parseSuffixWrappedTags(ctx)
		parseSuffixEpisodes(ctx)
		parseMultipleTitles(ctx, defaultSplitOptions().withSpace(false).withSeparators([]string{"/"}))
		return ctx.normalize()
	},
	Fansub喵萌奶茶屋: func(ctx *Context) *ParseResult {
		parseFansub(ctx)
		parsePrefixTextTags(ctx)
		parseSuffixWrappedTags(ctx)
		parseSuffixEpisodes(ctx)
		parseMultipleTitles(ctx, defaultSplitOptions().withSpace(false).withSeparators([]string{"/"}))
		return ctx.normalize()
	},
	Fansub雪飄工作室: func(ctx *Context) *ParseResult {
		parseFansub(ctx)
		parsePrefixWrappedTags(ctx)
		parseSuffixWrappedTags(ctx)
		parseSuffixEpisodes(ctx)
		parseMultipleTitles(ctx, defaultSplitOptions().withSpace(false).withSeparators([]string{"/"}))
		return ctx.normalize()
	},
	Fansub三明治摆烂组: func(ctx *Context) *ParseResult {
		parseFansub(ctx)
		parseSuffixWrappedTags(ctx)

		// hack: [三明治摆烂组&Pre-S] ... - 06.5 总集篇(S00E01) - [繁日内嵌]...
		if ctx.left <= ctx.right && ctx.tokens[ctx.right].text == " - " && !ctx.tokens[ctx.right].IsWrapped() {
			ctx.right--
		}

		parseSuffixEpisodes(ctx)

		// hack2: restore episode/season after a 总集篇 suffix
		if ctx.left <= ctx.right && ctx.result.Episode != nil && ctx.result.Season != nil &&
			stringsHasSuffix(ctx.tokens[ctx.right].text, " 总集篇") {
			episode := ctx.result.Episode
			season := ctx.result.Season
			ctx.result.Episode = nil
			parseSuffixEpisodes(ctx)
			ctx.result.Episode = episode
			ctx.result.Season = season
		}

		parseMultipleTitles(ctx, defaultSplitOptions())

		return ctx.normalize()
	},
}

// Parse mirrors parse() in parser.ts. fansub corresponds to options.fansub.
func Parse(title string, fansub string) *ParseResult {
	if title == "" {
		return nil
	}

	fileTitle, extension := parseFileExtension(title)

	tokens := tokenize(fileTitle)
	if len(tokens) == 0 {
		return nil
	}

	ctx := NewContext(tokens, fansub)
	if extension != "" {
		ctx.update2("file", "extension", extension)
	}

	parser := parsers[fansub]

	// Use pre-defined parser
	if parser != nil {
		return parser(ctx)
	}

	// Fallback to default parser

	// 1. Parse fansub
	parseFansub(ctx)
	if fansub == "" {
		fansub = ctx.result.FansubName()
	}
	if fansub == "" {
		return nil
	}

	if parsers[fansub] != nil {
		// Re-run with parser
		return Parse(title, fansub)
	}

	// Parse left tags
	parsePrefixWrappedTags(ctx)
	parsePrefixTextTags(ctx)

	// 2. Parse right tags
	parseSuffixWrappedTags(ctx)
	parseSuffixEpisodes(ctx)

	// 3. Parse title
	titles := parseMultipleTitles(ctx, defaultSplitOptions())
	if len(titles) == 0 {
		return nil
	}

	// 4. Postprocess
	return ctx.normalize()
}

func (r *ParseResult) FansubName() string {
	if r.Fansub == nil {
		return ""
	}
	return r.Fansub.Name
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
